package cron_ser

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm/clause"
)

const (
	ItemBatchSize   = 50 // 每次最多查 50 个订单的 items（看 API 限制）
	RequestDelay    = 300 * time.Millisecond
	ItemMaxRetries  = 3
	ItemBaseBackoff = 2 * time.Second
	ItemFlushBatch  = 100
)

// API 响应结构
type OrderItemsResponse struct {
	OrderID     string       `json:"orderId"`
	OrderNumber string       `json:"orderNumber"`
	Items       []ItemDetail `json:"items"`
}

type ItemDetail struct {
	ID                  string       `json:"id"`
	ShopID              string       `json:"shopId"`
	Status              string       `json:"status"`
	TrackingNumber      string       `json:"trackingNumber"`
	TrackingURL         string       `json:"trackingUrl"`
	ShipmentType        string       `json:"shipmentType"`
	DeliveryOption      string       `json:"deliveryOption"`
	IsFulfilledByJumia  bool         `json:"isFulfilledByJumia"`
	ItemPrice           float64      `json:"itemPrice"`
	PaidPrice           float64      `json:"paidPrice"`
	ShippingAmount      float64      `json:"shippingAmount"`
	ItemPriceLocal      float64      `json:"itemPriceLocal"`
	PaidPriceLocal      float64      `json:"paidPriceLocal"`
	ShippingAmountLocal float64      `json:"shippingAmountLocal"`
	ExchangeRate        float64      `json:"exchangeRate"`
	TaxAmount           float64      `json:"taxAmount"`
	VoucherAmount       float64      `json:"voucherAmount"`
	Country             Country      `json:"country"`
	Product             Product      `json:"product"`
	ShippingAddress     ShippingAddr `json:"shippingAddress"`
	CreatedAt           string       `json:"createdAt"`
	UpdatedAt           string       `json:"updatedAt"`
}

type Product struct {
	Name      string `json:"name"`
	SellerSku string `json:"sellerSku"`
	ImageURL  string `json:"imageUrl"`
}

// SyncOrderItems 根据已有订单拉取明细并同步
func SyncOrderItems(shopid, token string) error {
	global.Log.Info("🚚 开始同步订单明细数据...")

	// 1. 获取所有订单 ID
	var orderIDs []string
	err := global.DB.Model(&models.Order{}).Where("shop_ids LIKE ?", "%"+shopid+"%").Pluck("id", &orderIDs).Error
	if err != nil {
		global.Log.Errorf("❌ 查询订单ID失败: %v", err)
		return err
	}

	if len(orderIDs) == 0 {
		global.Log.Info("📭 无订单可同步明细")
		return nil
	}

	global.Log.Infof("📦 共 %d 个订单需要拉取明细", len(orderIDs))

	client := &http.Client{Timeout: 300 * time.Second}
	var allItems []models.OrderItem

	// 2. 分批处理订单 ID
	for i := 0; i < len(orderIDs); i += ItemBatchSize {
		end := i + ItemBatchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		batch := orderIDs[i:end]

		items, err := fetchOrderItems(client, batch, token)
		if err != nil {
			global.Log.Errorf("❌ 拉取订单明细失败: %v, 订单ID: %v", err, batch)
			continue // 继续下一批，不中断
		}
		// 🔥 在写入前填充 JumiaSKU
		err = enrichWithJumiaSKU(items)
		if err != nil {
			global.Log.Errorf("⚠️ 填充 JumiaSKU 失败: %v", err)
			// 可选择继续（使用空值）或跳过这批
		}
		// 🔥 在写入前填充 时间，用于后续的时间匹配
		err = enrichWithDate(items)
		if err != nil {
			global.Log.Errorf("⚠️ 填充 date 失败: %v", err)
			// 可选择继续（使用空值）或跳过这批
		}
		allItems = append(allItems, items...)
		time.Sleep(RequestDelay)
	}

	// 3. 批量 Upsert 到数据库
	if len(allItems) > 0 {
		err = flushOrderItemsBatch(allItems)
		if err != nil {
			global.Log.Errorf("❌ 批量写入订单明细失败: %v", err)
			return err
		}
	}

	global.Log.Infof("✅ 订单明细同步完成: 共处理 %d 个明细项", len(allItems))
	return nil
}

// SyncOrderItems 根据已有订单拉取明细并同步
func SyncONEOrderItems(token string, ordernumber []string) error {
	global.Log.Info("🚚 开始同步订单明细数据...")
	if len(ordernumber) == 0 {
		return nil
	}
	var orderIDs []string
	err := global.DB.Model(&models.Order{}).Where("number in ?", ordernumber).Pluck("id", &orderIDs).Error
	if err != nil {
		global.Log.Errorf("❌ 查询订单ID失败: %v", err)
		return err
	}
	if len(orderIDs) == 0 {
		global.Log.Info("📭 无订单可同步明细")
		return nil
	}

	global.Log.Infof("📦 共 %d 个订单需要拉取明细", len(orderIDs))

	client := &http.Client{Timeout: 300 * time.Second}
	var allItems []models.OrderItem

	// 2. 分批处理订单 ID
	for i := 0; i < len(orderIDs); i += ItemBatchSize {
		end := i + ItemBatchSize
		if end > len(orderIDs) {
			end = len(orderIDs)
		}
		batch := orderIDs[i:end]

		items, err := fetchOrderItems(client, batch, token)
		if err != nil {
			global.Log.Errorf("❌ 拉取订单明细失败: %v, 订单ID: %v", err, batch)
			continue // 继续下一批，不中断
		}
		// 🔥 在写入前填充 JumiaSKU
		err = enrichWithJumiaSKU(items)
		if err != nil {
			global.Log.Errorf("⚠️ 填充 JumiaSKU 失败: %v", err)
			// 可选择继续（使用空值）或跳过这批
		}
		// 🔥 在写入前填充 时间，用于后续的时间匹配
		err = enrichWithDate(items)
		if err != nil {
			global.Log.Errorf("⚠️ 填充 date 失败: %v", err)
			// 可选择继续（使用空值）或跳过这批
		}
		allItems = append(allItems, items...)
		time.Sleep(RequestDelay)
	}

	// 3. 批量 Upsert 到数据库
	if len(allItems) > 0 {
		err := flushOrderItemsBatch(allItems)
		if err != nil {
			global.Log.Errorf("❌ 批量写入订单明细失败: %v", err)
			return err
		}
	}

	global.Log.Infof("✅ 订单明细同步完成: 共处理 %d 个明细项", len(allItems))
	return nil
}

// fetchOrderItems 拉取一批订单的 items
func fetchOrderItems(client *http.Client, orderIDs []string, token string) ([]models.OrderItem, error) {
	baseURL := "https://vendor-api.jumia.com/orders/items"

	// 构建查询参数：orderId=xxx&orderId=yyy
	params := url.Values{}
	for _, id := range orderIDs {
		params.Add("orderId", id)
	}
	urlStr := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	global.Log.Debugf("📡 请求订单明细: %s", urlStr)

	var respData []OrderItemsResponse
	for attempt := 1; attempt <= ItemMaxRetries; attempt++ {
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			global.Log.Warnf("⚠️ 请求失败 (第 %d 次): %v", attempt, err)
			if attempt < ItemMaxRetries {
				time.Sleep(ItemBaseBackoff * time.Duration(1<<(attempt-1)))
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 422 {

			if len(orderIDs) == 1 {
				global.Log.Errorf("❌ 单订单 422，无法拉取: %s, %s", orderIDs[0], string(body))
				return nil, fmt.Errorf("422 for order %s", orderIDs[0])
			}

			global.Log.Warnf("⚠️ 422 批次失败，拆分订单重试: %v", orderIDs)

			var all []models.OrderItem
			for _, id := range orderIDs {
				time.Sleep(2 * time.Second)
				single, err := fetchOrderItems(client, []string{id}, token)
				if err != nil {
					global.Log.Errorf("❌ 单订单仍失败，跳过: %s, %v", id, err)
					continue
				}
				all = append(all, single...)
			}
			return all, nil
		}

		if resp.StatusCode != http.StatusOK {
			global.Log.Warnf("❌ API 错误 %d: %s", resp.StatusCode, string(body))
			if attempt < ItemMaxRetries {
				time.Sleep(ItemBaseBackoff * time.Duration(1<<(attempt-1)))
				continue
			}
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, &respData); err != nil {
			return nil, fmt.Errorf("解析失败: %w, 响应: %s", err, string(body))
		}

		break
	}

	// 转换为模型
	var items []models.OrderItem
	for _, r := range respData {
		for _, item := range r.Items {

			// 解析 CreatedAt
			var parsedTimeC time.Time
			if strC := item.CreatedAt; strC != "" {
				var err error
				parsedTimeC, err = time.Parse(time.RFC3339, strC)
				if err != nil {
					log.Printf("CreatedAt 时间解析失败，使用默认零值: %v, 值: %s", err, strC)
					parsedTimeC = time.Time{} // 默认零值（0001-01-01 00:00:00）
					// 或者使用 time.Now() 作为默认值？
					// parsedTimeC = time.Now()
				}
			} else {
				log.Printf("CreatedAt 为空，使用默认零值")
				parsedTimeC = time.Time{} // 或 time.Now()
			}

			// 解析 UpdatedAt
			var parsedTimeU time.Time
			if strU := item.UpdatedAt; strU != "" {
				var err error
				parsedTimeU, err = time.Parse(time.RFC3339, strU)
				if err != nil {
					log.Printf("UpdatedAt 时间解析失败，使用默认零值: %v, 值: %s", err, strU)
					parsedTimeU = time.Time{}
				}
			} else {
				log.Printf("UpdatedAt 为空，使用默认零值")
				parsedTimeU = time.Time{}
			}

			items = append(items, models.OrderItem{
				ID:                  item.ID,
				OrderID:             r.OrderID,
				OrderNumber:         r.OrderNumber,
				Status:              item.Status,
				TrackingNumber:      item.TrackingNumber,
				TrackingURL:         item.TrackingURL,
				ShipmentType:        item.ShipmentType,
				DeliveryOption:      item.DeliveryOption,
				IsFulfilledByJumia:  item.IsFulfilledByJumia,
				ShopID:              item.ShopID,
				ItemPrice:           item.ItemPrice,
				PaidPrice:           item.PaidPrice,
				ShippingAmount:      item.ShippingAmount,
				ItemPriceLocal:      item.ItemPriceLocal,
				PaidPriceLocal:      item.PaidPriceLocal,
				ShippingAmountLocal: item.ShippingAmountLocal,
				ExchangeRate:        item.ExchangeRate,
				TaxAmount:           item.TaxAmount,
				VoucherAmount:       item.VoucherAmount,
				CountryCode:         item.Country.Code,
				CountryName:         item.Country.Name,
				CountryCurrency:     item.Country.CurrencyCode,
				ProductName:         item.Product.Name,
				SellerSKU:           item.Product.SellerSku,
				ImageURL:            item.Product.ImageURL,
				ShippingFirstName:   item.ShippingAddress.FirstName,
				ShippingLastName:    item.ShippingAddress.LastName,
				ShippingAddress:     item.ShippingAddress.Address,
				ShippingCity:        item.ShippingAddress.City,
				ShippingPostalCode:  item.ShippingAddress.PostalCode,
				ShippingWard:        item.ShippingAddress.Ward,
				ShippingRegion:      item.ShippingAddress.Region,
				ShippingCountryName: item.ShippingAddress.CountryName,
				CreatedAt:           parsedTimeC,
				UpdatedAt:           parsedTimeU,
			})
		}
	}

	global.Log.Debugf("📥 拉取到 %d 个明细项", len(items))
	return items, nil
}

// flushOrderItemsBatch 批量 Upsert 写入
// flushOrderItemsBatch 批量 Upsert 写入（自动分批，避免占位符超限）
func flushOrderItemsBatch(items []models.OrderItem) error {
	if len(items) == 0 {
		return nil
	}

	// ✅ 定义每批最大条数（安全值）
	const batchSize = 500
	total := len(items)

	global.Log.Infof("🔄 开始批量 Upsert 订单明细，总数: %d，每批: %d", total, batchSize)

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batch := items[i:end]
		global.Log.Infof("📦 处理批次 [%d:%d]，共 %d 条", i, end, len(batch))

		// 执行单批次 Upsert
		result := global.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"status":                gorm.Expr("VALUES(status)"),
				"tracking_number":       gorm.Expr("VALUES(tracking_number)"),
				"tracking_url":          gorm.Expr("VALUES(tracking_url)"),
				"shipment_type":         gorm.Expr("VALUES(shipment_type)"),
				"delivery_option":       gorm.Expr("VALUES(delivery_option)"),
				"is_fulfilled_by_jumia": gorm.Expr("VALUES(is_fulfilled_by_jumia)"),
				"paid_price":            gorm.Expr("VALUES(paid_price)"),
				"shipping_amount":       gorm.Expr("VALUES(shipping_amount)"),
				"paid_price_local":      gorm.Expr("VALUES(paid_price_local)"),
				"shipping_amount_local": gorm.Expr("VALUES(shipping_amount_local)"),
				"tax_amount":            gorm.Expr("VALUES(tax_amount)"),
				"voucher_amount":        gorm.Expr("VALUES(voucher_amount)"),
				"product_name":          gorm.Expr("VALUES(product_name)"),
				"image_url":             gorm.Expr("VALUES(image_url)"),
				"shipping_address":      gorm.Expr("VALUES(shipping_address)"),
				"shipping_city":         gorm.Expr("VALUES(shipping_city)"),
				"shipping_postal_code":  gorm.Expr("VALUES(shipping_postal_code)"),
				"updated_at":            gorm.Expr("VALUES(updated_at)"),
				"jumia_sku":             gorm.Expr("VALUES(jumia_sku)"),
				// 其他字段...
			}),
		}).Create(&batch) // ← 传入的是 batch，不是全部 items

		if result.Error != nil {
			global.Log.Errorf("❌ 批次 [%d:%d] Upsert 失败: %v", i, end, result.Error)

			// 特别记录是否是占位符问题
			if strings.Contains(result.Error.Error(), "placeholders") {
				global.Log.Warnf("💡 建议将 batchSize 从 %d 进一步减小（如 500）", batchSize)
			}
			return fmt.Errorf("批次 [%d:%d] Upsert 失败: %w", i, end, result.Error)
		}

		global.Log.Infof("✅ 批次 [%d:%d] 成功，影响行数=%d", i, end, result.RowsAffected)
	}

	return nil
}

// enrichWithJumiaSKU 批量填充 JumiaSKU
func enrichWithJumiaSKU(items []models.OrderItem) error {
	if len(items) == 0 {
		global.Log.Infof("🟡 enrichWithJumiaSKU: 输入 items 为空，跳过处理")
		return nil
	}

	global.Log.Infof("🔍 开始填充 JumiaSKU，共 %d 个订单项", len(items))

	// 定义匹配 Key
	type Key struct {
		SellerSKU   string
		CountryName string
	}
	keySet := make(map[Key]bool)
	var keys []Key

	// === Step 1: 提取唯一 Key（带标准化）===
	global.Log.Infof("📦 提取 SellerSKU + CountryName 唯一键")
	for i, item := range items {

		sku := strings.TrimSpace(item.SellerSKU)
		country := strings.TrimSpace(item.CountryName)

		// 打印原始值和处理后值（用于对比）
		if item.SellerSKU != sku {
			global.Log.Debugf("✂️  SellerSKU 去空格: '%s' → '%s'", item.SellerSKU, sku)
		}
		if item.CountryName != country {
			global.Log.Debugf("✂️  CountryName 去空格: '%s' → '%s'", item.CountryName, country)
		}
		// 使用：
		key := Key{
			SellerSKU:   normalize(sku),
			CountryName: normalize(country),
		}
		if !keySet[key] {
			keySet[key] = true
			keys = append(keys, key)
			//global.Log.Infof("🔑 新增唯一 Key: (SellerSKU='%s', Country='%s')", sku, country)
		} else {
			global.Log.Debugf("🔁 Key 已存在，跳过: (SellerSKU='%s', Country='%s')", sku, country)
		}

		// 可选：打印前几条详细信息
		if i < 3 {
			//global.Log.Debugf("📋 Item[%d]: SellerSKU='%s'(len=%d), CountryName='%s'(len=%d)",
			//	i, sku, len(sku), country, len(country))
		}
	}

	global.Log.Infof("✅ 共提取 %d 个唯一 Key，准备查询数据库", len(keys))
	//for _, k := range keys {
	//	//global.Log.Debugf("🔍 查询 Key: SellerSKU='%s'(len=%d), CountryName='%s'(len=%d)",
	//	//	k.SellerSKU, len(k.SellerSKU), k.CountryName, len(k.CountryName))
	//}

	// === Step 2: 查询 UserProduct ===
	var products []models.UserProduct
	if len(keys) == 0 {
		global.Log.Warn("⚠️ 未提取到任何 Key，跳过数据库查询")
	} else {
		var conditions []string
		var args []interface{}
		for _, k := range keys {
			conditions = append(conditions, "(seller_sku = ? AND country_name = ?)")
			args = append(args, k.SellerSKU, k.CountryName)
		}

		// 🔥 开启 Debug 查看真实 SQL（生产环境可关闭）
		err := global.DB.
			// Debug(). // ← 取消注释可查看 SQL
			Select("seller_sku, country_name, jumia_sku").
			Where(strings.Join(conditions, " OR "), args...).
			Find(&products).Error

		if err != nil {
			global.Log.Errorf("❌ 查询 UserProduct 失败: %v", err)
			return fmt.Errorf("查询 UserProduct 失败: %w", err)
		}

		global.Log.Infof("✅ 数据库返回 %d 条 UserProduct 记录", len(products))
		for i, p := range products {
			sku := strings.TrimSpace(p.SellerSku)
			country := strings.TrimSpace(p.CountryName)
			global.Log.Infof("📥 [%d] 匹配到: SellerSku='%s'(len=%d), CountryName='%s'(len=%d) → JumiaSku='%s'",
				i, sku, len(sku), country, len(country), p.JumiaSku)
		}
	}

	// === Step 3: 构建映射表 ===
	skuMap := make(map[Key]string)
	for _, p := range products {
		sku := strings.TrimSpace(p.SellerSku)
		country := strings.TrimSpace(p.CountryName)
		key := Key{SellerSKU: sku, CountryName: country}
		skuMap[key] = p.JumiaSku
		//global.Log.Infof("🔗 建立映射: (SellerSku='%s', Country='%s') → JumiaSku='%s'",
		//	sku, country, p.JumiaSku)
	}

	// === Step 4: 填充 OrderItem ===
	successCount := 0
	failCount := 0
	for i := range items {
		originalSKU := items[i].SellerSKU
		originalCountry := items[i].CountryName

		sku := strings.TrimSpace(originalSKU)
		country := strings.TrimSpace(originalCountry)
		key := Key{SellerSKU: sku, CountryName: country}

		if jumiaSku, found := skuMap[key]; found {
			items[i].JumiaSKU = jumiaSku
			//global.Log.Infof("✅ [%d] 填充成功: '%s' + '%s' → %s",
			//	i, originalSKU, originalCountry, jumiaSku)
			successCount++
		} else {
			items[i].JumiaSKU = ""
			global.Log.Warnf("❌ [%d] 填充失败: 未找到匹配 (SellerSKU='%s'(len=%d), Country='%s'(len=%d))",
				i, sku, len(sku), country, len(country))
			failCount++
		}
	}

	global.Log.Infof("🎉 JumiaSKU 填充完成: 成功=%d, 失败=%d, 总数=%d",
		successCount, failCount, len(items))

	return nil
}
func normalize(s string) string {
	return strings.TrimSpace(strings.ToUpper(s))
}
func enrichWithDate(items []models.OrderItem) error {
	if len(items) == 0 {
		return nil // 空列表直接返回
	}

	// 1. 提取所有唯一的 OrderNumber
	orderNumbers := make(map[string]bool)
	var numbers []string
	for _, item := range items {
		if !orderNumbers[item.OrderNumber] {
			orderNumbers[item.OrderNumber] = true
			numbers = append(numbers, item.OrderNumber)
		}
	}

	// 2. 从数据库查询匹配的 Order 记录
	var orders []models.Order
	err := global.DB.
		Select("number, created_at, updated_at").
		Where("number IN ?", numbers).
		Find(&orders).Error

	if err != nil {
		return fmt.Errorf("查询 Order 表失败: %w", err)
	}

	// 3. 构建 map[orderNumber]Order 用于快速查找
	orderMap := make(map[string]models.Order)
	for _, order := range orders {
		orderMap[order.Number] = order
	}

	// 4. 遍历 items，填充 CreatedAt 和 UpdatedAt
	for i := range items {
		if order, found := orderMap[items[i].OrderNumber]; found {
			items[i].CreatedAt = order.CreatedAt
			items[i].UpdatedAt = order.UpdatedAt
		}
		// 如果没找到匹配的 Order，可以选择保留原值或设为空
		//global.Log.Info("插入时间失败")
	}
	global.Log.Info("插入时间成功")
	return nil
}
