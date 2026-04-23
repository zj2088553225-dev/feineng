package settlement

import (
	"backend/global"
	"backend/models"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"sync"
	"time"
)

type TaskStatus struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"` // "running", "done", "failed"
}

var (
	taskStatusMap = make(map[string]*TaskStatus)
	taskMutex     sync.RWMutex
	idGenerator   int64
)

func NewTaskID() string {
	taskMutex.Lock()
	idGenerator++
	taskMutex.Unlock()
	return fmt.Sprintf("task_%d", idGenerator)
}

func SetTaskStatus(taskID string, status *TaskStatus) {
	taskMutex.Lock()
	defer taskMutex.Unlock()
	taskStatusMap[taskID] = status
}

func GetTaskStatus(taskID string) (*TaskStatus, bool) {
	taskMutex.RLock()
	defer taskMutex.RUnlock()
	status, exists := taskStatusMap[taskID]
	return status, exists
}

// :todo这里需要存在根据一个默认配置计算所有用户对应国家的周期的结算数据
// CalculationSettlementData 计算每个用户的结算数据
func CalculationSettlementData(startDate, endDate time.Time) error {
	global.Log.Infof("结算任务执行时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	global.Log.Infof("开始处理结算周期: %s 到 %s",
		startDate.Format("2006-01-02 15:04:05"),
		endDate.Format("2006-01-02 15:04:05"))

	var users []models.UserModel
	if err := global.DB.Find(&users).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	for _, user := range users {
		if err := ProcessUserSettlement(user.ID, startDate, endDate); err != nil {
			global.Log.Errorf("用户 %d 结算处理失败: %v", user.ID, err)
			continue
		}
	}

	return nil
}

// :todo这里需要存在根据一个用户配置计算单个用户对应国家的周期的结算数据
// processUserSettlement 处理单个用户的结算（支持同一 SKU 多国销售）
func ProcessUserSettlement(userid uint, startDate, endDate time.Time) error {
	tx := global.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			global.Log.Errorf("用户 %d 处理 panic: %v", userid, r)
		}
	}()
	if tx.Error != nil {
		return tx.Error
	}

	// 1. 获取用户绑定的所有 SellerSKU
	var userSellerSkus []models.UserSellerSkuModel
	if err := tx.Where("user_id = ?", userid).Find(&userSellerSkus).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询用户 %d 的 seller_sku 失败: %w", userid, err)
	}
	if len(userSellerSkus) == 0 {
		global.Log.Infof("用户 %d 未绑定任何 SellerSKU，跳过", userid)
		tx.Rollback()
		return nil
	}
	var sellerSkus []string
	for _, item := range userSellerSkus {
		sellerSkus = append(sellerSkus, item.SellerSku)
	}

	// 2. 查询结算周期内的所有交易
	var transactions []models.Transaction
	if err := tx.Where("seller_sku IN ? AND transaction_date BETWEEN ? AND ? AND paid_status = ?", sellerSkus, startDate, endDate, 1).
		Find(&transactions).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("查询用户 %d 的交易记录失败: %w", userid, err)
	}
	if len(transactions) == 0 {
		global.Log.Infof("用户 %d 在周期内无交易记录，跳过", userid)
		tx.Rollback()
		return nil
	}

	// 3. 使用 (SellerSKU, CountryCode) 分组
	type SkuCountryKey struct {
		SellerSKU   string
		CountryCode string
	}
	skuCountryMap := make(map[SkuCountryKey]*models.UserSettlementDetail)

	for _, t := range transactions {
		key := SkuCountryKey{SellerSKU: t.SellerSKU, CountryCode: t.CountryCode}
		if _, exists := skuCountryMap[key]; !exists {
			skuCountryMap[key] = &models.UserSettlementDetail{
				UserID:              userid,
				SellerSKU:           t.SellerSKU,
				CountryCode:         t.CountryCode,
				SettlementStartDate: startDate,
				SettlementEndDate:   endDate,
			}
		}
		detail := skuCountryMap[key]

		switch t.TransactionType {
		case "Item Price", "Item Price Credit":
			detail.TotalSignedAmount += t.Amount
			if t.TransactionType == "Item Price Credit" {
				detail.SignedCount++
			}
		case "Commission", "Commission Credit":
			detail.JumiaCommission += t.Amount
		case "Outbound Fee", "Outbound Fee Credit":
			detail.OutboundFee += t.Amount
		case "Storage Fee":
			detail.StorageFee += t.Amount
		default:
			global.Log.Warnf("无需计算的交易类型: %s, 金额: %.2f", t.TransactionType, t.Amount)
		}
	}

	// 4. 按国家维度再汇总
	countryDetails := make(map[string][]*models.UserSettlementDetail)
	for _, detail := range skuCountryMap {
		// 计算 detail
		detail.CommissionRate = safeDiv(detail.JumiaCommission, detail.TotalSignedAmount)
		detail.ReceivedAmount = detail.TotalSignedAmount + detail.JumiaCommission + detail.OutboundFee + detail.StorageFee

		cloudRideRate, settlementRate := getCommissionRates(tx, userid, detail.CountryCode, detail.SellerSKU, startDate, endDate)
		detail.SettlementRate = settlementRate
		detail.CloudRideCommissionRate = cloudRideRate
		detail.CloudRideCommission = cloudRideRate * detail.TotalSignedAmount
		detail.ReviewFee = 10 * detail.SignedCount / settlementRate
		detail.PyvioFeeRate = 0.008
		detail.PyvioFee = 0
		if detail.ReceivedAmount > 0 {

			detail.PyvioFee = detail.ReceivedAmount * detail.PyvioFeeRate
		}
		detail.ActualSettleAmount = detail.ReceivedAmount - detail.CloudRideCommission - detail.ReviewFee - detail.PyvioFee
		detail.ActualSettleCNY = detail.ActualSettleAmount * settlementRate

		// 归入国家分组
		countryDetails[detail.CountryCode] = append(countryDetails[detail.CountryCode], detail)
	}

	// 5. 为每个国家写 summary + details
	for country, details := range countryDetails {
		var (
			totalSignedAmount, totalJumiaCommission, totalOutboundFee, totalStorageFee float64
			totalCloudRideCommission, totalPyvioFee, totalReviewFee                    float64
			totalReceivedAmount, totalSettleLocal, totalSettleCNY                      float64
			totalSignedCount                                                           float64
		)

		for _, d := range details {
			totalSignedAmount += d.TotalSignedAmount
			totalSignedCount += d.SignedCount
			totalJumiaCommission += d.JumiaCommission
			totalOutboundFee += d.OutboundFee
			totalStorageFee += d.StorageFee
			totalReceivedAmount += d.ReceivedAmount
			totalCloudRideCommission += d.CloudRideCommission
			totalPyvioFee += d.PyvioFee
			totalReviewFee += d.ReviewFee
			totalSettleLocal += d.ActualSettleAmount
			totalSettleCNY += d.ActualSettleCNY
		}

		// 查 summary
		var summary models.UserSettlementSummary
		err := tx.Where("user_id=? AND country_code=? AND settlement_start_date=? AND settlement_end_date=?",
			userid, country, startDate, endDate).First(&summary).Error

		if err == nil {
			// 更新
			summary.TotalSignedAmount = totalSignedAmount
			summary.SignedCount = totalSignedCount
			summary.TotalJumiaCommission = totalJumiaCommission
			summary.TotalOutboundFee = totalOutboundFee
			summary.TotalStorageFee = totalStorageFee
			summary.TotalCloudRideCommission = totalCloudRideCommission
			summary.TotalPyvioFee = totalPyvioFee
			summary.TotalReviewFee = totalReviewFee
			summary.ReceivedAmount = totalReceivedAmount
			summary.ActualSettleAmount = totalSettleLocal
			summary.ActualSettleCNY = totalSettleCNY
			summary.SettlementStatus = "待结算"

			if err := tx.Save(&summary).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("更新 UserSettlementSummary 失败: %w", err)
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新建
			summary = models.UserSettlementSummary{
				UserID:                   userid,
				CountryCode:              country,
				SettlementStartDate:      startDate,
				SettlementEndDate:        endDate,
				TotalSignedAmount:        totalSignedAmount,
				SignedCount:              totalSignedCount,
				TotalJumiaCommission:     totalJumiaCommission,
				TotalOutboundFee:         totalOutboundFee,
				TotalStorageFee:          totalStorageFee,
				TotalCloudRideCommission: totalCloudRideCommission,
				TotalPyvioFee:            totalPyvioFee,
				TotalReviewFee:           totalReviewFee,
				ReceivedAmount:           totalReceivedAmount,
				ActualSettleAmount:       totalSettleLocal,
				ActualSettleCNY:          totalSettleCNY,
				SettlementStatus:         "待结算",
			}
			if err := tx.Create(&summary).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("创建 UserSettlementSummary 失败: %w", err)
			}
		} else {
			tx.Rollback()
			return fmt.Errorf("查询 UserSettlementSummary 失败: %w", err)
		}

		// 6. 写 details
		for _, d := range details {
			d.SummaryID = summary.ID
			var existing models.UserSettlementDetail
			err := tx.Where("user_id=? AND seller_sku=? AND country_code=? AND settlement_start_date=? AND settlement_end_date=?",
				userid, d.SellerSKU, d.CountryCode, startDate, endDate).First(&existing).Error

			if err == nil {
				existing.SummaryID = summary.ID
				existing.TotalSignedAmount = d.TotalSignedAmount
				existing.SettlementRate = d.SettlementRate
				existing.SignedCount = d.SignedCount
				existing.JumiaCommission = d.JumiaCommission
				existing.CommissionRate = d.CommissionRate
				existing.OutboundFee = d.OutboundFee
				existing.StorageFee = d.StorageFee
				existing.ReceivedAmount = d.ReceivedAmount
				existing.CloudRideCommission = d.CloudRideCommission
				existing.CloudRideCommissionRate = d.CloudRideCommissionRate
				existing.PyvioFee = d.PyvioFee
				existing.PyvioFeeRate = d.PyvioFeeRate
				existing.ReviewFee = d.ReviewFee
				existing.ActualSettleAmount = d.ActualSettleAmount
				existing.ActualSettleCNY = d.ActualSettleCNY

				if err := tx.Save(&existing).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("更新 SettlementDetail 失败: %w", err)
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(d).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("创建 SettlementDetail 失败: %w", err)
				}
			} else {
				tx.Rollback()
				return fmt.Errorf("查询 SettlementDetail 失败: %w", err)
			}
		}

		global.Log.Infof("用户 %d 国家 %s 汇总完成: 总签收=%.2f, 实际结算=%.2f USD / %.2f CNY",
			userid, country, totalSignedAmount, totalSettleLocal, totalSettleCNY)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	return nil
}

// safeDiv 避免除零
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// getCommissionRates 查询指定用户在指定国家、时间、SKU 下的费率
func getCommissionRates(tx *gorm.DB, userID uint, countryCode, sellerSKU string, startDate, endDate time.Time) (float64, float64) {
	var config models.UserSettlementConfig
	//默认抽佣都是0.1， 10%
	cloudRide := 0.1
	//默认加纳硬编码汇率
	settleRate := 0.5
	//硬编码配置
	if countryCode == "NG" {
		settleRate = 0.004
	}
	if countryCode == "KE" {
		settleRate = 0.05
	}

	// 1. 用户特定配置
	err := tx.Where("user_id = ? AND country_code = ? ", userID, countryCode).
		Where("seller_sku = ? OR seller_sku = ''OR seller_sku IS NULL", sellerSKU).
		Where("? BETWEEN COALESCE(settlement_start_date, ?) AND COALESCE(settlement_end_date, ?)",
			startDate, startDate, endDate).
		Order("seller_sku DESC").
		First(&config).Error

	if err == nil {
		if config.CloudRideCommissionRate != nil {
			cloudRide = *config.CloudRideCommissionRate
		}
		if config.SettlementRate != nil {
			settleRate = *config.SettlementRate
		}

		return cloudRide, settleRate
	}

	// 2. 默认配置（UserID=0）
	global.Log.Warnf("用户 %d 未找到 %s 的配置，尝试使用默认配置", userID, countryCode)

	err = tx.Where("user_id = ? AND country_code = ? ", 0, countryCode).
		Where("seller_sku = ? OR seller_sku = ''OR seller_sku IS NULL", sellerSKU).
		Where("? BETWEEN COALESCE(settlement_start_date, ?) AND COALESCE(settlement_end_date, ?)",
			startDate, startDate, endDate).
		Order("seller_sku DESC").
		First(&config).Error

	if err == nil {
		if config.CloudRideCommissionRate != nil {
			cloudRide = *config.CloudRideCommissionRate
		}
		if config.SettlementRate != nil {
			settleRate = *config.SettlementRate
		}

		global.Log.Infof("使用默认配置（UserID=0）: %s, CloudRide=%.4f, Rate=%.4f", countryCode, cloudRide, settleRate)
		return cloudRide, settleRate
	}

	// 3. fallback
	global.Log.Errorf("⚠️ 未找到 %s 的任何配置（用户 %d），使用硬编码 fallback", countryCode, userID)
	return cloudRide, settleRate
}
