package cron_ser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	pageSize       = 100  // 每页 API 返回数量
	batchSize      = 1000 // 每批处理的产品数
	maxRetries     = 3    // 每页最大重试次数
	requestDelay   = 800 * time.Millisecond
	baseBackoff    = 2 * time.Second
	queryChunkSize = 500 // 查询 IN 分片大小
	writeChunkSize = 500 // 写入分片大小
)

// SyncStats 用于返回同步统计信息
type SyncStats struct {
	Fetched int // 从 API 拉取的原始产品总数（按 product 计）
	Skipped int // 因为空 jumia_sku 被跳过的数量
	Created int // 实际创建的数量
	Updated int // 实际更新的数量

	SuccessCount int
	FailureCount int
	TotalCount   int
}

func SyncJumiaProduct() {
	SyncUserProducts()
	SyncUserProductsTwo()
}
func SyncUserProducts() {
	var countBefore int64
	global.DB.Model(&models.UserProduct{}).Count(&countBefore)
	global.Log.Infof("同步前数据库中已有 %d 条 user_products 记录", countBefore)

	global.Log.Info("开始定时同步用户产品数据（优化版）")
	// 接收统计信息
	stats, err := FetchAndSyncProductsInBatches(global.Config.Jumia.AccessToken, "1")
	if err != nil {
		global.DB.Where("id = ?", 3).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Errorf("同步失败: %v", err)
		return
	}

	var countAfter int64
	global.DB.Model(&models.UserProduct{}).Count(&countAfter)

	// ✅ 打印最终汇总日志
	global.Log.Infof("✅ 同步完成汇总:")
	global.Log.Infof("   - 从 API 获取产品总数: %d", stats.Fetched*6)
	global.Log.Infof("   - 因 jumia_sku 为空跳过: %d", stats.Skipped)
	global.Log.Infof("   - 新增记录数: %d", stats.Created)
	global.Log.Infof("   - 更新记录数: %d", stats.Updated)
	global.Log.Infof("   - 数据库总条数 (同步后): %d", countAfter)
	global.DB.Where("id = ?", 3).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步用户产品成功",
	})
}
func SyncUserProductsTwo() {
	var countBefore int64
	global.DB.Model(&models.UserProduct{}).Count(&countBefore)
	global.Log.Infof("同步前数据库中已有 %d 条 user_products 记录", countBefore)

	global.Log.Info("开始定时同步用户产品数据（优化版）")
	// 接收统计信息
	stats, err := FetchAndSyncProductsInBatches(global.Config.JumiaTwo.AccessToken, "2")
	if err != nil {
		global.DB.Where("id = ?", 3).Updates(models.ServiceStatus{
			Status:  "错误",
			Message: err.Error(),
		})
		global.Log.Errorf("同步失败: %v", err)
		return
	}

	var countAfter int64
	global.DB.Model(&models.UserProduct{}).Count(&countAfter)

	// ✅ 打印最终汇总日志
	global.Log.Infof("✅ 同步完成汇总:")
	global.Log.Infof("   - 从 API 获取产品总数: %d", stats.Fetched*6)
	global.Log.Infof("   - 因 jumia_sku 为空跳过: %d", stats.Skipped)
	global.Log.Infof("   - 新增记录数: %d", stats.Created)
	global.Log.Infof("   - 更新记录数: %d", stats.Updated)
	global.Log.Infof("   - 数据库总条数 (同步后): %d", countAfter)
	global.DB.Where("id = ?", 3).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步用户产品成功",
	})
}

// ProductResponse 同前
type ProductResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Variations []struct {
		SellerSku       string `json:"sellerSku"`
		BusinessClients []struct {
			Sku         string `json:"sku"`
			CountryName string `json:"countryName"`
			Price       struct {
				LocalCurrency string  `json:"localCurrency"`
				LocalValue    float64 `json:"localValue"`
				Currency      string  `json:"currency"`
				Value         float64 `json:"value"`
				SalePrice     struct {
					LocalValue float64   `json:"localValue"`
					Value      float64   `json:"value"`
					StartAt    time.Time `json:"startAt"`
					EndAt      time.Time `json:"endAt"`
				} `json:"salePrice"`
			} `json:"price"`
		} `json:"businessClients"`
	} `json:"variations"`
}

type ProductResponsePage struct {
	Products   []ProductResponse `json:"products"`
	NextToken  *string           `json:"nextToken"`
	IsLastPage bool              `json:"isLastPage"`
}

func FetchAndSyncProductsInBatches(token, account string) (*SyncStats, error) {
	const baseURL = "https://vendor-api.jumia.com/catalog/products"

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
		buffer       []models.UserProduct
	)

	global.Log.Info("开始分页拉取并分批同步产品数据...")

	for {
		totalPages++
		url := fmt.Sprintf("%s?size=%d", baseURL, pageSize)
		if nextToken != "" {
			url = fmt.Sprintf("%s?token=%s", baseURL, nextToken)
		}

		var page ProductResponsePage
		var err error

		// 重试机制
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = requestPage(client, url, &page, token)
			if err == nil {
				break
			}

			if attempt < maxRetries {
				backoff := baseBackoff * time.Duration(1<<(attempt-1))
				global.Log.Warnf("请求失败 (第 %d 次): %v, %v 后重试: %s", attempt, err, backoff, url)
				time.Sleep(backoff)
			}
		}

		if err != nil {
			return nil, fmt.Errorf("拉取第 %d 页失败: %w", totalPages, err)
		}

		if len(page.Products) == 0 {
			global.Log.Info("当前页无产品数据")
		} else {
			global.Log.Debugf("第 %d 页: 拉取到 %d 个产品", totalPages, len(page.Products))
		}

		// ✅ 转换并过滤空 SKU（从源头控制）
		products := convertToUserProducts(page.Products, account)
		skippedCount := 0
		for _, p := range page.Products {
			for _, v := range p.Variations {
				for _, c := range v.BusinessClients {
					if c.Sku == "" {
						skippedCount++
					}
				}
			}
		}

		totalFetched += len(page.Products)
		totalSkipped += skippedCount
		buffer = append(buffer, products...)

		// 批量处理
		if len(buffer) >= batchSize {
			batchStats, err := flushBatch(buffer[:batchSize])
			if err != nil {
				return nil, fmt.Errorf("批量写入失败: %w", err)
			}
			totalCreated += batchStats.Created
			totalUpdated += batchStats.Updated
			buffer = buffer[batchSize:]
		}

		// 结束判断
		if page.IsLastPage || page.NextToken == nil {
			break
		}
		nextToken = *page.NextToken

		time.Sleep(requestDelay)
	}

	// 处理剩余
	if len(buffer) > 0 {
		batchStats, err := flushBatch(buffer)
		if err != nil {
			return nil, fmt.Errorf("最终批次写入失败: %w", err)
		}
		totalCreated += batchStats.Created
		totalUpdated += batchStats.Updated
	}

	global.Log.Infof("同步完成：共 %d 页，处理 %d 个原始产品", totalPages, totalFetched)

	return &SyncStats{
		Fetched: totalFetched,
		Skipped: totalSkipped,
		Created: totalCreated,
		Updated: totalUpdated,
	}, nil
}

// ✅ convertToUserProducts：从源头过滤空 Sku
func convertToUserProducts(products []ProductResponse, account string) []models.UserProduct {
	var result []models.UserProduct
	for _, p := range products {
		for _, v := range p.Variations {
			for _, c := range v.BusinessClients {
				// ✅ 严格过滤空值
				if c.Sku == "" {
					//global.Log.Warnf("跳过空 JumiaSku: SellerSku=%s, Country=%s, ProductName=%s", v.SellerSku, c.CountryName, p.Name)
					continue
				}

				if c.CountryName == "Senegal" || c.CountryName == "Ivory-Coast" || c.CountryName == "Uganda" {
					//跳过特定国家的数据
					//global.Log.Warnf("跳过空 CountryName: Sku=%s, SellerSku=%s, ProductName=%s", c.Sku, v.SellerSku, p.Name)
					continue
				}

				saleStartAt := parseTimePtr(c.Price.SalePrice.StartAt)
				saleEndAt := parseTimePtr(c.Price.SalePrice.EndAt)

				result = append(result, models.UserProduct{
					NameEn:          p.Name,
					SellerSku:       v.SellerSku,
					JumiaSku:        c.Sku,
					CountryName:     c.CountryName,
					LocalCurrency:   c.Price.LocalCurrency,
					LocalPriceValue: c.Price.LocalValue,
					PriceCurrency:   c.Price.Currency,
					PriceValue:      c.Price.Value,
					SaleLocalValue:  &c.Price.SalePrice.LocalValue,
					SaleValue:       &c.Price.SalePrice.Value,
					SaleStartAt:     saleStartAt,
					SaleEndAt:       saleEndAt,
					Account:         account,
				})
			}
		}
	}
	return result
}

// FlushBatchStats 返回本次 flush 的统计
// FlushBatchStats 返回批量操作统计
type FlushBatchStats struct {
	Created int
	Updated int
}

// ✅ flushBatch：返回创建和更新数量
func flushBatch(products []models.UserProduct) (*FlushBatchStats, error) {
	if len(products) == 0 {
		return &FlushBatchStats{}, nil
	}

	global.Log.Debugf("flushBatch: 开始处理 %d 个产品", len(products))

	// ✅ 提取 jumia_sku 并建立映射
	var jumiaSkus []string
	skuMap := make(map[string]models.UserProduct, len(products))
	for _, p := range products {
		jumiaSkus = append(jumiaSkus, p.JumiaSku)
		skuMap[p.JumiaSku] = p
	}

	if len(jumiaSkus) == 0 {
		global.Log.Warn("该批次无有效 jumia_sku（理论上不应发生）")
		return &FlushBatchStats{}, nil
	}

	global.Log.Debugf("准备查询数据库: %d 个唯一 jumia_sku", len(jumiaSkus))

	// ✅ 分片查询 existing 记录
	var existing []models.UserProduct
	for i := 0; i < len(jumiaSkus); i += queryChunkSize {
		end := i + queryChunkSize
		if end > len(jumiaSkus) {
			end = len(jumiaSkus)
		}
		chunk := jumiaSkus[i:end]

		var part []models.UserProduct
		err := global.DB.Where("jumia_sku IN ?", chunk).Find(&part).Error
		if err != nil {
			return nil, fmt.Errorf("分片查询失败: %w", err)
		}
		global.Log.Debugf("查询 chunk [%d:%d]: %d 个 SKU, 返回 %d 条记录", i, end, len(chunk), len(part))
		existing = append(existing, part...)
	}

	global.Log.Debugf("总计从数据库查到 %d 条现有记录", len(existing))

	// ✅ 构建 existingMap 用于快速判断是否存在
	existingMap := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		existingMap[p.JumiaSku] = struct{}{}
	}

	// ✅ 分类：创建 or 更新
	var toCreate, toUpdate []models.UserProduct
	for _, p := range products {
		if _, exists := existingMap[p.JumiaSku]; exists {
			toUpdate = append(toUpdate, p)
		} else {
			toCreate = append(toCreate, p)
		}
	}

	global.Log.Infof("批量处理: 创建 %d, 更新 %d", len(toCreate), len(toUpdate))

	// ✅ 在事务中执行写入
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		// ✅ 用于全局去重 SellerSku（避免同一批次重复绑定）
		createdSellerSkus := make(map[string]struct{})

		// ✅ 分片创建
		for i := 0; i < len(toCreate); i += writeChunkSize {
			end := i + writeChunkSize
			if end > len(toCreate) {
				end = len(toCreate)
			}

			chunk := toCreate[i:end]
			// 🔥 在 Create 之前：为每个产品设置 UserID 和 UserName
			for idx := range chunk {
				chunk[idx].UserID = 1
				chunk[idx].UserName = "admin" // 建议从配置或上下文获取，而非硬编码
			}

			// 1. 批量创建产品
			if err := tx.Create(chunk).Error; err != nil {
				return fmt.Errorf("创建产品失败: %w", err)
			}

			// 2. 构造 UserSellerSkuModel 绑定（去重）
			var skuBindings []models.UserSellerSkuModel
			for _, product := range chunk {
				sellerSku := product.SellerSku
				// 只有未处理过的 SellerSku 才添加
				if _, exists := createdSellerSkus[sellerSku]; !exists {
					createdSellerSkus[sellerSku] = struct{}{}
					skuBindings = append(skuBindings, models.UserSellerSkuModel{
						UserID:    1, // 固定绑定 user_id = 1
						SellerSku: sellerSku,
					})
				}
			}

			// 3. 批量插入绑定关系（忽略主键冲突）
			if len(skuBindings) > 0 {
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "user_id"}, {Name: "seller_sku"}}, // 复合主键
					DoNothing: true,                                                     // 有冲突则跳过
				}).Create(&skuBindings).Error; err != nil {
					return fmt.Errorf("创建用户 SKU 绑定失败: %w", err)
				}
				global.Log.Debugf("已绑定 %d 个新的 SellerSku 给 user_id=1", len(skuBindings))
			}
		}

		// ✅ 分片更新（upsert 指定字段）
		for i := 0; i < len(toUpdate); i += writeChunkSize {
			end := i + writeChunkSize
			if end > len(toUpdate) {
				end = len(toUpdate)
			}
			chunk := toUpdate[i:end]

			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "jumia_sku"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"name_en":           gorm.Expr("VALUES(name_en)"),
					"seller_sku":        gorm.Expr("VALUES(seller_sku)"),
					"country_name":      gorm.Expr("VALUES(country_name)"),
					"local_currency":    gorm.Expr("VALUES(local_currency)"),
					"local_price_value": gorm.Expr("VALUES(local_price_value)"),
					"price_currency":    gorm.Expr("VALUES(price_currency)"),
					"price_value":       gorm.Expr("VALUES(price_value)"),
					"sale_local_value":  gorm.Expr("VALUES(sale_local_value)"),
					"sale_value":        gorm.Expr("VALUES(sale_value)"),
					"sale_start_at":     gorm.Expr("VALUES(sale_start_at)"),
					"sale_end_at":       gorm.Expr("VALUES(sale_end_at)"),
					"account":           gorm.Expr("VALUES(account)"),
					// ❌ 不更新字段：name_zh, inventory, buy_url, sell_url
				}),
			}).Create(chunk).Error; err != nil {
				return fmt.Errorf("更新失败: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &FlushBatchStats{
		Created: len(toCreate),
		Updated: len(toUpdate),
	}, nil
}

func requestPage(client *http.Client, url string, target *ProductResponsePage, token string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 错误 %d: %.200s", resp.StatusCode, body)
	}

	return json.Unmarshal(body, target)
}

func parseTimePtr(t time.Time) *time.Time {
	if t.IsZero() || t.Year() <= 1 {
		return nil
	}
	return &t
}
