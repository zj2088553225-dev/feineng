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
	"strings"
	"time"
	"unicode"
)

const BatchSize = 100

// ImportGHCSVToMySQL 分批读取 CSV 并导入数据库（修复丢数据问题）
func ImportGHCSVToMySQL(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// 预扫描唯一 GHID 数量
	uniqueGHIDCount, err := countUniqueGHIDs(filePath)
	if err != nil {
		log.Printf("⚠️  预扫描失败: %v", err)
	} else {
		log.Printf("🔍 CSV 中唯一 GHID 数量: %d", uniqueGHIDCount)
	}

	utf8Reader := &utf8Reader{reader: file}
	reader := csv.NewReader(utf8Reader)
	reader.Comma = ','
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// 读取标题
	header, err := reader.Read()
	if err != nil {
		return 0, err
	}

	var totalProcessed int64
	var batch []models.CustomizeOrderGH
	var inserted, updated int64

	// 统计导入前已有记录
	var beforeCount int64
	global.DB.Model(&models.CustomizeOrderGH{}).Count(&beforeCount)
	log.Printf("📊 导入开始: 文件=%s, 列数=%d, 当前记录=%d", filePath, len(header), beforeCount)

	// 预加载已存在 GHID
	existingGHIDs := make(map[int]bool)
	if beforeCount > 0 {
		var results []struct{ GHID int }
		global.DB.Model(&models.CustomizeOrderGH{}).Pluck("gh_id", &results)
		for _, r := range results {
			existingGHIDs[r.GHID] = true
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

		order, err := parseRecordSafe(record)
		if err != nil {
			log.Printf("⚠️  解析失败 (第 %d 行): %v", totalProcessed, err)
			continue
		}
		if order == nil {
			continue
		}

		// 跳过无效 GHID
		if order.GHID == 0 {
			log.Printf("❌ 跳过无效 GHID (第 %d 行)", totalProcessed)
			continue
		}

		// 判断新增/更新
		if existingGHIDs[order.GHID] {
			updated++
			batchUpdated++
		} else {
			inserted++
			batchInserted++
			existingGHIDs[order.GHID] = true // 标记为已存在（本批次内）
		}

		batch = append(batch, *order)

		if len(batch) >= BatchSize {
			err := safeUpsertWithVerify(batch)
			if err != nil {
				log.Printf("❌ 批次插入失败: %v", err)
				// 打印失败批次的 GHID
				var ghids []int
				for _, o := range batch {
					ghids = append(ghids, o.GHID)
				}
				log.Printf("📋 失败批次 GHID: %v", ghids)
			} else {
				log.Printf("✅ 提交 %d 条 (新增=%d, 更新=%d), 累计影响=%d", len(batch), batchInserted, batchUpdated, inserted+updated)
			}
			batch = batch[:0]
			batchInserted = 0
			batchUpdated = 0
		}
	}

	// 最后一批
	if len(batch) > 0 {
		err := safeUpsertWithVerify(batch)
		if err != nil {
			log.Printf("❌ 最后一批提交失败: %v", err)
			var ghids []int
			for _, o := range batch {
				ghids = append(ghids, o.GHID)
			}
			log.Printf("📋 最后一批失败 GHID: %v", ghids)
		} else {
			log.Printf("✅ 最后一批提交 %d 条 (新增=%d, 更新=%d)", len(batch), batchInserted, batchUpdated)
		}
	}

	// 导入后统计
	var afterCount int64
	global.DB.Model(&models.CustomizeOrderGH{}).Count(&afterCount)
	log.Printf("🎉 导入完成: CSV总行=%d, 新增=%d, 更新=%d, 原=%d, 现=%d",
		totalProcessed, inserted, updated, beforeCount, afterCount)

	// 🔍 额外验证：检查是否有数据丢失
	if inserted+beforeCount != afterCount {
		log.Printf("🚨 警告: 数据不一致! 预期 %d 条，实际 %d 条，差额 %d",
			inserted+beforeCount, afterCount, inserted+beforeCount-afterCount)
	}

	return inserted + updated, nil
}

// safeUpsertWithVerify 批量插入 + 失败重试 + 插入后验证
func safeUpsertWithVerify(orders []models.CustomizeOrderGH) error {
	err := upsertBatchAndVerify(orders)
	if err != nil {
		log.Printf("⚠️ 批次失败，2秒后重试...")
		time.Sleep(2 * time.Second)
		err = upsertBatchAndVerify(orders)
		if err != nil {
			return fmt.Errorf("重试后仍失败: %w", err)
		}
	}
	return nil
}

// upsertBatchAndVerify 执行 upsert 并验证结果
func upsertBatchAndVerify(orders []models.CustomizeOrderGH) error {
	// 提取 GHID 用于验证
	var ghids []int
	for _, o := range orders {
		ghids = append(ghids, o.GHID)
	}

	// 执行 upsert
	result := global.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "gh_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"week":            clause.Expr{SQL: "VALUES(week)"},
			"date":            clause.Expr{SQL: "VALUES(date)"},
			"order_numb":      clause.Expr{SQL: "VALUES(order_numb)"},
			"qty":             clause.Expr{SQL: "VALUES(qty)"},
			"amount":          clause.Expr{SQL: "VALUES(amount)"},
			"order_shop":      clause.Expr{SQL: "VALUES(order_shop)"},
			"product_name":    clause.Expr{SQL: "VALUES(product_name)"},
			"first_name":      clause.Expr{SQL: "VALUES(first_name)"},
			"last_name":       clause.Expr{SQL: "VALUES(last_name)"},
			"phone_number":    clause.Expr{SQL: "VALUES(phone_number)"},
			"email_addr":      clause.Expr{SQL: "VALUES(email_addr)"},
			"address":         clause.Expr{SQL: "VALUES(address)"},
			"city":            clause.Expr{SQL: "VALUES(city)"},
			"jumia_sku":       clause.Expr{SQL: "VALUES(jumia_sku)"},
			"agents":          clause.Expr{SQL: "VALUES(agents)"},
			"called":          clause.Expr{SQL: "VALUES(called)"},
			"order_done":      clause.Expr{SQL: "VALUES(order_done)"},
			"call_comment":    clause.Expr{SQL: "VALUES(call_comment)"},
			"closest_pus":     clause.Expr{SQL: "VALUES(closest_pus)"},
			"order_number":    clause.Expr{SQL: "VALUES(order_number)"},
			"agent_comments":  clause.Expr{SQL: "VALUES(agent_comments)"},
			"seller_comments": clause.Expr{SQL: "VALUES(seller_comments)"},
			"wa_contact_made": clause.Expr{SQL: "VALUES(wa_contact_made)"},
			"person":          clause.Expr{SQL: "VALUES(person)"},
			"status":          clause.Expr{SQL: "VALUES(status)"},
			"tracking_url":    clause.Expr{SQL: "VALUES(tracking_url)"},
		}),
	}).Create(&orders)

	if result.Error != nil {
		return result.Error
	}

	// ✅ 验证：确保所有 GHID 都已存在
	var foundCount int64
	global.DB.Model(&models.CustomizeOrderGH{}).
		Where("gh_id IN ?", ghids).
		Count(&foundCount)

	if foundCount != int64(len(ghids)) {
		// 找出缺失的 GHID
		var existing []int
		global.DB.Model(&models.CustomizeOrderGH{}).
			Where("gh_id IN ?", ghids).
			Pluck("gh_id", &existing)

		existMap := make(map[int]bool)
		for _, id := range existing {
			existMap[id] = true
		}

		var missing []int
		for _, id := range ghids {
			if !existMap[id] {
				missing = append(missing, id)
			}
		}

		return fmt.Errorf("批次中 %d 条未写入: %v", len(missing), missing)
	}

	return nil
}

// 统计唯一 GHID
func countUniqueGHIDs(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	utf8Reader := &utf8Reader{reader: file}
	reader := csv.NewReader(utf8Reader)
	reader.Comma = ','
	reader.LazyQuotes = true

	if _, err := reader.Read(); err != nil {
		return 0, err
	}

	ghidSet := make(map[int]bool)
	var count, duplicates, invalid int

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(record) < 27 {
			continue
		}

		rawGHID := clean(record[26])
		if rawGHID == "" {
			invalid++
			continue
		}

		var ghid int
		_, err = fmt.Sscanf(rawGHID, "%d", &ghid)
		if err != nil {
			invalid++
			continue
		}

		if ghidSet[ghid] {
			duplicates++
		} else {
			ghidSet[ghid] = true
			count++
		}
	}

	if duplicates > 0 {
		log.Printf("🔍 发现 %d 个重复 GHID", duplicates)
	}
	if invalid > 0 {
		log.Printf("🔍 发现 %d 个无效 GHID", invalid)
	}

	return count, nil
}

// UTF-8 修复 Reader
type utf8Reader struct {
	reader io.Reader
}

func (r *utf8Reader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func clean(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u3000", "")
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' {
			return -1
		}
		return r
	}, s)
	s = strings.ToValidUTF8(s, "")
	if s == "#N/A" || s == "N/A" || s == "" {
		return ""
	}
	return s
}

func parseRecordSafe(record []string) (*models.CustomizeOrderGH, error) {
	const (
		MaxWK          = 10
		MaxOrderNumb   = 50
		MaxOrderShop   = 255
		MaxProductName = 500
		MaxName        = 100
		MaxPhone       = 100
		MaxEmail       = 255
		MaxAddress     = 500
		MaxCity        = 100
		MaxJumiaSKU    = 50
		MaxAgents      = 100
		MaxClosestPUS  = 100
		MaxOrderNumber = 50
		MaxPerson      = 100
		MaxStatus      = 50
		MaxTrackingURL = 500
		MaxComments    = 500
	)

	parseDate := func(value string) *time.Time {
		value = clean(value)
		if value == "" {
			return nil
		}
		// 调整顺序：先尝试 日/月/年，再尝试 月/日/年
		formats := []string{
			"2/1/2006", // 日/月/年
			"1/2/2006", // 月/日/年
			"2006-01-02",
			"2006-1-2",
			"1/2/06", // 注意：这个也需要对应调整，如果它也表示 日/月/年
			// "2/1/06",  // 如果 1/2/06 表示 月/日/年，则需要这个
		}
		for _, format := range formats {
			if t, err := time.Parse(format, value); err == nil {
				return &t
			}
		}
		log.Printf("📅 日期解析失败: '%s'", value)
		return nil
	}

	if len(record) < 27 {
		return nil, fmt.Errorf("字段数不足: %d", len(record))
	}

	var ghid int
	ghidStr := clean(record[26])
	if ghidStr != "" {
		_, err := fmt.Sscanf(ghidStr, "%d", &ghid)
		if err != nil {
			ghid = 0
		}
	}

	return &models.CustomizeOrderGH{
		Week:           truncate(record[0], MaxWK),
		Date:           parseDate(record[1]),
		OrderNumb:      truncate(record[2], MaxOrderNumb),
		Qty:            truncate(record[3], 10),
		Amount:         truncate(record[4], 20),
		OrderShop:      truncate(record[5], MaxOrderShop),
		ProductName:    truncate(record[6], MaxProductName),
		FirstName:      truncate(record[7], MaxName),
		LastName:       truncate(record[8], MaxName),
		PhoneNumber:    truncate(record[9], MaxPhone),
		EmailAddr:      truncate(record[10], MaxEmail),
		Address:        truncate(record[11], MaxAddress),
		City:           truncate(record[12], MaxCity),
		JumiaSKU:       truncate(record[13], MaxJumiaSKU),
		Agents:         truncate(record[14], MaxAgents),
		CALLED:         truncate(record[15], 100),
		OrderDone:      truncate(record[16], 100),
		CallComment:    truncate(record[17], MaxComments),
		ClosestPUS:     truncate(record[18], MaxClosestPUS),
		OrderNumber:    truncate(record[19], MaxOrderNumber),
		AgentComments:  truncate(record[20], MaxComments),
		SellerComments: truncate(record[21], MaxComments),
		WAContactMade:  truncate(record[22], 100),
		Person:         truncate(record[23], MaxPerson),
		Status:         truncate(record[24], MaxStatus),
		TrackingURL:    truncate(record[25], MaxTrackingURL),
		GHID:           ghid,
	}, nil
}

func truncate(s string, max int) string {
	s = clean(s)
	if len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}
