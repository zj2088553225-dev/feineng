package cron_ser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SyncOrders() {
	global.Log.Info("定时同步订单数据开始")
	_, err := FetchAndSyncOrdersInBatches(global.Config.Jumia.AccessToken)
	if err != nil {
		global.DB.Where("id = ?", 5).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	_, err = FetchAndSyncOrdersInBatches(global.Config.JumiaTwo.AccessToken)
	if err != nil {
		global.DB.Where("id = ?", 5).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	err = SyncOrderItems("fb9d1d71-9f02-489b-9930-df0e80a4ba53", global.Config.Jumia.AccessToken)
	if err != nil {
		global.DB.Where("id = ?", 5).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Error(err.Error())
		return
	}

	err = SyncOrderItems("f8162f4a-2ccb-4f2d-a711-acce9d5cf4a0", global.Config.JumiaTwo.AccessToken)
	if err != nil {
		global.DB.Where("id = ?", 5).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Error(err.Error())
		return
	}
	global.DB.Where("id = ?", 5).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: fmt.Sprintf("定时同步订单数据成功"),
	})
}

// API 响应结构
type OrderResponsePage struct {
	Orders     []APIOrder `json:"orders"`
	NextToken  *string    `json:"nextToken"`
	IsLastPage bool       `json:"isLastPage"`
}

type APIOrder struct {
	ID                       string       `json:"id"`
	ShopIDs                  []string     `json:"shopIds"`
	TotalItems               int          `json:"totalItems"`
	PackedItems              int          `json:"packedItems"`
	IsPrepayment             bool         `json:"isPrepayment"`
	HasMultipleStatus        bool         `json:"hasMultipleStatus"`
	HasItemsFulfilledByJumia bool         `json:"hasItemsFulfilledByJumia"`
	PendingSince             string       `json:"pendingSince"`
	Status                   string       `json:"status"`
	DeliveryOption           string       `json:"deliveryOption"`
	Number                   string       `json:"number"`
	TotalAmount              Amount       `json:"totalAmount"`
	TotalAmountLocal         Amount       `json:"totalAmountLocal"`
	Country                  Country      `json:"country"`
	ShippingAddress          ShippingAddr `json:"shippingAddress"`
	CreatedAt                string       `json:"createdAt"`
	UpdatedAt                string       `json:"updatedAt"`
}

type Amount struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

type Country struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	CurrencyCode string `json:"currencyCode"`
}

type ShippingAddr struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postalCode"`
	Ward        string `json:"ward"`
	Region      string `json:"region"`
	CountryName string `json:"countryName"`
}

// convertToUserOrders 转换 API 订单为本地模型
func convertToUserOrders(apiOrders []APIOrder) []models.Order {
	var result []models.Order
	for _, o := range apiOrders {
		if o.ID == "" {
			global.Log.Warnf("⚠️ 跳过无效订单: ID 为空, Number=%s", o.Number)
			continue
		}

		// 过滤国家：跳过 Senegal 和 Ivory-Coast
		if o.Country.Name == "Senegal" || o.Country.Name == "Ivory-Coast" {
			global.Log.Debugf("📍 跳过国家: %s, OrderID=%s, Number=%s", o.Country.Name, o.ID, o.Number)
			continue
		}
		// 假设 o.CreatedAt 是 string 类型，值为 "2025-06-24T09:14:16Z"
		str := o.CreatedAt

		// 使用 time.RFC3339 解析 ISO8601 格式时间（带 T 和 Z）
		parsedTimeC, err := time.Parse(time.RFC3339, str)
		if err != nil {
			log.Fatal("时间解析失败:", err)
		}
		strU := o.UpdatedAt
		parsedTimeU, err := time.Parse(time.RFC3339, strU)
		if err != nil {
			log.Fatal("时间解析失败:", err)
		}

		result = append(result, models.Order{
			ID:           o.ID,
			ShopIDs:      o.ShopIDs[0], // ✅ 传整个切片
			TotalItems:   o.TotalItems,
			PackedItems:  o.PackedItems,
			IsPrepayment: o.IsPrepayment,
			Status:       o.Status,
			Number:       o.Number,
			CreatedAt:    parsedTimeC,
			UpdatedAt:    parsedTimeU,

			TotalAmountCurrency:      o.TotalAmount.Currency,
			TotalAmountValue:         o.TotalAmount.Value,
			TotalAmountLocalCurrency: o.TotalAmountLocal.Currency,
			TotalAmountLocalValue:    o.TotalAmountLocal.Value,

			CountryCode:     o.Country.Code,
			CountryName:     o.Country.Name,
			CountryCurrency: o.Country.CurrencyCode,

			ShippingFirstName:   o.ShippingAddress.FirstName,
			ShippingLastName:    o.ShippingAddress.LastName,
			ShippingAddress:     o.ShippingAddress.Address,
			ShippingCity:        o.ShippingAddress.City,
			ShippingPostalCode:  o.ShippingAddress.PostalCode,
			ShippingWard:        o.ShippingAddress.Ward,
			ShippingRegion:      o.ShippingAddress.Region,
			ShippingCountryName: o.ShippingAddress.CountryName,
		})
	}
	return result
}

// flushOrderBatch 批量写入订单
func flushOrderBatch(orders []models.Order) (*FlushBatchStats, error) {
	if len(orders) == 0 {
		return &FlushBatchStats{}, nil
	}

	var orderIDs []string
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}
	sort.Strings(orderIDs)
	global.Log.Infof("📦 flushOrderBatch: 处理 %d 个订单, 示例 ID: %s",
		len(orders), truncateString(strings.Join(orderIDs[:min(5, len(orderIDs))], ","), 100))

	// 使用 GORM 的 Upsert（OnConflict）
	result := global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // 以 id 为唯一键
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":               gorm.Expr("VALUES(status)"),
			"packed_items":         gorm.Expr("VALUES(packed_items)"),
			"updated_at":           gorm.Expr("VALUES(updated_at)"),
			"total_items":          gorm.Expr("VALUES(total_items)"),
			"total_amount_value":   gorm.Expr("VALUES(total_amount_value)"),
			"shipping_address":     gorm.Expr("VALUES(shipping_address)"),
			"shipping_city":        gorm.Expr("VALUES(shipping_city)"),
			"shipping_postal_code": gorm.Expr("VALUES(shipping_postal_code)"),
		}),
	}).Create(&orders)

	if result.Error != nil {
		global.Log.Errorf("❌ Upsert 批量写入失败: %v", result.Error)
		return nil, result.Error
	}

	// 统计：GORM 不直接返回 created/updated 数量
	// 但我们可以通过影响行数估算（不精确）
	created := int(result.RowsAffected) // 注意：这不完全等于“创建”的数量
	updated := len(orders) - created
	if updated < 0 {
		updated = 0
	}

	global.Log.Infof("✅ Upsert 完成: 影响 %d 行, 估算 创建=%d, 更新=%d", result.RowsAffected, created, updated)

	return &FlushBatchStats{
		Created: created,
		Updated: updated,
	}, nil
}

//// FetchAndSyncOrdersInBatches 同步最近 3 个月订单
//func FetchAndSyncOrdersInBatches() (*SyncStats, error) {
//	now := time.Now()
//	endAt := now.Format("2006-01-02")
//	startAt := now.AddDate(0, 0, -1).Format("2006-01-02") // ✅ 修正为 3 个月
//
//	return FetchAndSyncOrdersInBatchesWithRange(startAt, endAt)
//}

// FetchAndSyncOrdersInBatches 同步过去 6 个月的订单，每批最多 2 个月
func FetchAndSyncOrdersInBatches(token string) (*SyncStats, error) {
	now := time.Now()
	var totalStats SyncStats

	// 向前推 6 个月，逐个 2 个月区间查询
	currentEnd := now

	for i := 0; i < 2; i++ { // 6个月 / 每次2个月 = 3次
		currentStart := currentEnd.AddDate(0, -2, 0) // 往前推2个月

		// 格式化时间
		startAt := currentStart.Format("2006-01-02")
		endAt := currentEnd.Format("2006-01-02")

		global.Log.Infof("📅 开始同步订单区间: %s 到 %s", startAt, endAt)

		// 调用实际同步函数
		stats, err := FetchAndSyncOrdersInBatchesWithRange(startAt, endAt, token)
		if err != nil {
			global.Log.Errorf("❌ 同步订单区间 [%s, %s] 失败: %v", startAt, endAt, err)
			// 可选择 continue 继续下一批，或 return 错误
			continue // 建议继续，避免因某一批失败导致整体中断
		}

		// 累计统计
		totalStats.SuccessCount += stats.SuccessCount
		totalStats.FailureCount += stats.FailureCount
		totalStats.TotalCount += stats.TotalCount

		// 下一批的结束时间为当前开始时间
		currentEnd = currentStart
	}

	global.Log.Infof("🎉 订单同步完成：总计成功=%d, 失败=%d, 总数=%d",
		totalStats.SuccessCount, totalStats.FailureCount, totalStats.TotalCount)

	return &totalStats, nil
}

// FetchAndSyncOrdersInBatchesWithRange 按时间范围同步订单
func FetchAndSyncOrdersInBatchesWithRange(startTime, endTime, token string) (*SyncStats, error) {
	const baseURL = "https://vendor-api.jumia.com/orders"

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var (
		nextToken    string
		totalPages   = 0
		totalFetched = 0
		totalSkipped = 0
		totalCreated = 0
		totalUpdated = 0
		buffer       []models.Order
	)

	global.Log.Infof("🚀 开始分页拉取并分批同步订单数据: %s 到 %s", startTime, endTime)

	for {
		totalPages++

		// ✅ 构建 URL：始终携带时间范围
		baseQuery := fmt.Sprintf("createdAfter=%s&createdBefore=%s&size=%d",
			url.QueryEscape(startTime),
			url.QueryEscape(endTime),
			pageSize,
		)
		var urlStr string
		if nextToken != "" {
			urlStr = fmt.Sprintf("%s?%s&token=%s", baseURL, baseQuery, url.QueryEscape(nextToken))
		} else {
			urlStr = fmt.Sprintf("%s?%s", baseURL, baseQuery)
		}

		global.Log.Infof("📘 开始拉取第 %d 页: URL=%s, NextToken=%s",
			totalPages,
			redactURLToken(urlStr),
			truncateString(nextToken, 20))

		var page OrderResponsePage
		var err error

		// 重试机制
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = requestOrderPage(client, urlStr, &page, token)
			if err == nil {
				break
			}
			if attempt < maxRetries {
				backoff := baseBackoff * time.Duration(1<<(attempt-1))
				global.Log.Warnf("❌ 请求失败 (第 %d 次): %v, %v 后重试", attempt, err, backoff)
				time.Sleep(backoff)
			}
		}

		if err != nil {
			global.Log.Errorf("🚨 拉取第 %d 页订单失败，终止同步: %v", totalPages, err)
			return nil, fmt.Errorf("拉取第 %d 页订单失败: %w", totalPages, err)
		}

		count := len(page.Orders)
		if count == 0 {
			global.Log.Info("📭 当前页无订单数据")
		} else {
			global.Log.Debugf("📥 第 %d 页: 拉取到 %d 个订单, FirstID=%s, LastID=%s",
				totalPages, count, page.Orders[0].ID, page.Orders[count-1].ID)
		}

		// 转换并过滤
		orders := convertToUserOrders(page.Orders)
		skipped := len(page.Orders) - len(orders)
		totalFetched += len(page.Orders)
		totalSkipped += skipped
		buffer = append(buffer, orders...)

		// 批量处理
		if len(buffer) >= batchSize {
			stats, err := flushOrderBatch(buffer[:batchSize])
			if err != nil {
				return nil, fmt.Errorf("批量写入订单失败: %w", err)
			}
			totalCreated += stats.Created
			totalUpdated += stats.Updated
			buffer = buffer[batchSize:]
		}

		// 是否结束
		if page.IsLastPage || page.NextToken == nil {
			global.Log.Infof("✅ 到达最后一页，分页结束")
			break
		}
		nextToken = *page.NextToken

		time.Sleep(requestDelay)
	}

	// 处理剩余
	if len(buffer) > 0 {
		global.Log.Infof("🧹 处理最后一批 %d 个订单", len(buffer))
		stats, err := flushOrderBatch(buffer)
		if err != nil {
			return nil, fmt.Errorf("最终批次写入失败: %w", err)
		}
		totalCreated += stats.Created
		totalUpdated += stats.Updated
	}

	global.Log.Infof("🎉 订单同步完成：共 %d 页，处理 %d 个原始订单，跳过 %d，创建 %d，更新 %d",
		totalPages, totalFetched, totalSkipped, totalCreated, totalUpdated)

	return &SyncStats{
		Fetched: totalFetched,
		Skipped: totalSkipped,
		Created: totalCreated,
		Updated: totalUpdated,
	}, nil
}

// requestOrderPage 发起 HTTP 请求并解析响应
func requestOrderPage(client *http.Client, urlStr string, target *OrderResponsePage, token string) error {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	global.Log.Debugf("📡 发起请求: %s, Auth=Bearer %s...",
		redactURLToken(urlStr),
		truncateString(global.Config.Jumia.AccessToken, 8))

	resp, err := client.Do(req)
	if err != nil {
		global.Log.Errorf("🌐 网络请求失败: %v, URL: %s", err, redactURLToken(urlStr))
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		global.Log.Errorf("📥 读取响应体失败: %v", err)
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	global.Log.Debugf("📥 响应状态: %d, Body长度: %d, 片段: %.200s",
		resp.StatusCode, len(body), string(body))

	if resp.StatusCode == 429 {
		global.Log.Warn("⏳ 触发限流 (429)，请检查速率")
		return fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		global.Log.Errorf("❌ API 错误响应: %d, Body: %.500s", resp.StatusCode, body)
		return fmt.Errorf("API 错误 %d: %.200s", resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, target); err != nil {
		global.Log.Errorf("🧩 JSON 反序列化失败, 原始响应: %.500s", body)
		return fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return nil
}

// 工具函数

// redactURLToken 隐藏 URL 中的 token 参数
func redactURLToken(rawURL string) string {
	re := regexp.MustCompile(`token=([^&]+)`)
	return re.ReplaceAllString(rawURL, "token=***")
}

// truncateString 截断字符串
func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// min 返回最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// joinIDs 提取前 n 个 ID 并拼接
func joinIDs(orders []models.Order, n int) string {
	if len(orders) == 0 {
		return ""
	}
	ids := make([]string, 0, min(n, len(orders)))
	for i := 0; i < min(n, len(orders)); i++ {
		ids = append(ids, orders[i].ID)
	}
	return strings.Join(ids, ",")
}
