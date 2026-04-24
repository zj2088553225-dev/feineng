package cron_ser

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm/clause"
)

type kilimallCostSheetPage struct {
	Records []models.CostSheet
	Total   int
}

func SyncKilimallCostSheets() {
	global.Log.Info("开始同步 Kilimall 物流计费单")

	cookie := strings.TrimSpace(global.Config.Kilimall.Cookie)
	authToken := strings.TrimSpace(global.Config.Kilimall.AuthToken)
	if cookie == "" && authToken == "" {
		updateKilimallServiceStatus("错误", "Kilimall 鉴权为空，请在 settings.yaml 中填写 kilimall.cookie 或 kilimall.auth_token")
		return
	}

	records, err := FetchKilimallCostSheets(cookie, authToken)
	if err != nil {
		if errors.Is(err, ErrKilimallAuthExpired) {
			updateKilimallServiceStatus("错误", "Kilimall Cookie/Token 已过期或被拦截，请手动更新 settings.yaml")
			return
		}
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 计费单同步失败: %v", err))
		return
	}
	if len(records) == 0 {
		updateKilimallServiceStatus("错误", "Kilimall 计费单返回为空或清洗后无有效数据")
		return
	}

	if err := upsertKilimallCostSheets(records); err != nil {
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 计费单入库失败: %v", err))
		return
	}

	orderNumbers := distinctCostSheetOrderNumbers(records)
	if err := recalculateKilimallOrderProfits(orderNumbers); err != nil {
		updateKilimallServiceStatus("错误", fmt.Sprintf("Kilimall 利润核算失败: %v", err))
		return
	}

	updateKilimallServiceStatus("正常", fmt.Sprintf("Kilimall 计费单同步成功，记录数: %d，核算订单数: %d", len(records), len(orderNumbers)))
}

func FetchKilimallCostSheets(cookie, authToken string) ([]models.CostSheet, error) {
	baseURL := strings.TrimSpace(global.Config.Kilimall.BaseURL)
	if baseURL == "" {
		baseURL = "https://seller-api.kilimall.ke"
	}
	apiPath := strings.TrimSpace(global.Config.Kilimall.CostSheetAPI)
	if apiPath == "" {
		apiPath = "/cost-sheets"
	}
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}

	pageSize := global.Config.Kilimall.CostSheetPageSize
	if pageSize <= 0 {
		pageSize = 50
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
	allRecords := make([]models.CostSheet, 0)
	seen := make(map[string]struct{})

	for page := 1; ; page++ {
		time.Sleep(delayWithJitter(delay))

		var (
			pageData kilimallCostSheetPage
			err      error
		)
		for attempt := 1; attempt <= maxRetries; attempt++ {
			pageData, err = fetchKilimallCostSheetPage(client, baseURL, apiPath, page, pageSize, cookie, authToken)
			if err == nil {
				break
			}
			if errors.Is(err, ErrKilimallAuthExpired) {
				return nil, err
			}
			if attempt < maxRetries {
				backoff := time.Duration(attempt*attempt) * time.Second
				global.Log.Warnf("Kilimall 计费单第 %d 页第 %d 次请求失败: %v，%v 后重试", page, attempt, err, backoff)
				time.Sleep(backoff)
			}
		}
		if err != nil {
			return nil, err
		}

		for _, record := range pageData.Records {
			if record.ID == "" || record.TransactionOrderNumber == "" {
				continue
			}
			if _, exists := seen[record.ID]; exists {
				continue
			}
			seen[record.ID] = struct{}{}
			allRecords = append(allRecords, record)
		}

		if len(pageData.Records) == 0 {
			break
		}
		if pageData.Total > 0 && page*pageSize >= pageData.Total {
			break
		}
		if pageData.Total == 0 && len(pageData.Records) < pageSize {
			break
		}
	}

	return allRecords, nil
}

func fetchKilimallCostSheetPage(client *http.Client, baseURL, apiPath string, page, pageSize int, cookie, authToken string) (kilimallCostSheetPage, error) {
	u, err := buildKilimallCostSheetURL(baseURL, apiPath, page, pageSize)
	if err != nil {
		return kilimallCostSheetPage{}, err
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return kilimallCostSheetPage{}, err
	}
	setKilimallHeaders(req, baseURL, cookie, authToken)

	resp, err := client.Do(req)
	if err != nil {
		return kilimallCostSheetPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return kilimallCostSheetPage{}, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return kilimallCostSheetPage{}, ErrKilimallAuthExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return kilimallCostSheetPage{}, fmt.Errorf("Kilimall 计费单 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return parseKilimallCostSheetPage(body)
}

func buildKilimallCostSheetURL(baseURL, apiPath string, page, pageSize int) (*url.URL, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + apiPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(pageSize))
	q.Set("pagination", strconv.Itoa(page))
	q.Set("skip", strconv.Itoa((page-1)*pageSize))
	startTime, endTime := kilimallTimeRange()
	q.Set("startTime", startTime)
	q.Set("endTime", endTime)
	u.RawQuery = q.Encode()
	return u, nil
}

func parseKilimallCostSheetPage(body []byte) (kilimallCostSheetPage, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return kilimallCostSheetPage{}, err
	}
	if code, ok := numericFromAny(payload["code"]); ok && code != 0 && code != 200 {
		msg := stringFromAny(payload["msg"])
		if msg == "" {
			msg = stringFromAny(payload["message"])
		}
		if isKilimallAuthErrorCode(code, msg) {
			return kilimallCostSheetPage{}, ErrKilimallAuthExpired
		}
		return kilimallCostSheetPage{}, fmt.Errorf("Kilimall 计费单业务错误 code=%v msg=%s", code, msg)
	}

	data := anyMap(payload["data"])
	if len(data) == 0 {
		data = payload
	}

	rawRecords := firstArray(data, "list", "records", "items", "rows", "costSheets", "cost_sheets")
	if rawRecords == nil {
		rawRecords = firstArray(payload, "list", "records", "items", "rows", "costSheets", "cost_sheets")
	}

	records := make([]models.CostSheet, 0, len(rawRecords))
	now := time.Now()
	for index, raw := range rawRecords {
		row := anyMap(raw)
		if len(row) == 0 {
			continue
		}
		record := mapKilimallCostSheet(row, index, now)
		if record.TransactionOrderNumber == "" || record.Amount == 0 {
			continue
		}
		records = append(records, record)
	}

	total := firstInt(data, "total", "count", "totalCount")
	if total == 0 {
		total = firstInt(payload, "total", "count", "totalCount")
	}
	return kilimallCostSheetPage{Records: records, Total: total}, nil
}

func mapKilimallCostSheet(row map[string]any, index int, now time.Time) models.CostSheet {
	transactionOrderNumber := firstStringFromMap(row,
		"transactionOrderNumber", "transaction_order_number", "orderNumber", "order_number", "orderSn", "order_sn", "orderNo", "order_no")
	trackingNumber := firstStringFromMap(row,
		"trackingNumber", "tracking_number", "logisticsNumber", "logistics_number", "waybillNo", "waybill_no", "trackingNo", "tracking_no")
	costType := firstStringFromMap(row,
		"costType", "cost_type", "feeType", "fee_type", "type", "feeName", "fee_name", "chargeType", "charge_type")
	currency := firstStringFromMap(row, "currency", "currencyCode", "currency_code", "amountCurrency", "amount_currency")
	amount := firstFloatFromMap(row, "amount", "feeAmount", "fee_amount", "costAmount", "cost_amount", "chargeAmount", "charge_amount", "deductAmount", "deduct_amount")
	chargeWeight := firstFloatFromMap(row, "chargeWeight", "charge_weight", "billingWeight", "billing_weight", "weight")
	deductionStatus := firstStringFromMap(row, "deductionStatus", "deduction_status", "deductStatus", "deduct_status", "status")
	rawStatus := firstStringFromMap(row, "rawStatus", "raw_status", "statusText", "status_text", "state")
	occurredAt := firstTimeFromMap(row, "occurredAt", "occurred_at", "createdTime", "created_time", "date", "time", "updatedTime", "updated_time")

	id := firstStringFromMap(row, "id", "costSheetId", "cost_sheet_id", "billId", "bill_id", "recordId", "record_id")
	if id == "" {
		id = buildKilimallCostSheetID(transactionOrderNumber, trackingNumber, costType, amount, occurredAt, index)
	}

	return models.CostSheet{
		ID:                     id,
		TransactionOrderNumber: transactionOrderNumber,
		TrackingNumber:         trackingNumber,
		CostType:               costType,
		ChargeWeight:           chargeWeight,
		Currency:               currency,
		Amount:                 amount,
		DeductionStatus:        deductionStatus,
		RawStatus:              rawStatus,
		OccurredAt:             occurredAt,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func upsertKilimallCostSheets(records []models.CostSheet) error {
	if len(records) == 0 {
		return nil
	}
	const batchSize = 300
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]
		if err := global.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"transaction_order_number", "tracking_number", "cost_type", "charge_weight",
				"currency", "amount", "deduction_status", "raw_status", "occurred_at", "updated_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	return nil
}

func recalculateKilimallOrderProfits(orderNumbers []string) error {
	orderNumbers = cleanOrderNumbers(orderNumbers)
	if len(orderNumbers) == 0 {
		return nil
	}

	type costTotal struct {
		TransactionOrderNumber string
		Total                  float64
	}
	var totals []costTotal
	if err := global.DB.Model(&models.CostSheet{}).
		Select("transaction_order_number, SUM(amount) AS total").
		Where("transaction_order_number IN ?", orderNumbers).
		Group("transaction_order_number").
		Scan(&totals).Error; err != nil {
		return err
	}

	totalMap := make(map[string]float64, len(totals))
	for _, total := range totals {
		totalMap[strings.TrimSpace(total.TransactionOrderNumber)] = total.Total
	}

	var orders []models.Order
	if err := global.DB.Where("number IN ?", orderNumbers).Find(&orders).Error; err != nil {
		return err
	}
	for _, order := range orders {
		totalShippingCost := totalMap[strings.TrimSpace(order.Number)]
		netProfit := order.TotalAmountLocalValue - totalShippingCost
		if err := global.DB.Model(&models.Order{}).
			Where("id = ?", order.ID).
			Updates(map[string]any{
				"total_shipping_cost": totalShippingCost,
				"net_profit":          netProfit,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func distinctCostSheetOrderNumbers(records []models.CostSheet) []string {
	seen := make(map[string]struct{})
	orderNumbers := make([]string, 0)
	for _, record := range records {
		orderNumber := strings.TrimSpace(record.TransactionOrderNumber)
		if orderNumber == "" {
			continue
		}
		if _, exists := seen[orderNumber]; exists {
			continue
		}
		seen[orderNumber] = struct{}{}
		orderNumbers = append(orderNumbers, orderNumber)
	}
	return orderNumbers
}

func cleanOrderNumbers(orderNumbers []string) []string {
	seen := make(map[string]struct{})
	cleaned := make([]string, 0, len(orderNumbers))
	for _, orderNumber := range orderNumbers {
		trimmed := strings.TrimSpace(orderNumber)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

func buildKilimallCostSheetID(orderNumber, trackingNumber, costType string, amount float64, occurredAt *time.Time, index int) string {
	parts := []string{
		strings.TrimSpace(orderNumber),
		strings.TrimSpace(trackingNumber),
		strings.TrimSpace(costType),
		strconv.FormatFloat(amount, 'f', 6, 64),
		strconv.Itoa(index),
	}
	if occurredAt != nil && !occurredAt.IsZero() {
		parts = append(parts, occurredAt.UTC().Format(time.RFC3339Nano))
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "kilimall_cost_" + hex.EncodeToString(sum[:])
}

func firstArray(data map[string]any, keys ...string) []any {
	for _, key := range keys {
		if raw, ok := data[key]; ok {
			if arr, ok := raw.([]any); ok {
				return arr
			}
		}
	}
	return nil
}

func anyMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return nil
}

func firstStringFromMap(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringFromAny(data[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstFloatFromMap(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := numericFromAny(data[key]); ok {
			return value
		}
	}
	return 0
}

func firstTimeFromMap(data map[string]any, keys ...string) *time.Time {
	for _, key := range keys {
		raw := stringFromAny(data[key])
		if raw == "" {
			continue
		}
		parsed := parseKilimallTime(raw)
		if !parsed.IsZero() {
			return &parsed
		}
	}
	return nil
}

func firstInt(data map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := numericFromAny(data[key]); ok {
			return int(value)
		}
	}
	return 0
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func numericFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(strings.ReplaceAll(trimmed, ",", ""), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func isKilimallAuthErrorCode(code float64, msg string) bool {
	if code == 1001 || code == 401 || code == 403 {
		return true
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden") || strings.Contains(lower, "auth") || strings.Contains(lower, "token")
}
