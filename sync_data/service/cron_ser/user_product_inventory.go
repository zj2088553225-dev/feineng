package cron_ser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"sync_data/global"
	"sync_data/models"
)

// --- 1. 新的 API 响应结构 ---
type StockDetail struct {
	Sid            string `json:"sid"`
	Sku            string `json:"sku"`
	Name           string `json:"name"`
	AvailableUnits int    `json:"availableUnits"` // 实际可用库存
	BusinessClient struct {
		Name        string `json:"name"`        // 国家名称，如 "Ghana"
		Code        string `json:"code"`        // 如 "jumia-gh"
		CountryCode string `json:"countryCode"` // "GH"
	} `json:"businessClient"`
	NonSellableResponseDTO struct {
		Quarantine int `json:"quarantine"`
		Defective  int `json:"defective"`
		Picked     int `json:"picked"`
		Packed     int `json:"packed"`
		Shipped    int `json:"shipped"`
	} `json:"nonSellableResponseDTO"`
}

type StockResponse struct {
	Content          []StockDetail `json:"content"`
	TotalPages       int           `json:"totalPages"`
	TotalElements    int64         `json:"totalElements"`
	NumberOfElements int           `json:"numberOfElements"`
	Last             bool          `json:"last"`
	First            bool          `json:"first"`
	Size             int           `json:"size"`
	Page             int           `json:"number"` // 当前页码
}

// --- 2. 统计结构 ---
type SyncResult struct {
	UpdatedCount  int
	NotFoundCount int
	ErrorCount    int
	CountryName   string
	Duration      time.Duration
	Err           error
}

func SyncJumiaProductInventory() {
	SyncUserProductInventory()
	SyncUserProductInventoryTwo()
}

// --- 3. 主函数 ---
func SyncUserProductInventory() {
	start := time.Now()
	global.Log.Info("【产品库存同步】开始同步 Jumia 库存数据（使用 consignment API）...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]SyncResult)

	// 固定 businessClientCode 和 shopSid（可配置）
	businessClientCodes := []string{"jumia-gh", "jumia-ke", "jumia-ng"}
	shopSid := "fb9d1d71-9f02-489b-9930-df0e80a4ba53"

	for _, bCode := range businessClientCodes {
		wg.Add(1)
		go func(bCode string) {
			defer wg.Done()
			startTime := time.Now()
			result := syncInventoryFromStockDetails(global.Config.Jumia.JumiaCenterToken, bCode, shopSid, "1")
			result.Duration = time.Since(startTime)
			mu.Lock()
			results[result.CountryName] = result
			mu.Unlock()
		}(bCode)

		time.Sleep(200 * time.Millisecond) // 避免并发过猛
	}

	wg.Wait()

	// 打印汇总
	global.Log.Info("【产品库存同步】所有国家同步完成，汇总结果：")
	totalUpdated, totalNotFound, totalErrors := 0, 0, 0

	for country, res := range results {
		if res.Err != nil {
			global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
				Status:  "错误",
				Message: res.Err.Error(),
			})
			global.Log.Error("❌ [%s] 同步失败: %v (耗时: %v)", country, res.Err, res.Duration)
			totalErrors++
			continue
		}
		global.Log.Info(fmt.Sprintf("✅ [%s] 更新: %d 条 | 未找到: %d 条 | 耗时: %v",
			country, res.UpdatedCount, res.NotFoundCount, res.Duration))
		totalUpdated += res.UpdatedCount
		totalNotFound += res.NotFoundCount
	}

	global.Log.Info(fmt.Sprintf("🏁 总结：共更新 %d 条库存，%d 条 SKU 未匹配，%d 个国家失败，总耗时: %v",
		totalUpdated, totalNotFound, totalErrors, time.Since(start)))
	global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步用户产品库存成功",
	})
}

// --- 3. 主函数 ---
func SyncUserProductInventoryTwo() {
	start := time.Now()
	global.Log.Info("【产品库存同步】开始同步 Jumia 库存数据（使用 consignment API）...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]SyncResult)

	// 固定 businessClientCode 和 shopSid（可配置）
	businessClientCodes := []string{"jumia-gh", "jumia-ke", "jumia-ng"}
	shopSid := "f8162f4a-2ccb-4f2d-a711-acce9d5cf4a0"

	for _, bCode := range businessClientCodes {
		wg.Add(1)
		go func(bCode string) {
			defer wg.Done()
			startTime := time.Now()
			result := syncInventoryFromStockDetails(global.Config.JumiaTwo.JumiaCenterToken, bCode, shopSid, "2")
			result.Duration = time.Since(startTime)
			mu.Lock()
			results[result.CountryName] = result
			mu.Unlock()
		}(bCode)

		time.Sleep(200 * time.Millisecond) // 避免并发过猛
	}

	wg.Wait()

	// 打印汇总
	global.Log.Info("【产品库存同步】所有国家同步完成，汇总结果：")
	totalUpdated, totalNotFound, totalErrors := 0, 0, 0

	for country, res := range results {
		if res.Err != nil {
			global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
				Status:  "错误",
				Message: res.Err.Error(),
			})
			global.Log.Errorf("❌ [%s] 同步失败: %v (耗时: %v)", country, res.Err, res.Duration)
			totalErrors++
			continue
		}
		global.Log.Info(fmt.Sprintf("✅ [%s] 更新: %d 条 | 未找到: %d 条 | 耗时: %v",
			country, res.UpdatedCount, res.NotFoundCount, res.Duration))
		totalUpdated += res.UpdatedCount
		totalNotFound += res.NotFoundCount
	}

	global.Log.Info(fmt.Sprintf("🏁 总结：共更新 %d 条库存，%d 条 SKU 未匹配，%d 个国家失败，总耗时: %v",
		totalUpdated, totalNotFound, totalErrors, time.Since(start)))
	global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步用户产品库存成功",
	})
}

// --- 5. 从 stock-details API 同步库存 ---
func syncInventoryFromStockDetails(token, businessClientCode, shopSid, account string) SyncResult {
	result := SyncResult{}
	client := &http.Client{Timeout: 30 * time.Second}
	page := 0
	var totalCount int

	// 存储 SKU → 库存映射
	skuToInventory := make(map[string]int)
	var countryName string

	global.Log.Info(fmt.Sprintf("🌍 开始同步 businessClient: %s", businessClientCode))

	// --- Step 1: 分页拉取 stock-details ---
	for {
		url := fmt.Sprintf(
			"https://api-consignment-services.jumia.com/api/stock-details?size=20&page=%d&businessClientCode=%s&shopSid=%s",
			page, businessClientCode, shopSid,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			result.Err = fmt.Errorf("创建请求失败: %w", err)
			return result
		}

		req.Header.Set("accept", "application/json, text/plain, */*")
		req.Header.Set("accept-language", "en")
		req.Header.Set("authorization", "Bearer "+token)
		req.Header.Set("origin", "https://vendorcenter.jumia.com")
		req.Header.Set("priority", "u=1, i")
		req.Header.Set("referer", "https://vendorcenter.jumia.com/")
		req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-site")
		req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
		req.Header.Set("x_master_shop_sid", shopSid)
		req.Header.Set("x_shop_sid_list", shopSid)

		resp, err := client.Do(req)
		if err != nil {
			result.Err = fmt.Errorf("请求 API 失败: %w", err)
			return result
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result.Err = fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(bodyBytes))
			return result
		}

		var apiResp StockResponse
		if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
			result.Err = fmt.Errorf("解析 JSON 失败: %w", err)
			return result
		}

		// 第一次请求时确定国家名称
		if len(apiResp.Content) > 0 && countryName == "" {
			countryName = apiResp.Content[0].BusinessClient.Name
			result.CountryName = countryName
		}

		// 提取库存
		for _, item := range apiResp.Content {
			skuToInventory[item.Sku] = item.AvailableUnits
		}
		totalCount += len(apiResp.Content)

		global.Log.Info(fmt.Sprintf("✅ [%s] 已加载第 %d 页，新增 %d 个，累计 %d 个",
			businessClientCode, page, len(apiResp.Content), totalCount))

		if apiResp.Last {
			break
		}
		page++
	}

	if len(skuToInventory) == 0 {
		//result.Err = fmt.Errorf("无 SKU 数据可同步")
		global.Log.Info("无 SKU 数据可同步")
		return result
	}

	global.Log.Info(fmt.Sprintf("📥 [%s] API 拉取完成：共 %d 页，%d 个唯一 SKU", countryName, page+1, len(skuToInventory)))

	// --- Step 2: 批量查询数据库 ---
	global.Log.Info(fmt.Sprintf("🔍 [%s] 查询数据库匹配 %d 个 SKU...", countryName, len(skuToInventory)))

	var userProducts []models.UserProduct
	if err := global.DB.
		Where("country_name = ? AND seller_sku IN ? AND account = ?", countryName, getKeys(skuToInventory), account).
		Find(&userProducts).Error; err != nil {
		result.Err = fmt.Errorf("数据库查询失败: %w", err)
		return result
	}

	global.Log.Info(fmt.Sprintf("🎉 [%s] 匹配到 %d 个数据库记录", countryName, len(userProducts)))

	// 构建 map
	dbSkuMap := make(map[string]*models.UserProduct)
	for i := range userProducts {
		dbSkuMap[userProducts[i].SellerSku] = &userProducts[i]
	}

	// --- Step 3: 批量更新 ---
	tx := global.DB.Begin()

	var updates []models.UserProduct
	for sku, inv := range skuToInventory {
		if record, exists := dbSkuMap[sku]; exists {
			record.Inventory = inv
			updates = append(updates, *record)
		} else {
			result.NotFoundCount++
		}
	}

	if len(updates) > 0 {
		if err := tx.Save(&updates).Error; err != nil {
			tx.Rollback()
			result.Err = fmt.Errorf("批量更新失败: %w", err)
			return result
		}
		result.UpdatedCount = len(updates)
		global.Log.Info(fmt.Sprintf("✅ [%s] 批量更新提交成功，共 %d 条", countryName, len(updates)))
	} else {
		tx.Rollback()
		global.Log.Info(fmt.Sprintf("⚠️ [%s] 无数据需要更新", countryName))
	}

	tx.Commit()
	result.Err = nil
	return result
}

// --- 辅助函数 ---
func getKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- 3. 同步某个人的库存 ---
func SyncUserProductInventoryForone() {
	start := time.Now()
	global.Log.Info("【产品库存同步】开始同步 Jumia 库存数据（使用 consignment API）...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[string]SyncResult)

	// 固定 businessClientCode 和 shopSid（可配置）
	businessClientCodes := []string{"jumia-gh", "jumia-ke", "jumia-ng"}
	shopSid := "fb9d1d71-9f02-489b-9930-df0e80a4ba53"

	for _, bCode := range businessClientCodes {
		wg.Add(1)
		go func(bCode string) {
			defer wg.Done()
			startTime := time.Now()
			result := SyncInventoryFromStockDetailsForone(global.Config.Jumia.JumiaCenterToken, bCode, shopSid, "1", "杨涛")
			result.Duration = time.Since(startTime)
			mu.Lock()
			results[result.CountryName] = result
			mu.Unlock()
		}(bCode)

		time.Sleep(200 * time.Millisecond) // 避免并发过猛
	}

	wg.Wait()

	// 打印汇总
	global.Log.Info("【产品库存同步】所有国家同步完成，汇总结果：")
	totalUpdated, totalNotFound, totalErrors := 0, 0, 0

	for country, res := range results {
		if res.Err != nil {
			global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
				Status:  "错误",
				Message: res.Err.Error(),
			})
			global.Log.Error("❌ [%s] 同步失败: %v (耗时: %v)", country, res.Err, res.Duration)
			totalErrors++
			continue
		}
		global.Log.Info(fmt.Sprintf("✅ [%s] 更新: %d 条 | 未找到: %d 条 | 耗时: %v",
			country, res.UpdatedCount, res.NotFoundCount, res.Duration))
		totalUpdated += res.UpdatedCount
		totalNotFound += res.NotFoundCount
	}

	global.Log.Info(fmt.Sprintf("🏁 总结：共更新 %d 条库存，%d 条 SKU 未匹配，%d 个国家失败，总耗时: %v",
		totalUpdated, totalNotFound, totalErrors, time.Since(start)))
	global.DB.Where("id = ?", 4).Updates(models.ServiceStatus{
		Status:  "正常",
		Message: "定时同步用户产品库存成功",
	})
}

// --- 5. 从 stock-details API 同步库存 ---
func SyncInventoryFromStockDetailsForone(token, businessClientCode, shopSid, account, username string) SyncResult {
	result := SyncResult{}
	client := &http.Client{Timeout: 30 * time.Second}
	page := 0
	var totalCount int

	// 存储 SKU → 库存映射
	skuToInventory := make(map[string]int)
	var countryName string

	global.Log.Info(fmt.Sprintf("🌍 开始同步 businessClient: %s", businessClientCode))

	// --- Step 1: 分页拉取 stock-details ---
	for {
		url := fmt.Sprintf(
			"https://api-consignment-services.jumia.com/api/stock-details?size=20&page=%d&businessClientCode=%s&shopSid=%s",
			page, businessClientCode, shopSid,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			result.Err = fmt.Errorf("创建请求失败: %w", err)
			return result
		}

		req.Header.Set("accept", "application/json, text/plain, */*")
		req.Header.Set("accept-language", "en")
		req.Header.Set("authorization", "Bearer "+token)
		req.Header.Set("origin", "https://vendorcenter.jumia.com")
		req.Header.Set("priority", "u=1, i")
		req.Header.Set("referer", "https://vendorcenter.jumia.com/")
		req.Header.Set("sec-ch-ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("sec-fetch-dest", "empty")
		req.Header.Set("sec-fetch-mode", "cors")
		req.Header.Set("sec-fetch-site", "same-site")
		req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36")
		req.Header.Set("x_master_shop_sid", shopSid)
		req.Header.Set("x_shop_sid_list", shopSid)

		resp, err := client.Do(req)
		if err != nil {
			result.Err = fmt.Errorf("请求 API 失败: %w", err)
			return result
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result.Err = fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(bodyBytes))
			return result
		}

		var apiResp StockResponse
		if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
			result.Err = fmt.Errorf("解析 JSON 失败: %w", err)
			return result
		}

		// 第一次请求时确定国家名称
		if len(apiResp.Content) > 0 && countryName == "" {
			countryName = apiResp.Content[0].BusinessClient.Name
			result.CountryName = countryName
		}

		// 提取库存
		for _, item := range apiResp.Content {
			skuToInventory[item.Sku] = item.AvailableUnits
		}
		totalCount += len(apiResp.Content)

		global.Log.Info(fmt.Sprintf("✅ [%s] 已加载第 %d 页，新增 %d 个，累计 %d 个",
			businessClientCode, page, len(apiResp.Content), totalCount))

		if apiResp.Last {
			break
		}
		page++
	}

	if len(skuToInventory) == 0 {
		//result.Err = fmt.Errorf("无 SKU 数据可同步")
		global.Log.Info("无 SKU 数据可同步")
		return result
	}

	global.Log.Info(fmt.Sprintf("📥 [%s] API 拉取完成：共 %d 页，%d 个唯一 SKU", countryName, page+1, len(skuToInventory)))

	// --- Step 2: 批量查询数据库 ---
	global.Log.Info(fmt.Sprintf("🔍 [%s] 查询数据库匹配 %d 个 SKU...", countryName, len(skuToInventory)))

	var userProducts []models.UserProduct
	if err := global.DB.
		Where("country_name = ? AND seller_sku IN ? AND account = ? AND user_name = ?", countryName, getKeys(skuToInventory), account, username).
		Find(&userProducts).Error; err != nil {
		result.Err = fmt.Errorf("数据库查询失败: %w", err)
		return result
	}

	global.Log.Info(fmt.Sprintf("🎉 [%s] 匹配到 %d 个数据库记录", countryName, len(userProducts)))

	// 构建 map
	dbSkuMap := make(map[string]*models.UserProduct)
	for i := range userProducts {
		dbSkuMap[userProducts[i].SellerSku] = &userProducts[i]
	}

	// --- Step 3: 批量更新 ---
	tx := global.DB.Begin()

	var updates []models.UserProduct
	for sku, inv := range skuToInventory {
		if record, exists := dbSkuMap[sku]; exists {
			record.Inventory = inv
			updates = append(updates, *record)
		} else {
			result.NotFoundCount++
		}
	}

	if len(updates) > 0 {
		if err := tx.Save(&updates).Error; err != nil {
			tx.Rollback()
			result.Err = fmt.Errorf("批量更新失败: %w", err)
			return result
		}
		result.UpdatedCount = len(updates)
		global.Log.Info(updates)
		global.Log.Info(fmt.Sprintf("✅ [%s] 批量更新提交成功，共 %d 条", countryName, len(updates)))
	} else {
		tx.Rollback()
		global.Log.Info(fmt.Sprintf("⚠️ [%s] 无数据需要更新", countryName))
	}

	tx.Commit()
	result.Err = nil
	return result
}
