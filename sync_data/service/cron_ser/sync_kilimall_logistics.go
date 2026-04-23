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
	UpdatedAt       time.Time
}

type kilimallEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func SyncKilimallLogistics() {
	global.Log.Info("开始同步 Kilimall 物流数据")

	cookie := strings.TrimSpace(global.Config.Kilimall.Cookie)
	authToken := strings.TrimSpace(global.Config.Kilimall.AuthToken)
	if cookie == "" && authToken == "" {
		updateKilimallServiceStatus("错误", "Kilimall 鉴权为空，请在 settings.yaml 中填写 kilimall.cookie 或 kilimall.auth_token")
		return
	}

	records, err := FetchKilimallLogistics(cookie, authToken)
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

	if err := upsertKilimallLogistics(cleaned); err != nil {
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 入库失败: %v", err))
		return
	}

	updateKilimallServiceStatus("正常", fmt.Sprintf("Kilimall 物流同步成功，记录数: %d", len(cleaned)))
}

func FetchKilimallLogistics(cookie, authToken string) ([]KilimallLogisticsRecord, error) {
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

	for page := 1; ; page++ {
		time.Sleep(delayWithJitter(delay))

		var (
			records []KilimallLogisticsRecord
			hasNext bool
			err    error
		)

		for attempt := 1; attempt <= maxRetries; attempt++ {
			records, hasNext, err = fetchKilimallPage(client, baseURL, apiPath, page, pageSize, cookie, authToken)
			if err == nil {
				break
			}
			if errors.Is(err, ErrKilimallAuthExpired) {
				return nil, err
			}
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * time.Second
				global.Log.Warnf("Kilimall 第 %d 页第 %d 次请求失败: %v，%v 后重试", page, attempt, err, backoff)
				time.Sleep(backoff)
			}
		}
		if err != nil {
			return nil, err
		}

		allRecords = append(allRecords, records...)
		if !hasNext || len(records) == 0 {
			break
		}
	}

	return allRecords, nil
}

func fetchKilimallPage(client *http.Client, baseURL, apiPath string, page, pageSize int, cookie, authToken string) ([]KilimallLogisticsRecord, bool, error) {
	u, err := buildKilimallURL(baseURL, apiPath, page, pageSize)
	if err != nil {
		return nil, false, err
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, false, err
	}
	setKilimallHeaders(req, baseURL, cookie, authToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, ErrKilimallAuthExpired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("kilimall http status=%d body=%.400s", resp.StatusCode, string(body))
	}

	records, totalPage, err := parseKilimallResponse(body)
	if err != nil {
		return nil, false, err
	}

	hasNext := totalPage == 0 || page < totalPage
	return records, hasNext, nil
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

func parseKilimallResponse(body []byte) ([]KilimallLogisticsRecord, int, error) {
	var envelope kilimallEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, 0, fmt.Errorf("kilimall json parse failed: %w, body=%.400s", err, string(body))
	}

	if envelope.Code == http.StatusUnauthorized || envelope.Code == http.StatusForbidden {
		return nil, 0, ErrKilimallAuthExpired
	}
	if envelope.Code != 0 && envelope.Code != 200 {
		msgLower := strings.ToLower(envelope.Msg)
		if strings.Contains(msgLower, "auth") || strings.Contains(msgLower, "forbidden") || strings.Contains(msgLower, "unauthorized") {
			return nil, 0, ErrKilimallAuthExpired
		}
		return nil, 0, fmt.Errorf("kilimall api error code=%d msg=%s", envelope.Code, envelope.Msg)
	}

	return extractKilimallRecords(envelope.Data)
}

func extractKilimallRecords(raw json.RawMessage) ([]KilimallLogisticsRecord, int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, 0, nil
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(raw, &dataMap); err != nil {
		var arr []map[string]interface{}
		if errArr := json.Unmarshal(raw, &arr); errArr == nil {
			records := make([]KilimallLogisticsRecord, 0, len(arr))
			for _, row := range arr {
				records = append(records, mapKilimallGenericRecord(row))
			}
			return records, 0, nil
		}
		return nil, 0, fmt.Errorf("kilimall data parse failed: %w", err)
	}

	recordRows := extractRecordRows(dataMap)
	totalPage := intFromMap(dataMap, "totalPage", "pages", "pageCount", "totalPages")

	records := make([]KilimallLogisticsRecord, 0, len(recordRows))
	for _, row := range recordRows {
		records = append(records, mapKilimallGenericRecord(row))
	}
	return records, totalPage, nil
}

func extractRecordRows(data map[string]interface{}) []map[string]interface{} {
	keys := []string{"records", "list", "rows", "items", "orderList", "orders", "result"}
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if rows := normalizeRows(value); len(rows) > 0 {
				return rows
			}
		}
	}

	if nested, ok := data["data"].(map[string]interface{}); ok {
		return extractRecordRows(nested)
	}
	return nil
}

func normalizeRows(v interface{}) []map[string]interface{} {
	rawList, ok := v.([]interface{})
	if !ok {
		return nil
	}
	rows := make([]map[string]interface{}, 0, len(rawList))
	for _, item := range rawList {
		row, ok := item.(map[string]interface{})
		if ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func mapKilimallGenericRecord(row map[string]interface{}) KilimallLogisticsRecord {
	orderID := stringFromMap(row, "orderId", "order_id", "orderID")
	orderNumber := stringFromMap(row, "orderNumber", "order_number", "orderNo", "order_no")
	trackingNumber := stringFromMap(row, "trackingNumber", "tracking_number", "trackingNo", "shippingNo", "shippingNumber", "logisticsNo")
	trackingURL := stringFromMap(row, "trackingUrl", "tracking_url", "trackUrl", "logisticsUrl")
	shopID := stringFromMap(row, "shopId", "shop_id", "storeId", "store_id")
	status := stringFromMap(row, "status", "orderStatus", "shippingStatus")

	id := stringFromMap(row, "id", "itemId", "orderItemId")
	if id == "" {
		id = buildKilimallSyntheticID(orderID, orderNumber, trackingNumber)
	}

	updatedAt := parseKilimallTimeValue(valueFromMap(row, "updatedAt", "updated_at", "updateTime", "update_time"))

	return KilimallLogisticsRecord{
		ID:              id,
		OrderID:         orderID,
		OrderNumber:     orderNumber,
		TrackingNumber:  trackingNumber,
		TrackingURL:     trackingURL,
		Status:          status,
		ShipmentType:    stringFromMap(row, "shipmentType", "shipment_type"),
		DeliveryOption:  stringFromMap(row, "deliveryOption", "delivery_option"),
		ShopID:          shopID,
		CountryCode:     stringFromMap(row, "countryCode", "country_code"),
		CountryName:     stringFromMap(row, "countryName", "country_name"),
		CountryCurrency: stringFromMap(row, "currency", "countryCurrency", "country_currency"),
		UpdatedAt:       updatedAt,
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
				"country_currency", "updated_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}

	return nil
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

func normalizeKilimallStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "0", "pending", "wait_ship", "to_ship", "待发货":
		return "pending"
	case "1", "shipped", "in_transit", "运输中", "已发货":
		return "shipped"
	case "2", "delivered", "已签收", "completed":
		return "delivered"
	case "3", "cancelled", "canceled", "已取消":
		return "cancelled"
	default:
		if s == "" {
			return "unknown"
		}
		return s
	}
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

func buildKilimallSyntheticID(orderID, orderNumber, trackingNumber string) string {
	orderKey := strings.TrimSpace(orderID)
	if orderKey == "" {
		orderKey = strings.TrimSpace(orderNumber)
	}
	trackingKey := strings.TrimSpace(trackingNumber)
	if trackingKey == "" {
		trackingKey = "no-track"
	}
	if orderKey == "" {
		orderKey = "unknown-order"
	}
	return "kilimall_" + orderKey + "_" + trackingKey
}

func valueFromMap(m map[string]interface{}, keys ...string) interface{} {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func stringFromMap(m map[string]interface{}, keys ...string) string {
	v := valueFromMap(m, keys...)
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(val, 'f', -1, 64))
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func intFromMap(m map[string]interface{}, keys ...string) int {
	v := valueFromMap(m, keys...)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(val))
		return n
	default:
		return 0
	}
}

func parseKilimallTimeValue(v interface{}) time.Time {
	if v == nil {
		return time.Now()
	}

	switch val := v.(type) {
	case float64:
		return parseUnixMillis(int64(val))
	case int64:
		return parseUnixMillis(val)
	case int:
		return parseUnixMillis(int64(val))
	case string:
		raw := strings.TrimSpace(val)
		if raw == "" {
			return time.Now()
		}
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return parseUnixMillis(n)
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
	}
	return time.Now()
}

func parseUnixMillis(ts int64) time.Time {
	if ts <= 0 {
		return time.Now()
	}
	if ts > 1e12 {
		return time.UnixMilli(ts)
	}
	return time.Unix(ts, 0)
}
