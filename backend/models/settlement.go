package models

import "time"

// UserSettlementDetail 结算明细表（按 SellerSKU 细粒度结算）
type UserSettlementDetail struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 基础信息
	UserID              uint      `gorm:"not null;index:idx_detail_user" json:"user_id"`            // 关联用户ID
	SummaryID           uint      `gorm:"not null;index:idx_detail_summary" json:"summary_id"`      // 关联汇总ID（新增关键字段）
	SellerSKU           string    `gorm:"size:255;not null;index:idx_detail_sku" json:"seller_sku"` // 绑定的SKU
	CountryCode         string    `gorm:"size:10;index:idx_detail_country" json:"country_code"`     // 国家代码
	SettlementStartDate time.Time `gorm:"index:idx_detail_period" json:"settlement_start_date"`     // 结算周期开始
	SettlementEndDate   time.Time `gorm:"index:idx_detail_period" json:"settlement_end_date"`       // 结算周期结束

	// 交易数据
	TotalSignedAmount float64 `gorm:"type:decimal(50,2);default:0" json:"total_signed_amount"` // 总签收金额（当地货币）
	SettlementRate    float64 `gorm:"type:decimal(50,6);default:0.55" json:"settlement_rate"`  // 结算汇率（当地货币 -> CNY）
	SignedCount       float64 `gorm:"default:0" json:"signed_count"`                           // 签收笔数，item credit数

	// 平台费用
	JumiaCommission float64 `gorm:"type:decimal(50,2);default:0" json:"jumia_commission"`    // Jumia抽佣（当地货币）
	CommissionRate  float64 `gorm:"type:decimal(5,4);default:0.1000" json:"commission_rate"` // jumia抽佣比例
	OutboundFee     float64 `gorm:"type:decimal(50,2);default:0" json:"outbound_fee"`        // 妥头出库费（当地货币）
	StorageFee      float64 `gorm:"type:decimal(50,2);default:0" json:"storage_fee"`         // 库存费（当地货币）

	// 到账与第三方费用
	ReceivedAmount          float64 `gorm:"type:decimal(50,2);default:0" json:"received_amount"`           // 实际到账金额（当地货币）
	CloudRideCommission     float64 `gorm:"type:decimal(50,2);default:0" json:"cloud_ride_commission"`     // 云驰抽佣（当地货币）
	CloudRideCommissionRate float64 `gorm:"type:decimal(5,4);default:0" json:"cloud_ride_commission_rate"` // 云驰抽佣比例
	PyvioFee                float64 `gorm:"type:decimal(50,5);default:0" json:"pyvio_fee"`                 // Pyvio手续费（当地货币）
	PyvioFeeRate            float64 `gorm:"type:decimal(5,4);default:0.0080" json:"pyvio_fee_rate"`        // 手续费比例

	ReviewFee float64 `gorm:"type:decimal(50,2);default:0" json:"review_fee"` // 审单费用（当地货币）

	// 最终结算
	ActualSettleAmount float64 `gorm:"type:decimal(50,2);default:0" json:"actual_settle_amount"` // 实际结算金额（当地货币）
	ActualSettleCNY    float64 `gorm:"type:decimal(50,2);default:0" json:"actual_settle_cny"`    // 实际结算金额（CNY）
}

// UserSettlementSummary 用户结算汇总表（按用户+周期汇总）
type UserSettlementSummary struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 基础信息
	UserID              uint      `gorm:"not null;uniqueIndex:idx_summary_user_period" json:"user_id"`      // 用户ID
	CountryCode         string    `gorm:"size:10;uniqueIndex:idx_summary_user_period" json:"country_code"`  // 国家代码（可选：取自明细平均或用户设置）
	SettlementStartDate time.Time `gorm:"uniqueIndex:idx_summary_user_period" json:"settlement_start_date"` // 周期开始
	SettlementEndDate   time.Time `gorm:"uniqueIndex:idx_summary_user_period" json:"settlement_end_date"`   // 周期结束

	// 汇总交易数据
	TotalSignedAmount float64 `gorm:"type:decimal(50,2);default:0" json:"total_signed_amount"` // 所有SKU总签收金额（当地货币）
	SignedCount       float64 `gorm:"default:0" json:"signed_count"`                           // 总签收笔数

	// 汇总平台费用
	TotalJumiaCommission float64 `gorm:"type:decimal(50,2);default:0" json:"total_jumia_commission"` // Jumia总抽佣
	TotalOutboundFee     float64 `gorm:"type:decimal(50,2);default:0" json:"total_outbound_fee"`     // 总出库费
	TotalStorageFee      float64 `gorm:"type:decimal(50,2);default:0" json:"total_storage_fee"`      // 总库存费

	// 汇总第三方费用
	ReceivedAmount           float64 `gorm:"type:decimal(50,2);default:0" json:"received_amount"`             // 实际到账总金额（当地货币）
	TotalCloudRideCommission float64 `gorm:"type:decimal(50,2);default:0" json:"total_cloud_ride_commission"` // 云驰总抽佣
	TotalPyvioFee            float64 `gorm:"type:decimal(50,2);default:0" json:"total_pyvio_fee"`             // Pyvio总手续费
	TotalReviewFee           float64 `gorm:"type:decimal(50,2);default:0" json:"total_review_fee"`            // 审单总费用

	// 到账与结算
	ActualSettleAmount float64 `gorm:"type:decimal(50,2);default:0" json:"actual_settle_amount"` // 实际结算总金额（当地货币）
	ActualSettleCNY    float64 `gorm:"type:decimal(50,2);default:0" json:"actual_settle_cny"`    // 实际结算总金额（CNY）

	// 状态
	SettlementStatus string `gorm:"size:50;default:'pending';index:idx_summary_status" json:"settlement_status"`

	// 可选：反向关联明细列表
	Details []UserSettlementDetail `gorm:"foreignKey:SummaryID"`
}

// UserSettlementConfig 用户抽佣与服务费率配置表
// 用于支持：按用户、国家、SKU、周期 的差异化配置（含结算汇率）
type UserSettlementConfig struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 配置维度（联合索引支持精确匹配）
	UserID      uint   `gorm:"not null;index:idx_config_user_country_sku_period" json:"user_id"`
	CountryCode string `gorm:"size:10;not null;index:idx_config_user_country_sku_period" json:"country_code"`
	SellerSKU   string `gorm:"size:255;index:idx_config_user_country_sku_period" json:"seller_sku"` // 空字符串表示通用SKU

	// 生效周期（对应结算周期）
	SettlementStartDate *time.Time `gorm:"index:idx_config_user_country_sku_period" json:"settlement_start_date"`
	SettlementEndDate   *time.Time `gorm:"index:idx_config_user_country_sku_period" json:"settlement_end_date"`

	// 各项费率（指针表示可选覆盖）
	CloudRideCommissionRate *float64 `gorm:"type:decimal(5,4)" json:"cloud_ride_commission_rate"` // 云驰服务费率

	// ✅ 重点：结算汇率（当地货币 → CNY）
	SettlementRate *float64 `gorm:"type:decimal(50,6)" json:"settlement_rate"` // 结算汇率（当地货币 → CNY）

	Status string `gorm:"size:25;default:初始化;index:idx_status" json:"status"` // 状态，初始化，运行中，运行成功，运行失败
}
