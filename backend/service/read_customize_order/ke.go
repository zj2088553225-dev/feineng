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
func ImportKECSVToMySQL(filePath string) (int64, error) {
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
	var batch []models.CustomizeOrderKE
	var inserted, updated int64

	// 查询导入前数量
	var beforeCount int64
	global.DB.Model(&models.CustomizeOrderKE{}).Count(&beforeCount)
	log.Printf("📥 导入开始: 文件=%s, 当前数据库记录=%d", filePath, beforeCount)

	// 预加载已存在的 KEID
	existingKEIDs := make(map[int]bool)
	if beforeCount > 0 {
		var results []struct{ KEID int }
		global.DB.Model(&models.CustomizeOrderKE{}).Pluck("ke_id", &results)
		for _, r := range results {
			existingKEIDs[r.KEID] = true
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

		order, err := parseRecordSafeKE(record)
		if err != nil {
			log.Printf("⚠️  解析失败 (第 %d 行): %v", totalProcessed, err)
			continue
		}
		if order == nil {
			continue
		}

		if order.KEID == 0 {
			log.Printf("❌ 跳过无效 KEID (第 %d 行)", totalProcessed)
			continue
		}

		if existingKEIDs[order.KEID] {
			updated++
			batchUpdated++
		} else {
			inserted++
			batchInserted++
			existingKEIDs[order.KEID] = true
		}

		batch = append(batch, *order)

		if len(batch) >= BatchSize {
			err := safeUpsertKEWithVerify(batch)
			if err != nil {
				log.Printf("❌ 批次插入失败: %v", err)
				var keids []int
				for _, o := range batch {
					keids = append(keids, o.KEID)
				}
				log.Printf("📋 失败批次 KEID: %v", keids)
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
		err := safeUpsertKEWithVerify(batch)
		if err != nil {
			log.Printf("❌ 最后一批提交失败: %v", err)
			var keids []int
			for _, o := range batch {
				keids = append(keids, o.KEID)
			}
			log.Printf("📋 最后一批失败 KEID: %v", keids)
		} else {
			log.Printf("✅ 最后一批提交 %d 条", len(batch))
		}
	}

	// 统计结果
	var afterCount int64
	global.DB.Model(&models.CustomizeOrderKE{}).Count(&afterCount)
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
func parseRecordSafeKE(record []string) (*models.CustomizeOrderKE, error) {
	if len(record) < 30 {
		return nil, fmt.Errorf("字段不足，期望 >=30，实际 %d", len(record))
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
	keidStr := clean(record[29])
	if keidStr == "" || keidStr == "#N/A" {
		return nil, nil
	}

	var keid int
	// 处理科学计数法：2.54072E+12
	if strings.ContainsAny(keidStr, "eE") {
		if f, err := strconv.ParseFloat(keidStr, 64); err == nil {
			keid = int(f)
		} else {
			log.Printf("❌ KEID 科学计数法解析失败: %s", keidStr)
			return nil, nil
		}
	} else {
		if i, err := strconv.Atoi(keidStr); err == nil {
			keid = i
		} else {
			log.Printf("❌ KEID 非数字: %s", keidStr)
			return nil, nil
		}
	}

	if keid == 0 {
		return nil, nil
	}

	return &models.CustomizeOrderKE{
		KEID:                keid,
		First:               truncate(clean(record[0]), 20),
		CallDate:            parseCallDate(record[1]), // 日/月/年
		ID:                  truncate(clean(record[2]), 100),
		OrderDate:           parseOrderDate(record[3]), // 2025.04.16 13:06:57
		ItemName:            truncate(clean(record[4]), 100),
		Price:               truncate(clean(record[5]), 100),
		Qty:                 truncate(clean(record[6]), 100),
		CustomerName:        truncate(clean(record[7]), 100),
		PhoneNumber:         cleanPhoneNumber(clean(record[8])), // 科学计数法
		PhoneNumber2:        cleanPhoneNumber(clean(record[9])), // 科学计数法
		Address:             truncate(clean(record[10]), 200),
		City:                truncate(clean(record[11]), 50),
		Region:              truncate(clean(record[12]), 50),
		Email:               truncate(clean(record[13]), 100),
		JumiaSKU:            truncate(clean(record[14]), 50),
		PickUpStations:      truncate(clean(record[15]), 100),
		SellerAgent:         truncate(clean(record[16]), 50),
		Called:              truncate(clean(record[17]), 100),
		OrderStatus:         truncate(clean(record[18]), 50),
		Reached:             truncate(clean(record[19]), 100),
		ShippingMethod:      truncate(clean(record[20]), 50),
		JumiaSalesAgentName: truncate(clean(record[21]), 50),
		OrderPlaced:         truncate(clean(record[22]), 50),
		OrderNumber:         truncate(clean(record[23]), 50),
		SellerComment:       truncate(clean(record[24]), 200),
		Ordered:             truncate(clean(record[25]), 200),
		Person:              truncate(clean(record[26]), 50),
		Status:              truncate(clean(record[27]), 50),
		TrackingURL:         truncate(clean(record[28]), 200),
	}, nil
}

// cleanPhoneNumber 处理科学计数法手机号
func cleanPhoneNumber(s string) string {
	s = clean(s)
	if s == "" || s == "#N/A" || s == "NULL" {
		return ""
	}
	if strings.ContainsAny(s, "eE") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return strconv.FormatInt(int64(f), 10)
		}
	}
	return s
}
func safeUpsertKEWithVerify(orders []models.CustomizeOrderKE) error {
	err := upsertBatchKEAndVerify(orders)
	if err != nil {
		log.Printf("⚠️ 批次失败，2秒后重试...")
		time.Sleep(2 * time.Second)
		err = upsertBatchKEAndVerify(orders)
		if err != nil {
			return fmt.Errorf("重试后仍失败: %w", err)
		}
	}
	return nil
}

func upsertBatchKEAndVerify(orders []models.CustomizeOrderKE) error {
	var keids []int
	for _, o := range orders {
		keids = append(keids, o.KEID)
	}

	result := global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "ke_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"first":                  clause.Expr{SQL: "VALUES(first)"},
			"call_date":              clause.Expr{SQL: "VALUES(call_date)"},
			"id":                     clause.Expr{SQL: "VALUES(id)"},
			"order_date":             clause.Expr{SQL: "VALUES(order_date)"},
			"item_name":              clause.Expr{SQL: "VALUES(item_name)"},
			"price":                  clause.Expr{SQL: "VALUES(price)"},
			"qty":                    clause.Expr{SQL: "VALUES(qty)"},
			"customer_name":          clause.Expr{SQL: "VALUES(customer_name)"},
			"phone_number":           clause.Expr{SQL: "VALUES(phone_number)"},
			"phone_number_2":         clause.Expr{SQL: "VALUES(phone_number_2)"},
			"address":                clause.Expr{SQL: "VALUES(address)"},
			"city":                   clause.Expr{SQL: "VALUES(city)"},
			"region":                 clause.Expr{SQL: "VALUES(region)"},
			"email":                  clause.Expr{SQL: "VALUES(email)"},
			"jumia_sku":              clause.Expr{SQL: "VALUES(jumia_sku)"},
			"pick_up_stations":       clause.Expr{SQL: "VALUES(pick_up_stations)"},
			"seller_agent":           clause.Expr{SQL: "VALUES(seller_agent)"},
			"called":                 clause.Expr{SQL: "VALUES(called)"},
			"order_status":           clause.Expr{SQL: "VALUES(order_status)"},
			"reached":                clause.Expr{SQL: "VALUES(reached)"},
			"shipping_method":        clause.Expr{SQL: "VALUES(shipping_method)"},
			"jumia_sales_agent_name": clause.Expr{SQL: "VALUES(jumia_sales_agent_name)"},
			"order_placed":           clause.Expr{SQL: "VALUES(order_placed)"},
			"order_number":           clause.Expr{SQL: "VALUES(order_number)"},
			"seller_comment":         clause.Expr{SQL: "VALUES(seller_comment)"},
			"ordered":                clause.Expr{SQL: "VALUES(ordered)"},
			"person":                 clause.Expr{SQL: "VALUES(person)"},
			"status":                 clause.Expr{SQL: "VALUES(status)"},
			"tracking_url":           clause.Expr{SQL: "VALUES(tracking_url)"},
		}),
	}).Create(&orders)

	if result.Error != nil {
		return result.Error
	}

	// 验证写入
	var found int64
	global.DB.Model(&models.CustomizeOrderKE{}).Where("ke_id IN ?", keids).Count(&found)
	if found != int64(len(keids)) {
		return fmt.Errorf("部分数据未写入，期望 %d，实际 %d", len(keids), found)
	}

	return nil
}
