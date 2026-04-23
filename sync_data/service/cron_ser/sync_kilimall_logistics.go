package cron_ser

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm/clause"
)

var (
	ErrKilimallAuthExpired = errors.New("kilimall auth expired")
	kilimallRand           = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const kilimallServiceStatusID = 13

type KilimallLogisticsRecord struct {
	ID              string
	OrderID         string
	OrderNumber     string
	TrackingNumber  string
	TrackingURL     string
	Status          string
	ShipmentType    string
	DeliveryOption  string
	ShopID          string
	CountryCode     string
	CountryName     string
	CountryCurrency string
	ProductName     string
	SellerSKU       string
	ImageURL        string
	ItemPrice       float64
	PaidPrice       float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type KilimallParentOrder struct {
	ID                       string
	ShopIDs                  string
	Status                   string
	Number                   string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	TotalAmountLocalCurrency string
	TotalAmountLocalValue    float64
	CountryCode              string
	CountryName              string
	CountryCurrency          string
}

type kilimallOrderListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total  int             `json:"total"`
		Orders []kilimallOrder `json:"orders"`
	} `json:"data"`
}

type kilimallOrder struct {
	OrderID             int64         `json:"orderId"`
	OrderSn             string        `json:"orderSn"`
	RegionCode          string        `json:"regionCode"`
	StoreID             int64         `json:"storeId"`
	CreatedTime         string        `json:"createdTime"`
	PaidTime            string        `json:"paidTime"`
	ConfirmedTime       string        `json:"confirmedTime"`
	ShippedTime         string        `json:"shippedTime"`
	DeliveredTime       string        `json:"deliveredTime"`
	CompletedTime       string        `json:"completedTime"`
	CancelledTime       string        `json:"cancelledTime"`
	RejectedTime        string        `json:"rejectedTime"`
	OrderStatus         int           `json:"orderStatus"`
	OrderTrackingNumber string        `json:"orderTrackingNumber"`
	LogisticsNumber     string        `json:"LogisticsNumber"`
	DeliveryType        interface{}   `json:"deliveryType"`
	PayAmount           float64       `json:"payAmount"`
	Skus                []kilimallSKU `json:"skus"`
}

type kilimallSKU struct {
	ID         int64   `json:"id"`
	Title      string  `json:"title"`
	Spec       string  `json:"spec"`
	IDByVendor string  `json:"idByVendor"`
	SkuID      int64   `json:"skuId"`
	ImgURL     string  `json:"imgUrl"`
	DealPrice  float64 `json:"dealPrice"`
	SalePrice  float64 `json:"salePrice"`
	Amount     float64 `json:"amount"`
}

func SyncKilimallLogistics() {
	global.Log.Info("开始同步 Kilimall 物流数据")

	cookie := strings.TrimSpace(global.Config.Kilimall.Cookie)
	authToken := strings.TrimSpace(global.Config.Kilimall.AuthToken)
	if cookie == "" && authToken == "" {
		updateKilimallServiceStatus("错误", "Kilimall 鉴权为空，请在 settings.yaml 中填写 kilimall.cookie 或 kilimall.auth_token")
		return
	}

	records, parentOrders, err := FetchKilimallLogistics(cookie, authToken)
	if err != nil {
		if errors.Is(err, ErrKilimallAuthExpired) {
			updateKilimallServiceStatus("错误", "Kilimall Cookie/Token 已过期或被拦截，请手动更新 settings.yaml")
			return
		}
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 同步失败: %v", err))
		return
	}

	cleaned := cleanKilimallRecords(records)
	if len(cleaned) == 0 {
		updateKilimallServiceStatus("错误", "Kilimall 返回为空或清洗后无有效数据")
		return
	}

	if err := upsertKilimallOrders(parentOrders); err != nil {
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 父订单入库失败: %v", err))
		return
	}

	if err := upsertKilimallLogistics(cleaned); err != nil {
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 入库失败: %v", err))
		return
	}

	updateKilimallServiceStatus("正常", fmt.Sprintf("Kilimall 物流同步成功，记录数: %d", len(cleaned)))
}

func FetchKilimallLogistics(cookie, authToken string) ([]KilimallLogisticsRecord, []KilimallParentOrder, error) {
	baseURL := strings.TrimSpace(global.Config.Kilimall.BaseURL)
	if baseURL == "" {
		baseURL = "https://seller-api.kilimall.ke"
	}
	apiPath := strings.TrimSpace(global.Config.Kilimall.LogisticsAPI)
	if apiPath == "" {
		apiPath = "/order-list"
	}
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}

	pageSize := global.Config.Kilimall.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	maxRetries := global.Config.Kilimall.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	delay := time.Duration(global.Config.Kilimall.DelayMs) * time.Millisecond
	if delay <= 0 {
		delay = 1200 * time.Millisecond
	}

	client := &http.Client{Timeout: 40 * time.Second}
	allRecords := make([]KilimallLogisticsRecord, 0)
	allParentOrders := make([]KilimallParentOrder, 0)

	for page := 1; ; page++ {
		time.Sleep(delayWithJitter(delay))

		var (
			records      []KilimallLogisticsRecord
			parentOrders []KilimallParentOrder
			hasNext      bool
			err          error
		)

		for attempt := 1; attempt <= maxRetries; attempt++ {
			records, parentOrders, hasNext, err = fetchKilimallPage(client, baseURL, apiPath, page, pageSize, cookie, authToken)
			if err == nil {
				break
			}
			if errors.Is(err, ErrKilimallAuthExpired) {
				return nil, nil, err
			}
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * time.Second
				global.Log.Warnf("Kilimall 第 %d 页第 %d 次请求失败: %v，%v 后重试", page, attempt, err, backoff)
				time.Sleep(backoff)
			}
		}
		if err != nil {
			return nil, nil, err
		}

		allRecords = append(allRecords, records...)
		allParentOrders = append(allParentOrders, parentOrders...)
		if !hasNext || len(records) == 0 {
			break
		}
	}

	return allRecords, dedupKilimallParentOrders(allParentOrders), nil
}

func fetchKilimallPage(client *http.Client, baseURL, apiPath string, page, pageSize int, cookie, authToken string) ([]KilimallLogisticsRecord, []KilimallParentOrder, bool, error) {
	u, err := buildKilimallURL(baseURL, apiPath, page, pageSize)
	if err != nil {
		return nil, nil, false, err
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, false, err
	}
	setKilimallHeaders(req, baseURL, cookie, authToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, false, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, nil, false, ErrKilimallAuthExpired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, false, fmt.Errorf("kilimall http status=%d body=%.400s", resp.StatusCode, string(body))
	}

	var result kilimallOrderListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, false, fmt.Errorf("kilimall json parse failed: %w body=%.400s", err, string(body))
	}
	if result.Code != 0 && result.Code != 200 {
		msg := strings.ToLower(result.Msg)
		if strings.Contains(msg, "auth") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "unauthorized") {
			return nil, nil, false, ErrKilimallAuthExpired
		}
		return nil, nil, false, fmt.Errorf("kilimall api error code=%d msg=%s", result.Code, result.Msg)
	}

	records := flattenKilimallOrders(result.Data.Orders)
	parentOrders := mapKilimallParentOrders(result.Data.Orders)
	hasNext := page*pageSize < result.Data.Total
	return records, parentOrders, hasNext, nil
}

func flattenKilimallOrders(orders []kilimallOrder) []KilimallLogisticsRecord {
	records := make([]KilimallLogisticsRecord, 0)
	for _, order := range orders {
		orderID := strconv.FormatInt(order.OrderID, 10)
		shopID := strconv.FormatInt(order.StoreID, 10)
		trackingNumber := firstNonEmpty(strings.TrimSpace(order.OrderTrackingNumber), strings.TrimSpace(order.LogisticsNumber))
		status := mapKilimallOrderStatus(order)
		createdAt := parseKilimallTime(order.CreatedTime)
		updatedAt := pickLatestTime(order)
		deliveryOption := strings.TrimSpace(fmt.Sprintf("%v", order.DeliveryType))

		if len(order.Skus) == 0 {
			records = append(records, KilimallLogisticsRecord{
				ID:             buildKilimallSyntheticID(orderID, order.OrderSn, trackingNumber, ""),
				OrderID:        orderID,
				OrderNumber:    strings.TrimSpace(order.OrderSn),
				TrackingNumber: trackingNumber,
				Status:         status,
				DeliveryOption: deliveryOption,
				ShopID:         shopID,
				CountryCode:    strings.TrimSpace(order.RegionCode),
				CountryName:    mapRegionToCountry(order.RegionCode),
				PaidPrice:      order.PayAmount,
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			})
			continue
		}

		for _, sku := range order.Skus {
			sellerSKU := strings.TrimSpace(sku.IDByVendor)
			if sellerSKU == "" && sku.SkuID > 0 {
				sellerSKU = strconv.FormatInt(sku.SkuID, 10)
			}
			price := sku.Amount
			if price <= 0 {
				if sku.SalePrice > 0 {
					price = sku.SalePrice
				} else {
					price = sku.DealPrice
				}
			}
			productName := strings.TrimSpace(sku.Title)
			if strings.TrimSpace(sku.Spec) != "" {
				productName = strings.TrimSpace(productName + " " + sku.Spec)
			}
			records = append(records, KilimallLogisticsRecord{
				ID:              buildKilimallSyntheticID(orderID, order.OrderSn, trackingNumber, strconv.FormatInt(sku.ID, 10)),
				OrderID:         orderID,
				OrderNumber:     strings.TrimSpace(order.OrderSn),
				TrackingNumber:  trackingNumber,
				Status:          status,
				DeliveryOption:  deliveryOption,
				ShopID:          shopID,
				CountryCode:     strings.TrimSpace(order.RegionCode),
				CountryName:     mapRegionToCountry(order.RegionCode),
				ProductName:     productName,
				SellerSKU:       sellerSKU,
				ImageURL:        strings.TrimSpace(sku.ImgURL),
				ItemPrice:       price,
				PaidPrice:       price,
				CreatedAt:       createdAt,
				UpdatedAt:       updatedAt,
				CountryCurrency: mapRegionToCurrency(order.RegionCode),
			})
		}
	}
	return records
}

func buildKilimallURL(baseURL, apiPath string, page, pageSize int) (*url.URL, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + apiPath)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(pageSize))
	q.Set("pagination", strconv.Itoa(page))

	orderStatus := global.Config.Kilimall.OrderStatus
	if orderStatus < 0 {
		orderStatus = 0
	}
	q.Set("orderStatus", strconv.Itoa(orderStatus))

	returnSkus := global.Config.Kilimall.ReturnSkus
	if returnSkus <= 0 {
		returnSkus = 1
	}
	q.Set("returnSkus", strconv.Itoa(returnSkus))

	regionID := global.Config.Kilimall.RegionID
	if regionID <= 0 {
		regionID = 6
	}
	q.Set("regionId", strconv.Itoa(regionID))

	regionCode := strings.TrimSpace(global.Config.Kilimall.RegionCode)
	if regionCode == "" {
		regionCode = "KE"
	}
	q.Set("regionCode", regionCode)

	timeType := global.Config.Kilimall.TimeType
	if timeType <= 0 {
		timeType = 1
	}
	q.Set("timeType", strconv.Itoa(timeType))

	startTime, endTime := kilimallTimeRange()
	q.Set("startTime", startTime)
	q.Set("endTime", endTime)

	u.RawQuery = q.Encode()
	return u, nil
}

func setKilimallHeaders(req *http.Request, baseURL, cookie, authToken string) {
	origin := kilimallOriginFromAPI(baseURL)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,ms;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	req.Header.Set("kili-language", "zh")
	req.Header.Set("request-nonce", strconv.FormatInt(time.Now().UnixMilli(), 10))

	if authToken != "" {
		req.Header.Set("accesstoken", authToken)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
}

func upsertKilimallLogistics(records []KilimallLogisticsRecord) error {
	items := make([]models.OrderItem, 0, len(records))
	for _, rec := range records {
		if rec.ID == "" {
			continue
		}
		items = append(items, models.OrderItem{
			ID:              rec.ID,
			OrderID:         rec.OrderID,
			OrderNumber:     rec.OrderNumber,
			TrackingNumber:  rec.TrackingNumber,
			TrackingURL:     rec.TrackingURL,
			Status:          rec.Status,
			ShipmentType:    rec.ShipmentType,
			DeliveryOption:  rec.DeliveryOption,
			ShopID:          rec.ShopID,
			CountryCode:     rec.CountryCode,
			CountryName:     rec.CountryName,
			CountryCurrency: rec.CountryCurrency,
			ProductName:     rec.ProductName,
			SellerSKU:       rec.SellerSKU,
			ImageURL:        rec.ImageURL,
			ItemPrice:       rec.ItemPrice,
			PaidPrice:       rec.PaidPrice,
			CreatedAt:       rec.CreatedAt,
			UpdatedAt:       rec.UpdatedAt,
		})
	}

	if len(items) == 0 {
		return nil
	}

	const batchSize = 300
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		if err := global.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"order_id", "order_number", "tracking_number", "tracking_url", "status",
				"shipment_type", "delivery_option", "shop_id", "country_code", "country_name",
				"country_currency", "product_name", "seller_sku", "image_url",
				"item_price", "paid_price", "created_at", "updated_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}

	return nil
}

func upsertKilimallOrders(parentOrders []KilimallParentOrder) error {
	if len(parentOrders) == 0 {
		return nil
	}

	orders := make([]models.Order, 0, len(parentOrders))
	for _, o := range parentOrders {
		if o.ID == "" {
			continue
		}
		orders = append(orders, models.Order{
			ID:                       o.ID,
			ShopIDs:                  o.ShopIDs,
			Status:                   o.Status,
			Number:                   o.Number,
			CreatedAt:                o.CreatedAt,
			UpdatedAt:                o.UpdatedAt,
			TotalAmountLocalCurrency: o.TotalAmountLocalCurrency,
			TotalAmountLocalValue:    o.TotalAmountLocalValue,
			CountryCode:              o.CountryCode,
			CountryName:              o.CountryName,
			CountryCurrency:          o.CountryCurrency,
		})
	}

	const batchSize = 300
	for i := 0; i < len(orders); i += batchSize {
		end := i + batchSize
		if end > len(orders) {
			end = len(orders)
		}
		batch := orders[i:end]

		if err := global.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"shop_ids", "status", "number", "created_at", "updated_at",
				"total_amount_local_currency", "total_amount_local_value",
				"country_code", "country_name", "country_currency",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	return nil
}

func mapKilimallParentOrders(orders []kilimallOrder) []KilimallParentOrder {
	mapped := make([]KilimallParentOrder, 0, len(orders))
	for _, order := range orders {
		orderID := strconv.FormatInt(order.OrderID, 10)
		if orderID == "0" || orderID == "" {
			continue
		}
		countryCode := strings.TrimSpace(order.RegionCode)
		mapped = append(mapped, KilimallParentOrder{
			ID:                       orderID,
			ShopIDs:                  strconv.FormatInt(order.StoreID, 10),
			Status:                   mapKilimallOrderStatus(order),
			Number:                   strings.TrimSpace(order.OrderSn),
			CreatedAt:                firstNonZeroTime(parseKilimallTime(order.CreatedTime), time.Now()),
			UpdatedAt:                firstNonZeroTime(pickLatestTime(order), time.Now()),
			TotalAmountLocalCurrency: mapRegionToCurrency(countryCode),
			TotalAmountLocalValue:    order.PayAmount,
			CountryCode:              countryCode,
			CountryName:              mapRegionToCountry(countryCode),
			CountryCurrency:          mapRegionToCurrency(countryCode),
		})
	}
	return mapped
}

func dedupKilimallParentOrders(orders []KilimallParentOrder) []KilimallParentOrder {
	m := make(map[string]KilimallParentOrder, len(orders))
	for _, o := range orders {
		if o.ID == "" {
			continue
		}
		existing, ok := m[o.ID]
		if !ok || o.UpdatedAt.After(existing.UpdatedAt) {
			m[o.ID] = o
		}
	}
	result := make([]KilimallParentOrder, 0, len(m))
	for _, o := range m {
		result = append(result, o)
	}
	return result
}

func cleanKilimallRecords(records []KilimallLogisticsRecord) []KilimallLogisticsRecord {
	dedup := make(map[string]KilimallLogisticsRecord, len(records))
	for _, rec := range records {
		if rec.ID == "" {
			continue
		}
		if rec.OrderID == "" && rec.OrderNumber == "" {
			continue
		}
		rec.Status = normalizeKilimallStatus(rec.Status)
		existing, ok := dedup[rec.ID]
		if !ok || rec.UpdatedAt.After(existing.UpdatedAt) {
			dedup[rec.ID] = rec
		}
	}

	cleaned := make([]KilimallLogisticsRecord, 0, len(dedup))
	for _, rec := range dedup {
		cleaned = append(cleaned, rec)
	}
	return cleaned
}

func mapKilimallOrderStatus(order kilimallOrder) string {
	if strings.TrimSpace(order.CancelledTime) != "" || strings.TrimSpace(order.RejectedTime) != "" {
		return "CANCELLED"
	}
	if strings.TrimSpace(order.DeliveredTime) != "" || strings.TrimSpace(order.CompletedTime) != "" {
		return "DELIVERED"
	}
	if strings.TrimSpace(order.ShippedTime) != "" || strings.TrimSpace(order.OrderTrackingNumber) != "" || strings.TrimSpace(order.LogisticsNumber) != "" {
		return "SHIPPED"
	}
	if strings.TrimSpace(order.PaidTime) != "" || strings.TrimSpace(order.ConfirmedTime) != "" {
		return "PROCESSING"
	}

	switch order.OrderStatus {
	case 0:
		return "PENDING"
	case 1:
		return "PENDING"
	case 2:
		return "PROCESSING"
	case 3:
		return "PROCESSING"
	case 4:
		return "SHIPPED"
	case 5:
		return "DELIVERED"
	case 6:
		return "DELIVERED"
	case 7:
		return "CANCELLED"
	default:
		return "PENDING"
	}
}

func normalizeKilimallStatus(status string) string {
	s := strings.ToUpper(strings.TrimSpace(status))
	if s == "" {
		return "PENDING"
	}
	return s
}

func updateKilimallServiceStatus(status, message string) {
	global.DB.Where("id = ?", kilimallServiceStatusID).Updates(models.ServiceStatus{
		Status:  status,
		Message: message,
	})
}

func kilimallOriginFromAPI(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "https://seller.kilimall.ke"
	}
	host := strings.Replace(u.Host, "seller-api.", "seller.", 1)
	return u.Scheme + "://" + host
}

func kilimallTimeRange() (string, string) {
	start := strings.TrimSpace(global.Config.Kilimall.StartTime)
	end := strings.TrimSpace(global.Config.Kilimall.EndTime)
	if start != "" && end != "" {
		return start, end
	}

	now := time.Now().UTC()
	defaultStart := now.AddDate(0, -3, 0)
	if start == "" {
		start = defaultStart.Format("2006-01-02T15:04:05.000Z")
	}
	if end == "" {
		end = now.Format("2006-01-02T15:04:05.999Z")
	}
	return start, end
}

func delayWithJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return 1200 * time.Millisecond
	}
	jitter := time.Duration(kilimallRand.Intn(500)) * time.Millisecond
	return base + jitter
}

func buildKilimallSyntheticID(orderID, orderNumber, trackingNumber, skuID string) string {
	orderKey := strings.TrimSpace(orderID)
	if orderKey == "" {
		orderKey = strings.TrimSpace(orderNumber)
	}
	if orderKey == "" {
		orderKey = "unknown-order"
	}
	trackKey := strings.TrimSpace(trackingNumber)
	if trackKey == "" {
		trackKey = "no-track"
	}
	skuKey := strings.TrimSpace(skuID)
	if skuKey == "" {
		skuKey = "no-sku"
	}
	return "kilimall_" + orderKey + "_" + trackKey + "_" + skuKey
}

func pickLatestTime(order kilimallOrder) time.Time {
	candidates := []string{
		order.CompletedTime,
		order.DeliveredTime,
		order.ShippedTime,
		order.ConfirmedTime,
		order.PaidTime,
		order.CreatedTime,
	}
	latest := time.Time{}
	for _, raw := range candidates {
		t := parseKilimallTime(raw)
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}
	if latest.IsZero() {
		return time.Now()
	}
	return latest
}

func parseKilimallTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.000Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func mapRegionToCountry(regionCode string) string {
	switch strings.ToUpper(strings.TrimSpace(regionCode)) {
	case "KE":
		return "Kenya"
	case "NG":
		return "Nigeria"
	case "GH":
		return "Ghana"
	default:
		return strings.ToUpper(strings.TrimSpace(regionCode))
	}
}

func mapRegionToCurrency(regionCode string) string {
	switch strings.ToUpper(strings.TrimSpace(regionCode)) {
	case "KE":
		return "KES"
	case "NG":
		return "NGN"
	case "GH":
		return "GHS"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Now()
}
