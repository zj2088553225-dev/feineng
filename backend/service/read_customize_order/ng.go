// read_customize_order/csv_import.go
package read_customize_order

import (
	"backend/global"
	"backend/models"
	"encoding/csv"
	"fmt"
	"gorm.io/gorm/clause"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// ImportKECSVToMySQL 分批读取 CSV 并导入数据库
func ImportNGCSVToMySQL(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// 读取标题
	header, err := reader.Read()
	if err != nil {
		return 0, err
	}
	log.Printf("📊 CSV 标题行: %v", header)

	var totalProcessed int64
	var batch []models.CustomizeOrderNG
	var inserted, updated int64

	// 查询导入前数量
	var beforeCount int64
	global.DB.Model(&models.CustomizeOrderNG{}).Count(&beforeCount)
	log.Printf("📥 导入开始: 文件=%s, 当前数据库记录=%d", filePath, beforeCount)

	// 预加载已存在的 KEID
	existingNGIDs := make(map[int]bool)
	if beforeCount > 0 {
		var results []struct{ NGID int }
		global.DB.Model(&models.CustomizeOrderNG{}).Pluck("ng_id", &results)
		for _, r := range results {
			existingNGIDs[r.NGID] = true
		}
	}

	batchInserted := 0
	batchUpdated := 0

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("⚠️  读取行出错: %v", err)
			continue
		}

		totalProcessed++

		order, err := parseRecordSafeNG(record)
		if err != nil {
			log.Printf("⚠️  解析失败 (第 %d 行): %v", totalProcessed, err)
			continue
		}
		if order == nil {
			continue
		}

		if order.NGID == 0 {
			log.Printf("❌ 跳过无效 NGID (第 %d 行)", totalProcessed)
			continue
		}

		if existingNGIDs[order.NGID] {
			updated++
			batchUpdated++
		} else {
			inserted++
			batchInserted++
			existingNGIDs[order.NGID] = true
		}

		batch = append(batch, *order)

		if len(batch) >= BatchSize {
			err := safeUpsertNGWithVerify(batch)
			if err != nil {
				log.Printf("❌ 批次插入失败: %v", err)
				var keids []int
				for _, o := range batch {
					keids = append(keids, o.NGID)
				}
				log.Printf("📋 失败批次 NGID: %v", keids)
			} else {
				log.Printf("✅ 提交 %d 条 (新增=%d, 更新=%d)", len(batch), batchInserted, batchUpdated)
			}
			batch = batch[:0]
			batchInserted = 0
			batchUpdated = 0
		}
	}

	// 最后一批
	if len(batch) > 0 {
		err := safeUpsertNGWithVerify(batch)
		if err != nil {
			log.Printf("❌ 最后一批提交失败: %v", err)
			var keids []int
			for _, o := range batch {
				keids = append(keids, o.NGID)
			}
			log.Printf("📋 最后一批失败 KEID: %v", keids)
		} else {
			log.Printf("✅ 最后一批提交 %d 条", len(batch))
		}
	}

	// 统计结果
	var afterCount int64
	global.DB.Model(&models.CustomizeOrderNG{}).Count(&afterCount)
	log.Printf("🎉 导入完成: CSV总行=%d, 新增=%d, 更新=%d, 原=%d, 现=%d",
		totalProcessed, inserted, updated, beforeCount, afterCount)

	// 数据一致性检查
	if inserted+beforeCount != afterCount {
		log.Printf("🚨 警告: 数据不一致! 预期 %d 条，实际 %d 条，差额 %d",
			inserted+beforeCount, afterCount, inserted+beforeCount-afterCount)
	}

	return inserted + updated, nil
}

// parseRecordSafeKE 解析一行 CSV 数据
func parseRecordSafeNG(record []string) (*models.CustomizeOrderNG, error) {
	if len(record) < 29 {
		return nil, fmt.Errorf("字段不足，期望 >=29，实际 %d", len(record))
	}

	// 解析 Call Date: 日/月/年，如 1/5/2025
	parseCallDate := func(s string) *time.Time {
		s = clean(s)
		if s == "" || s == "#N/A" {
			return nil
		}
		// 优先尝试 日/月/年
		if t, err := time.Parse("2/1/2006", s); err == nil {
			return &t
		}
		log.Printf("📅 CallDate 解析失败: '%s'", s)
		return nil
	}

	// 解析 Order Date: 2025.04.16 13:06:57
	parseOrderDate := func(s string) *time.Time {
		s = clean(s)
		if s == "" || s == "#N/A" {
			return nil
		}
		// 注意：Go 时间布局是 2006.01.02 15:04:05
		if t, err := time.Parse("2006.01.02 15:04:05", s); err == nil {
			return &t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			return &t
		}
		if t, err := time.Parse("2006.01.02", s); err == nil {
			return &t
		}
		log.Printf("📅 OrderDate 解析失败: '%s'", s)
		return nil
	}

	// 解析 KEID（第30列，record[29]）
	ngidStr := clean(record[28])
	if ngidStr == "" || ngidStr == "#N/A" {
		return nil, nil
	}

	var ngid int
	// 处理科学计数法：2.54072E+12
	if strings.ContainsAny(ngidStr, "eE") {
		if f, err := strconv.ParseFloat(ngidStr, 64); err == nil {
			ngid = int(f)
		} else {
			log.Printf("❌ NGID 科学计数法解析失败: %s", ngidStr)
			return nil, nil
		}
	} else {
		if i, err := strconv.Atoi(ngidStr); err == nil {
			ngid = i
		} else {
			log.Printf("❌ NGID 非数字: %s", ngidStr)
			return nil, nil
		}
	}

	if ngid == 0 {
		return nil, nil
	}

	return &models.CustomizeOrderNG{
		NGID:           ngid,
		Week:           truncate(clean(record[0]), 20),
		Date:           parseCallDate(record[1]), // 日/月/年
		ID:             truncate(clean(record[2]), 100),
		Time:           parseOrderDate(record[3]), // 2025.04.16 13:06:57
		ItemName:       truncate(clean(record[4]), 100),
		Price:          truncate(clean(record[5]), 20),
		Qty:            truncate(clean(record[6]), 10),
		CustomerName:   truncate(clean(record[7]), 100),
		PhoneNumber:    truncate(clean(record[8]), 100),
		PhoneNumber2:   truncate(clean(record[9]), 100),
		Address:        truncate(clean(record[10]), 200),
		City:           truncate(clean(record[11]), 50),
		Region:         truncate(clean(record[12]), 50),
		Email:          truncate(clean(record[13]), 100),
		JumiaSKU:       truncate(clean(record[14]), 50),
		PusAddress:     truncate(clean(record[15]), 100),
		SellerAgent:    truncate(clean(record[16]), 50),
		Called:         truncate(clean(record[17]), 100),
		OrderStatus:    truncate(clean(record[18]), 50),
		Reached:        truncate(clean(record[19]), 20),
		ShippingMethod: truncate(clean(record[20]), 50),
		JumiaAgentName: truncate(clean(record[21]), 50),
		OrderPlaced:    truncate(clean(record[22]), 50),
		OrderNumber:    truncate(clean(record[23]), 50),
		SellerComment:  truncate(clean(record[24]), 200),
		Person:         truncate(clean(record[26]), 50),
		Status:         truncate(clean(record[27]), 50),
		TrackingURL:    truncate(clean(record[28]), 200),
	}, nil
}

func safeUpsertNGWithVerify(orders []models.CustomizeOrderNG) error {
	err := upsertBatchNGAndVerify(orders)
	if err != nil {
		log.Printf("⚠️ 批次失败，2秒后重试...")
		time.Sleep(2 * time.Second)
		err = upsertBatchNGAndVerify(orders)
		if err != nil {
			return fmt.Errorf("重试后仍失败: %w", err)
		}
	}
	return nil
}

func upsertBatchNGAndVerify(orders []models.CustomizeOrderNG) error {
	var ngids []int
	for _, o := range orders {
		ngids = append(ngids, o.NGID)
	}

	result := global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ng_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"week":             clause.Expr{SQL: "VALUES(week)"},
			"date":             clause.Expr{SQL: "VALUES(date)"},
			"id":               clause.Expr{SQL: "VALUES(id)"},
			"time":             clause.Expr{SQL: "VALUES(time)"},
			"item_name":        clause.Expr{SQL: "VALUES(item_name)"},
			"price":            clause.Expr{SQL: "VALUES(price)"},
			"qty":              clause.Expr{SQL: "VALUES(qty)"},
			"customer_name":    clause.Expr{SQL: "VALUES(customer_name)"},
			"phone_number":     clause.Expr{SQL: "VALUES(phone_number)"},
			"phone_number_2":   clause.Expr{SQL: "VALUES(phone_number_2)"},
			"address":          clause.Expr{SQL: "VALUES(address)"},
			"city":             clause.Expr{SQL: "VALUES(city)"},
			"region":           clause.Expr{SQL: "VALUES(region)"},
			"email":            clause.Expr{SQL: "VALUES(email)"},
			"jumia_sku":        clause.Expr{SQL: "VALUES(jumia_sku)"},
			"pus_address":      clause.Expr{SQL: "VALUES(pus_address)"},
			"seller_agent":     clause.Expr{SQL: "VALUES(seller_agent)"},
			"called":           clause.Expr{SQL: "VALUES(called)"},
			"order_status":     clause.Expr{SQL: "VALUES(order_status)"},
			"reached":          clause.Expr{SQL: "VALUES(reached)"},
			"shipping_method":  clause.Expr{SQL: "VALUES(shipping_method)"},
			"jumia_agent_name": clause.Expr{SQL: "VALUES(jumia_agent_name)"},
			"order_placed":     clause.Expr{SQL: "VALUES(order_placed)"},
			"order_number":     clause.Expr{SQL: "VALUES(order_number)"},
			"seller_comment":   clause.Expr{SQL: "VALUES(seller_comment)"},
			"person":           clause.Expr{SQL: "VALUES(person)"},
			"status":           clause.Expr{SQL: "VALUES(status)"},
			"tracking_url":     clause.Expr{SQL: "VALUES(tracking_url)"},
		}),
	}).Create(&orders)

	if result.Error != nil {
		return result.Error
	}

	// 验证写入
	var found int64
	global.DB.Model(&models.CustomizeOrderNG{}).Where("ng_id IN ?", ngids).Count(&found)
	if found != int64(len(ngids)) {
		return fmt.Errorf("部分数据未写入，期望 %d，实际 %d", len(ngids), found)
	}

	return nil
}
