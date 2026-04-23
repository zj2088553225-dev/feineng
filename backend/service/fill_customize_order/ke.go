// fill_customize_order.go
package fill_customize_order

import (
	"backend/global" // 替换为您的实际项目路径
	"backend/models" // 替换为您的实际项目路径
	"fmt"
	"gorm.io/gorm/clause"
	"log"
	"strings"
	"time"
)

// FillAllCustomizeOrderGH 填充 CustomizeOrderGH 表的 Status, TrackingURL 和 Person
// 新逻辑：通过 JumiaSKU → UserProduct → SellerSku → UserSellerSkuModel → UserID → UserName
// batchSize: 每次处理的记录数量，建议 500-1000
func FillAllCustomizeOrderKE(batchSize int) (FillResult, error) {
	var result FillResult
	db := global.DB
	if db == nil {
		return result, fmt.Errorf("全局数据库连接 global.DB 为 nil")
	}
	if batchSize <= 0 {
		batchSize = 100
	}

	startTime := time.Now()
	var lastID int

	log.Println("[FillAllCustomizeOrderKE] 开始填充 CustomizeOrderKE 表数据...")

	for {
		// 1. 分页查询 CustomizeOrderGH
		var currentBatch []struct {
			KEID        int    `gorm:"column:ke_id"`
			OrderNumber string `gorm:"column:order_number"`
			JumiaSKU    string `gorm:"column:jumia_sku"`
		}
		err := db.Model(&models.CustomizeOrderKE{}).
			Select("ke_id, order_number, jumia_sku").
			Where("ke_id > ?", lastID).
			Order("ke_id ASC").
			Limit(batchSize).
			Find(&currentBatch).Error

		if err != nil {
			return result, fmt.Errorf("分页查询 CustomizeOrderKE 失败: %w", err)
		}

		if len(currentBatch) == 0 {
			break
		}

		log.Printf("[FillAllCustomizeOrderKE] 处理批次: %d 条 (KEID %d - %d)",
			len(currentBatch), currentBatch[0].KEID, currentBatch[len(currentBatch)-1].KEID)

		// 提取并清洗数据
		var orderNumbers []string
		var jumiaSKUs []string
		orderIDMap := make(map[string]int)
		jumiaSKUToKEIDMap := make(map[string]int)
		batchEmptyOrderNumber := 0

		for _, item := range currentBatch {
			// 清洗 JumiaSKU 并收集
			cleanedJumiaSKU := strings.TrimSpace(item.JumiaSKU)
			if cleanedJumiaSKU != "" {
				jumiaSKUs = append(jumiaSKUs, cleanedJumiaSKU)
				jumiaSKUToKEIDMap[cleanedJumiaSKU] = item.KEID
			}

			// 清洗 OrderNumber 并统计
			cleanedOrderNumber := strings.TrimSpace(item.OrderNumber)
			if cleanedOrderNumber == "" {
				batchEmptyOrderNumber++
				result.EmptyOrderNumber++
				continue
			}
			orderNumbers = append(orderNumbers, cleanedOrderNumber)
			orderIDMap[cleanedOrderNumber] = item.KEID
		}

		log.Printf("[FillAllCustomizeOrderKE] 有效订单号: %d, 空订单号: %d, 非空 JumiaSKU: %d",
			len(orderNumbers), batchEmptyOrderNumber, len(jumiaSKUs))

		// 批次统计
		batchUpdatedStatus := 0
		batchUpdatedTrackingURL := 0
		batchUpdatedPerson := 0
		batchUpdatedTotal := 0
		batchNotFoundOrderItem := 0
		batchNotFoundUser := 0
		var notFoundOrderNumbers []string
		var notFoundJumiaSKUs []string

		// 2. 查询 OrderItem（状态 & 追踪链接）
		var orderItemResults []struct {
			OrderNumber string `gorm:"column:order_number"`
			Status      string `gorm:"column:status"`
			TrackingURL string `gorm:"column:tracking_url"`
		}
		if len(orderNumbers) > 0 {
			err = db.Table("order_items").
				Select("TRIM(order_number) AS order_number, status, tracking_url").
				Where("TRIM(order_number) IN ?", orderNumbers).
				Find(&orderItemResults).Error

			if err != nil {
				return result, fmt.Errorf("查询 order_items 失败: %w", err)
			}
			log.Printf("[FillAllCustomizeOrderKE] 匹配到 %d 条 OrderItem 记录", len(orderItemResults))
		}

		// 构建 OrderNumber → 数据映射（使用清洗后的 key）
		orderItemMap := make(map[string]struct {
			Status      string
			TrackingURL string
		})
		for _, item := range orderItemResults {
			cleaned := strings.TrimSpace(item.OrderNumber)
			orderItemMap[cleaned] = struct {
				Status      string
				TrackingURL string
			}{Status: item.Status, TrackingURL: item.TrackingURL}
		}

		// 3. 查询 UserProduct 获取 UserName（大小写 + 空格不敏感）
		var userResults []UserProductWithUser
		if len(jumiaSKUs) > 0 {
			// 构建小写清洗列表用于查询
			var lowerJumiaSKUs []string
			for _, sku := range jumiaSKUs {
				lowerJumiaSKUs = append(lowerJumiaSKUs, strings.ToLower(strings.TrimSpace(sku)))
			}

			err = db.Table("user_products AS up").
				Select(`
					TRIM(up.jumia_sku) AS jumia_sku,
					up.seller_sku,
					ussm.user_id,
					um.user_name
				`).
				Joins("JOIN user_seller_sku_models AS ussm ON TRIM(up.seller_sku) = TRIM(ussm.seller_sku)").
				Joins("JOIN user_models AS um ON um.id = ussm.user_id").
				Where("LOWER(TRIM(up.jumia_sku)) IN ?", lowerJumiaSKUs).
				Scan(&userResults).Error

			if err != nil {
				log.Printf("[FillAllCustomizeOrderKE] 查询 UserProduct 失败: %v", err)
			} else {
				log.Printf("[FillAllCustomizeOrderKE] 匹配到 %d 条用户记录", len(userResults))
			}
		}

		// 构建 userNameMap：使用小写 + 清洗后的 SKU 作为 key
		userNameMap := make(map[string]string)
		for _, ur := range userResults {
			key := strings.ToLower(strings.TrimSpace(ur.JumiaSku))
			if ur.PersonUserName != "" {
				userNameMap[key] = ur.PersonUserName
			}
		}

		// 4. 准备更新数据
		var updates []map[string]interface{}
		for _, item := range currentBatch {
			keID := item.KEID
			update := map[string]interface{}{"ke_id": keID}
			updated := false

			// 1. 更新 Person（使用清洗 + 小写 key 匹配）
			cleanedJumiaSKU := strings.TrimSpace(item.JumiaSKU)
			if cleanedJumiaSKU != "" {
				key := strings.ToLower(cleanedJumiaSKU)
				if userName, found := userNameMap[key]; found {
					update["person"] = userName
					batchUpdatedPerson++
					updated = true
				} else {
					batchNotFoundUser++
					notFoundJumiaSKUs = append(notFoundJumiaSKUs, item.JumiaSKU)
				}
			}

			// 2. 更新 Status 和 TrackingURL
			cleanedOrderNumber := strings.TrimSpace(item.OrderNumber)
			if cleanedOrderNumber != "" {
				if orderItemData, found := orderItemMap[cleanedOrderNumber]; found {
					if orderItemData.Status != "" {
						update["status"] = orderItemData.Status
						batchUpdatedStatus++
						updated = true
					}
					if orderItemData.TrackingURL != "" {
						update["tracking_url"] = orderItemData.TrackingURL
						batchUpdatedTrackingURL++
						updated = true
					}
				} else {
					batchNotFoundOrderItem++
					notFoundOrderNumbers = append(notFoundOrderNumbers, item.OrderNumber)
				}
			}

			// 3. 即使 OrderNumber 为空，只要 Person 更新了，也要更新
			if updated {
				updates = append(updates, update)
				batchUpdatedTotal++
			}
		}

		// 5. 批量更新
		if len(updates) > 0 {
			dbResult := db.Model(&models.CustomizeOrderKE{}).
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "ke_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"status", "tracking_url", "person"}),
				}).
				Create(&updates)

			if dbResult.Error != nil {
				log.Printf("[FillAllCustomizeOrderKE] 批量更新失败: %v", dbResult.Error)
			} else {
				log.Printf("[FillAllCustomizeOrderKE] 成功更新 %d 条记录", len(updates))
			}
		}

		// 更新全局统计
		result.TotalProcessed += len(currentBatch)
		result.UpdatedStatus += batchUpdatedStatus
		result.UpdatedTrackingURL += batchUpdatedTrackingURL
		result.UpdatedPerson += batchUpdatedPerson
		result.TotalUpdated += batchUpdatedTotal
		result.NotFoundInOrderItem += batchNotFoundOrderItem

		// 日志输出
		if len(notFoundOrderNumbers) > 0 {
			log.Printf("[FillAllCustomizeOrderKE] 未找到的订单号 (%d): %v", len(notFoundOrderNumbers), notFoundOrderNumbers)
		}
		if len(notFoundJumiaSKUs) > 0 {
			log.Printf("[FillAllCustomizeOrderKE] 未找到的 JumiaSKU (%d): %v", len(notFoundJumiaSKUs), notFoundJumiaSKUs)
		}

		log.Printf("[FillAllCustomizeOrderKE] 批次完成: 处理=%d, 更新=%d, 空号=%d, 未匹配订单=%d, 未匹配用户=%d",
			len(currentBatch), batchUpdatedTotal, batchEmptyOrderNumber, batchNotFoundOrderItem, batchNotFoundUser)

		lastID = currentBatch[len(currentBatch)-1].KEID
	}

	result.ElapsedTime = time.Since(startTime)

	log.Printf(`[FillAllCustomizeOrderKE] ✅ 填充完成！
		总计处理: %d
		订单号为空: %d
		未找到订单: %d
		更新 Status: %d
		更新 TrackingURL: %d
		更新 Person: %d
		总更新: %d
		耗时: %v`,
		result.TotalProcessed,
		result.EmptyOrderNumber,
		result.NotFoundInOrderItem,
		result.UpdatedStatus,
		result.UpdatedTrackingURL,
		result.UpdatedPerson,
		result.TotalUpdated,
		result.ElapsedTime)

	return result, nil
}
