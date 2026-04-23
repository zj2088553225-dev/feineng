package user_api

import (
	"backend/global"
	"backend/models"
	"backend/models/res"
	"backend/service/common"
	"backend/service/settlement"
	"backend/untils/jwts"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

// 管理员查看结算任务配置
func (UserApi) AdminSettlementConfig(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	condition := models.UserSettlementConfig{}
	if pageInfo.PartnerID > 0 {
		condition = models.UserSettlementConfig{
			UserID: pageInfo.PartnerID,
		}
	}

	// 设置选项
	option := common.Option{
		PageInfo: pageInfo,
		CustomCond: func(db *gorm.DB) *gorm.DB {
			// 精确匹配 country_name 和 status
			if pageInfo.CountryCode != "" {
				db = db.Where("country_code = ?", pageInfo.CountryCode)
			}
			// 处理日期范围
			startDate := pageInfo.StartDate
			endDate := pageInfo.EndDate
			if startDate != "" && len(startDate) == 10 {
				startDate += " 00:00:00.000"
			}
			if endDate != "" && len(endDate) == 10 {
				endDate += " 23:59:59.000"
			}

			if startDate != "" && endDate != "" {
				db = db.Where("settlement_start_date <= ?", endDate).
					Where("settlement_end_date >= ?", startDate)
			}
			return db
		},
	}

	list, count, err := common.ComList(condition, option)

	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		global.Log.Error(err.Error())
		return
	}

	res.OkWithList(list, count, c)
}

// 国家筛选配置和结算，周期筛选配置和结算，使用合伙人id筛选配置和结算
// SettlementConfigRequest 用于接收前端请求
type SettlementConfigRequest struct {
	UserID      uint   `json:"user_id"` // 0 表示通用配置
	CountryCode string `json:"country_code" binding:"required,len=2,uppercase"`
	SellerSKU   string `json:"seller_sku"` // 可为空，表示通用 SKU 配置

	SettlementStartDate time.Time `json:"settlement_start_date" binding:"required"`
	SettlementEndDate   time.Time `json:"settlement_end_date" binding:"required"`

	CloudRideCommissionRate *float64 `json:"cloud_ride_commission_rate" binding:"omitempty,min=0,max=1"`
	SettlementRate          *float64 `json:"settlement_rate" binding:"omitempty,gt=0"`
}

// 管理员增加配置
func (UserApi) AdminAddSettlementConfig(c *gin.Context) {
	var req SettlementConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	// 校验：结束时间 >= 开始时间
	if req.SettlementEndDate.Before(req.SettlementStartDate) {
		res.FailWithMessage("结束时间不能早于开始时间", c)
		return
	}

	db := global.DB
	var config models.UserSettlementConfig
	//查询是否有相同配置
	cond := db.Where("user_id = ? AND country_code = ? AND seller_sku = ?", req.UserID, req.CountryCode, req.SellerSKU)

	cond = cond.Where("settlement_start_date = ?", req.SettlementStartDate).
		Where("settlement_end_date = ?", req.SettlementEndDate)

	if err := cond.First(&config).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			global.Log.Error("数据库查询失败: " + err.Error())
			res.FailWithMessage("系统错误", c)
			return
		}
		//查询不到创建新配置
		config = models.UserSettlementConfig{
			UserID:                  req.UserID,
			CountryCode:             req.CountryCode,
			SellerSKU:               req.SellerSKU,
			SettlementStartDate:     &req.SettlementStartDate,
			SettlementEndDate:       &req.SettlementEndDate,
			CloudRideCommissionRate: req.CloudRideCommissionRate,
			SettlementRate:          req.SettlementRate,
		}
		if err := db.Create(&config).Error; err != nil {
			global.Log.Error("创建配置失败: " + err.Error())
			res.FailWithMessage("创建失败", c)
			return
		}
	} else {
		res.FailWithMessage("配置已经存在", c)
		return
	}

	res.OkWithMessage("增加配置成功", c)
}

type SettlementConfigUpdateRequest struct {
	ID                      uint     `json:"id"`
	CloudRideCommissionRate *float64 `json:"cloud_ride_commission_rate" binding:"omitempty,min=0,max=1"`
	SettlementRate          *float64 `json:"settlement_rate" binding:"omitempty,gt=0"`
}

// 管理员修改配置
func (UserApi) AdminUpdateSettlementConfig(c *gin.Context) {
	var req SettlementConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	var config models.UserSettlementConfig
	//查询是否有相同配置
	err := global.DB.Where("id = ?", req.ID).First(&config).Error
	if err != nil {
		global.Log.Error(err.Error())
		res.FailWithMessage("查询不到配置信息", c)
		return
	}
	config.CloudRideCommissionRate = req.CloudRideCommissionRate
	config.SettlementRate = req.SettlementRate
	err = global.DB.Save(&config).Error
	if err != nil {
		global.Log.Error(err.Error())
		res.FailWithMessage("保存修改配置信息失败", c)
		return
	}

	res.OkWithMessage("修改配置成功", c)
}

// 管理员删除配置
func (UserApi) AdminDeleteSettlementConfig(c *gin.Context) {
	ID := c.Param("id")

	var config models.UserSettlementConfig
	//查询是否有相同配置
	err := global.DB.Where("id = ?", ID).First(&config).Error
	if err != nil {
		global.Log.Error(err.Error())
		res.FailWithMessage("查询不到配置信息", c)
		return
	}

	err = global.DB.Delete(&config).Error
	if err != nil {
		global.Log.Error(err.Error())
		res.FailWithMessage("删除配置失败", c)
		return
	}

	res.OkWithMessage("删除配置成功", c)
}

// 手动开启任务，重新计算结算数据
func (UserApi) TriggerSettlementCalculation(c *gin.Context) {
	ID := c.Param("id")
	var config models.UserSettlementConfig
	err := global.DB.Where("id = ?", ID).First(&config).Error // ✅ 用 First，语义更明确
	if err != nil {
		global.Log.Error("查询配置失败: " + err.Error())
		res.FailWithMessage("无效的配置信息", c)
		return
	}

	// ✅ 1. 更新状态为“运行中”
	config.Status = "运行中"
	err = global.DB.Save(&config).Error
	if err != nil {
		global.Log.Error("更新配置状态失败: " + err.Error())
		res.FailWithMessage("保存配置状态失败", c)
		return
	}

	// ✅ 2. 启动异步任务（传入 config 的副本）
	go func(cfg models.UserSettlementConfig) { // ✅ 传入副本
		var err error
		if cfg.UserID == 0 {
			err = settlement.CalculationSettlementData(*cfg.SettlementStartDate, *cfg.SettlementEndDate)
		} else {
			err = settlement.ProcessUserSettlement(cfg.UserID, *cfg.SettlementStartDate, *cfg.SettlementEndDate)
		}

		// ✅ 3. 更新最终状态
		status := "运行成功"
		if err != nil {
			status = "运行失败"
			global.Log.Error(fmt.Sprintf("结算任务执行失败 [ConfigID=%d]: %v", cfg.ID, err))
		}

		// ✅ 4. 重新查询 + 更新，避免并发写冲突
		var updatedConfig models.UserSettlementConfig
		dbErr := global.DB.Where("id = ?", cfg.ID).First(&updatedConfig).Error
		if dbErr != nil {
			global.Log.Error(fmt.Sprintf("重新查询配置失败 [ConfigID=%d]: %v", cfg.ID, dbErr))
			return
		}

		updatedConfig.Status = status
		dbErr = global.DB.Save(&updatedConfig).Error
		if dbErr != nil {
			global.Log.Error(fmt.Sprintf("保存最终状态失败 [ConfigID=%d]: %v", cfg.ID, dbErr))
		}
	}(config) // ✅ 传入当前 config 副本

	// ✅ 返回成功
	res.OkWithData(gin.H{
		"message": "结算任务已启动，请及时查询状态",
	}, c)
}

// 管理员查看所有用户的结算信息，结算信息以及结算详情
func (UserApi) AdminSettlementView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	condition := models.UserSettlementSummary{}
	if pageInfo.PartnerID > 0 {
		condition = models.UserSettlementSummary{
			UserID: pageInfo.PartnerID,
		}
	}

	// 设置选项
	option := common.Option{
		PageInfo: pageInfo,
		Preload:  []string{"Details"}, // ✅ 关键：加载明细
		CustomCond: func(db *gorm.DB) *gorm.DB {
			// 精确匹配 country_name 和 status
			if pageInfo.CountryCode != "" {
				db = db.Where("country_code = ?", pageInfo.CountryCode)
			}
			if pageInfo.PaidStatus != "" {
				db = db.Where("settlement_status = ?", pageInfo.PaidStatus)
			}
			// 处理日期范围
			startDate := pageInfo.StartDate
			endDate := pageInfo.EndDate
			if startDate != "" && len(startDate) == 10 {
				startDate += " 00:00:00.000"
			}
			if endDate != "" && len(endDate) == 10 {
				endDate += " 23:59:59.000"
			}

			if startDate != "" && endDate != "" {
				db = db.Where("settlement_start_date <= ?", endDate).
					Where("settlement_end_date >= ?", startDate)
			}
			return db
		},
	}

	list, count, err := common.ComList(condition, option)
	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		global.Log.Error(err.Error())
		return
	}

	res.OkWithList(list, count, c)
}

type EditSettlementRequest struct {
	SettlementID     uint   `json:"settlement_id" binding:"required,gt=0"`
	SettlementStatus string `json:"settlement_status" binding:"required,oneof=待结算 已结算"`
}

// AdminEditSettlementView 允许管理员更新结算单状态
func (UserApi) AdminEditSettlementView(c *gin.Context) {
	var req EditSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.FailWithError(err, &req, c)
		return
	}

	var summary models.UserSettlementSummary
	// 查询结算记录
	err := global.DB.Where("id = ?", req.SettlementID).First(&summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailWithMessage("结算记录不存在", c)
			return
		}
		global.Log.Error("数据库查询错误: " + err.Error())
		res.FailWithMessage("系统错误", c)
		return
	}

	// 更新状态
	summary.SettlementStatus = req.SettlementStatus
	if err := global.DB.Save(&summary).Error; err != nil {
		global.Log.Error("更新结算状态失败: " + err.Error())
		res.FailWithMessage("更新失败", c)
		return
	}

	res.OkWithMessage("结算状态更新成功", c)
}

// 用户查看自己的结算信息，结算信息以及结算详情
func (UserApi) GetUserSettlementView(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.CustomClaims)

	// 查询条件：仅当前用户
	condition := models.UserSettlementSummary{
		UserID: claims.UserID,
	}
	// 设置选项
	option := common.Option{
		PageInfo: pageInfo,
		Preload:  []string{"Details"}, // ✅ 关键：加载明细
		CustomCond: func(db *gorm.DB) *gorm.DB {
			// 精确匹配 country_name 和 status
			if pageInfo.CountryCode != "" {
				db = db.Where("country_code = ?", pageInfo.CountryCode)
			}
			if pageInfo.PaidStatus != "" {
				db = db.Where("settlement_status = ?", pageInfo.PaidStatus)
			}
			// 处理日期范围
			startDate := pageInfo.StartDate
			endDate := pageInfo.EndDate
			if startDate != "" && len(startDate) == 10 {
				startDate += " 00:00:00.000"
			}
			if endDate != "" && len(endDate) == 10 {
				endDate += " 23:59:59.000"
			}

			if startDate != "" && endDate != "" {
				db = db.Where("settlement_start_date <= ?", endDate).
					Where("settlement_end_date >= ?", startDate)
			}
			return db
		},
	}

	list, count, err := common.ComList(condition, option)
	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		global.Log.Error(err.Error())
		return
	}

	res.OkWithList(list, count, c)
}

// 获取周期内的各个国家的总和以及公司利润
func (UserApi) GetUserSettlementTotal(c *gin.Context) {
	var pageInfo models.PageInfo
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		res.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}

	// 定义本地时区（CST，UTC+8）
	var loc = time.FixedZone("CST", 8*3600) // 或直接用 time.Local

	// ✅ 正确解析：在本地时区下解析
	startDate, err := time.ParseInLocation("2006-01-02", pageInfo.StartDate, loc)
	if err != nil {
		res.FailWithMessage("开始日期格式错误", c)
		return
	}

	// ✅ 解析结束时间：包含整天
	endDateStr := pageInfo.EndDate + " 23:59:59"
	endDate, err := time.ParseInLocation("2006-01-02 15:04:05", endDateStr, loc)
	if err != nil {
		res.FailWithMessage("结束日期格式错误", c)
		return
	}

	// ✅ 调用聚合函数
	////结算周期的起止时间完全落在用户选择的时间范围内，才会被统计
	data, err := GetCountryPeriodSummary(pageInfo.CountryCode, startDate, endDate)
	if err != nil {
		global.Log.Error(err.Error())
		res.FailWithMessage("聚合周期内结算数据失败", c)
		return
	}

	condition := models.UserSettlementSummary{}

	// 设置选项
	option := common.Option{
		PageInfo: pageInfo,
		CustomCond: func(db *gorm.DB) *gorm.DB {
			// 精确匹配 country_code
			if pageInfo.CountryCode != "" {
				db = db.Where("country_code = ?", pageInfo.CountryCode)
			}
			// 处理日期范围
			startDate := pageInfo.StartDate
			endDate := pageInfo.EndDate
			if startDate != "" && len(startDate) == 10 {
				startDate += " 00:00:00.000"
			}
			if endDate != "" && len(endDate) == 10 {
				endDate += " 23:59:59.000"
			}
			//结算周期的起止时间完全落在用户选择的时间范围内，才会被统计
			if startDate != "" && endDate != "" {
				db = db.Where("settlement_start_date >= ?", startDate).
					Where("settlement_end_date <= ?", endDate)
			}
			return db
		},
	}

	list, _, err := common.ComList(condition, option)
	if err != nil {
		res.FailWithMessage("查询失败: "+err.Error(), c)
		global.Log.Error(err.Error())
		return
	}
	// ✅ 查询合营合伙人表
	var coopPartners []models.CooperationPartner
	if err := global.DB.Find(&coopPartners).Error; err != nil {
		global.Log.Error("查询合营合伙人失败: " + err.Error())
	}

	// 建立 user_id -> CooperationPartner 对应关系，方便获取 rate 和 note
	coopMap := make(map[int64]models.CooperationPartner)
	for _, p := range coopPartners {
		coopMap[int64(p.UserID)] = p
	}

	// ✅ 收入计算
	var totalActualSettleAmount float64
	var totalCooperationDeduction float64
	// 存储每个合营合伙人的计算明细
	var perPartnerCooperation []map[string]interface{}

	for _, summary := range list {

		if partner, ok := coopMap[int64(summary.UserID)]; ok {
			// ⚡ 如果在合营表里，就按比例扣除
			// ⚡ 如果在合营表里，就按比例扣除
			//这是公司收入要减去的合营合伙人的金额
			//特殊情况rate=1的话，为团队账号
			var coopAmount float64
			if partner.Rate != 1 {
				coopAmount = summary.ActualSettleAmount * partner.Rate
			} else {
				coopAmount = summary.ReceivedAmount - summary.TotalPyvioFee
			}
			totalCooperationDeduction += coopAmount

			perPartnerCooperation = append(perPartnerCooperation, map[string]interface{}{
				"actual_settle_amount": summary.ActualSettleAmount,
				"cooperation_amount":   coopAmount,
				"rate":                 partner.Rate,
				"note":                 partner.Note,
			})

		} else {
			//这是公司收入要减去的合伙人的金额
			totalActualSettleAmount += summary.ActualSettleAmount
		}
	}

	// ✅ 公司利润公式
	companyProfits := data.ReceivedAmount - totalActualSettleAmount - totalCooperationDeduction
	data.CompanyProfits = companyProfits
	// 后端定义一个汇率 map，根据国家代码取汇率
	var exchangeRates = map[string]float64{
		"GH": 0.5,   // 加纳当地货币 → 人民币
		"NG": 0.004, // 尼日利亚奈拉 → 人民币
		"KE": 0.05,  // 肯尼亚先令 → 人民币
	}

	companyProfitsCNY := companyProfits * exchangeRates[pageInfo.CountryCode]

	// ✅ 组装计算详情
	calculationDetail := map[string]interface{}{
		"ReceivedAmount":            data.ReceivedAmount,       // 实际到账
		"TotalActualSettleAmount":   totalActualSettleAmount,   // 合伙人结算
		"TotalCooperationDeduction": totalCooperationDeduction, // 合营扣除
		"PerPartnerCooperation":     perPartnerCooperation,     // 每个合营合伙人的明细
		"Formula":                   "CompanyProfits = ReceivedAmount - TotalActualSettleAmount - Σ(Partner.SettleAmount * Partner.Rate)",
		"Result":                    companyProfits,                      // 最终结果
		"Rate":                      exchangeRates[pageInfo.CountryCode], // 汇率
		"ResultCNY":                 companyProfitsCNY,                   // 汇率换算后的结果
	}

	// ✅ 响应增加计算详情
	res.OkWithData(gin.H{
		"summary":           data,
		"calculationDetail": calculationDetail,
	}, c)
}

type CountryPeriodSummary struct {
	CountryCode         string    `json:"country_code"`
	SettlementStartDate time.Time `json:"settlement_start_date"`
	SettlementEndDate   time.Time `json:"settlement_end_date"`

	// 汇总指标
	TotalSignedAmount        float64 `json:"total_signed_amount"`
	TotalSignedCount         float64 `json:"total_signed_count"`
	TotalJumiaCommission     float64 `json:"total_jumia_commission"`
	TotalOutboundFee         float64 `json:"total_outbound_fee"`
	TotalStorageFee          float64 `json:"total_storage_fee"`
	ReceivedAmount           float64 `json:"received_amount"`
	TotalCloudRideCommission float64 `json:"total_cloud_ride_commission"`
	TotalPyvioFee            float64 `json:"total_pyvio_fee"`
	TotalReviewFee           float64 `json:"total_review_fee"`
	ActualSettleAmount       float64 `json:"actual_settle_amount"`
	ActualSettleCNY          float64 `json:"actual_settle_cny"`
	UserCount                int64   `json:"user_count"` // 统计参与汇总的用户数

	CompanyProfits float64 `json:"company_profits"` //公司利润
}

// GetCountryPeriodSummary 获取国家周期汇总数据（完全包含逻辑）
// GetCountryPeriodSummary 获取国家周期汇总数据（完全包含逻辑）
func GetCountryPeriodSummary(countryCode string, startDate, endDate time.Time) (*CountryPeriodSummary, error) {
	var summary CountryPeriodSummary

	// ✅ 1. 记录输入参数（直接使用传入的时间）
	global.Log.Info(fmt.Sprintf(
		"开始聚合国家结算数据: countryCode=%s, 查询周期=%s 至 %s",
		countryCode,
		startDate.Format("2006-01-02 15:04:05"),
		endDate.Format("2006-01-02 15:04:05"),
	))

	// ✅ 2. SQL：使用 ? 占位符，避免字符串拼接时间
	query := `
        SELECT 
            COALESCE(SUM(total_signed_amount), 0) AS total_signed_amount,
            COALESCE(SUM(signed_count), 0) AS total_signed_count,
            COALESCE(SUM(total_jumia_commission), 0) AS total_jumia_commission,
            COALESCE(SUM(total_outbound_fee), 0) AS total_outbound_fee,
            COALESCE(SUM(total_storage_fee), 0) AS total_storage_fee,
            COALESCE(SUM(received_amount), 0) AS received_amount,
            COALESCE(SUM(total_cloud_ride_commission), 0) AS total_cloud_ride_commission,
            COALESCE(SUM(total_pyvio_fee), 0) AS total_pyvio_fee,
            COALESCE(SUM(total_review_fee), 0) AS total_review_fee,
            COALESCE(SUM(actual_settle_amount), 0) AS actual_settle_amount,
            COALESCE(SUM(actual_settle_cny), 0) AS actual_settle_cny,
            COALESCE(COUNT(DISTINCT user_id), 0) AS user_count
        FROM user_settlement_summaries
        WHERE country_code = ?
          AND settlement_start_date >= ?
          AND settlement_end_date <= ?
    `

	startTime := time.Now()

	// ✅ 3. 参数顺序：前3个是 SELECT 的字段，后3个是 WHERE 条件
	err := global.DB.Raw(query,
		countryCode, // ?4: WHERE country_code
		startDate,   // ?5: WHERE start >= ?
		endDate,     // ?6: WHERE end <= ?
	).Scan(&summary).Error

	duration := time.Since(startTime)
	global.Log.Info(fmt.Sprintf("SQL查询完成，耗时: %v, 错误: %v", duration, err))

	if err != nil {
		global.Log.Error(fmt.Sprintf(
			"聚合国家结算数据失败: %v, 参数: %s, %s, %s",
			err,
			countryCode,
			startDate.Format("2006-01-02 15:04:05"),
			endDate.Format("2006-01-02 15:04:05"),
		))
		return nil, err
	}

	summary.CountryCode = countryCode
	summary.SettlementStartDate = startDate
	summary.SettlementEndDate = endDate

	// ✅ 5. 输出结果日志
	global.Log.Info(fmt.Sprintf(
		"聚合成功: 国家=%s, 总金额=%.2f, 用户数=%d, 查询周期=%s 至 %s",
		summary.CountryCode,
		summary.ActualSettleAmount,
		summary.UserCount,
		summary.SettlementStartDate.Format("2006-01-02"),
		summary.SettlementEndDate.Format("2006-01-02"),
	))

	return &summary, nil
}
