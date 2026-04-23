package read_transaction_csv

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync_data/global"
	"sync_data/models"
	"time"

	"gorm.io/gorm/clause"
)

func nullString(s string) sql.NullString {
	if len(strings.TrimSpace(s)) == 0 {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: strings.TrimSpace(s), Valid: true}
}

const (
	DateTimeLayout = "2006-01-02 15:04:05"
	DateLayout     = "2006-01-02"
)

func parseTime2(layout, s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0000-00-00" || s == "0000-00-00 00:00:00" {
		return nil, nil // 返回 nil 表示 NULL
	}
	t, err := time.Parse(layout, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
func parseTime(layout, s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	// 指定东八区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to load location: %w", err)
	}

	// 用 ParseInLocation 确保按本地时区解析
	return time.ParseInLocation(layout, s, loc)
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0.0, fmt.Errorf("empty float string")
	}
	return strconv.ParseFloat(s, 64)
}

func ReadCSVTransactionToMysql(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("无法打开文件 %s: %v", filePath, err)
		return err
	}
	defer file.Close()

	global.Log.Info("开始读取 CSV 文件...")

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("读取 CSV 失败: %v", err)
		return err
	}

	if len(records) == 0 {
		global.Log.Warn("CSV 文件为空")
		return fmt.Errorf("CSV 文件为空")
	}

	// 跳过标题行
	records = records[1:]

	totalCount := len(records)
	global.Log.Infof("共发现 %d 条交易记录，开始处理...", totalCount)

	var (
		successCount = 0
		skipCount    = 0
	)

	tx := global.DB.Begin()
	if tx.Error != nil {
		global.Log.Errorf("开启事务失败: %v", tx.Error)
		return fmt.Errorf("开启事务失败")
	}

	var transactions []models.Transaction
	batchSize := 100

	// 所有要更新的字段（排除 created_at, updated_at 自动管理）
	updateFields := []string{
		"transaction_date",
		"transaction_type",
		"transaction_state",
		"details",
		"seller_sku",
		"jumia_sku",
		"amount",
		"statement_start_date",
		"statement_end_date",
		"paid_status",
		"order_no",
		"order_item_no",
		"order_item_status",
		"shipping_provider",
		"tracking_number",
		"comment",
		"local_exchange_rate",
		"country_code",
		"statement_number",
		// 注意：不更新 transaction_date 和 transaction_number
	}

	for i, record := range records {
		if i%100 == 0 {
			global.Log.Infof("正在处理第 %d/%d 条记录, TransactionNumber: %s", i+1, totalCount, record[2])
		}

		if len(record) != 20 {
			global.Log.Warnf("第 %d 行跳过: 列数异常，期望 20，实际 %d -> %v", i+1, len(record), record)
			skipCount++
			continue
		}

		transactionDate, err := parseTime(DateTimeLayout, record[0])

		if err != nil {
			global.Log.Warnf("第 %d 行跳过: TransactionDate 解析失败 [%s] -> %v", i+1, record[0], err)
			skipCount++
			continue
		}

		statementStartDate, err := parseTime2(DateTimeLayout, record[8])
		if err != nil {
			global.Log.Warnf("第 %d 行跳过: StatementStartDate 解析失败 [%s] -> %v", i+1, record[8], err)
			skipCount++
			//continue
		}

		statementEndDate, err := parseTime2(DateTimeLayout, record[9])
		if err != nil {
			global.Log.Warnf("第 %d 行跳过: StatementEndDate 解析失败 [%s] -> %v", i+1, record[9], err)
			skipCount++
			//continue
		}

		amount, err := parseFloat(record[7])
		if err != nil {
			global.Log.Warnf("第 %d 行跳过: Amount 解析失败 [%s] -> %v", i+1, record[7], err)
			skipCount++
			continue
		}

		localExchangeRate, err := parseFloat(record[17])
		if err != nil {
			global.Log.Warnf("第 %d 行跳过: LocalExchangeRate 解析失败 [%s] -> %v", i+1, record[17], err)
			skipCount++
			continue
		}

		paidStatus := strings.ToLower(strings.TrimSpace(record[10])) == "paid"

		transaction := models.Transaction{
			TransactionDate:    transactionDate,
			TransactionType:    strings.TrimSpace(record[1]),
			TransactionNumber:  strings.TrimSpace(record[2]),
			TransactionState:   strings.TrimSpace(record[3]),
			Details:            strings.TrimSpace(record[4]),
			SellerSKU:          strings.TrimSpace(record[5]),
			JumiaSKU:           strings.TrimSpace(record[6]),
			Amount:             amount,
			StatementStartDate: statementStartDate,
			StatementEndDate:   statementEndDate,
			PaidStatus:         paidStatus,
			OrderNo:            models.StringToNullString(record[11]),
			OrderItemNo:        models.StringToNullString(record[12]),
			OrderItemStatus:    models.StringToNullString(record[13]),
			ShippingProvider:   models.StringToNullString(record[14]),
			TrackingNumber:     models.StringToNullString(record[15]),
			Comment:            models.StringToNullString(record[16]),
			LocalExchangeRate:  localExchangeRate,
			CountryCode:        strings.TrimSpace(record[18]),
			StatementNumber:    strings.TrimSpace(record[19]),
		}

		transactions = append(transactions, transaction)
		successCount++

		// 批量 UPSERT
		if len(transactions) >= batchSize {
			result := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "transaction_number"}}, // 冲突字段
				DoUpdates: clause.Assignments(buildUpdateMap(updateFields)),
			}).Create(&transactions)

			if result.Error != nil {
				global.Log.Errorf("批量 UPSERT 失败: %v，事务将回滚", result.Error)
				tx.Rollback()
				return result.Error
			}
			global.Log.Infof("✅ 批量 UPSERT %d 条（已处理至第 %d 条）", len(transactions), i+1)
			transactions = []models.Transaction{}
		}
	}

	// 处理最后一批
	if len(transactions) > 0 {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "transaction_number"}},
			DoUpdates: clause.Assignments(buildUpdateMap(updateFields)),
		}).Create(&transactions)

		if result.Error != nil {
			global.Log.Errorf("最后一批 UPSERT 失败: %v", result.Error)
			tx.Rollback()
			return result.Error
		}
		global.Log.Infof("✅ UPSERT 最后 %d 条记录", len(transactions))
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		global.Log.Errorf("事务提交失败: %v", err)
		return err
	}

	// 统计输出
	global.Log.Infof("📊 数据导入完成！")
	global.Log.Infof("✅ 成功 UPSERT: %d 条", successCount)
	global.Log.Infof("⚠️  跳过（格式错误）: %d 条", skipCount)
	global.Log.Infof("📌 总记录数: %d", totalCount)

	return nil
}

// buildUpdateMap 构建字段 -> VALUES(xxx) 映射
func buildUpdateMap(fields []string) map[string]interface{} {
	updates := make(map[string]interface{})
	for _, f := range fields {
		updates[f] = clause.Expr{SQL: "VALUES(" + f + ")"} // 使用 clause.Expr 更安全
	}
	updates["updated_at"] = time.Now() // 手动更新 updated_at
	return updates
}
