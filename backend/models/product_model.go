package models

import (
	"time"
)

// 自定义的用户产品数据表结构
// 自定义的用户产品数据表结构（优化版）
type UserProduct struct {
	// ✅ 新增：真正的主键
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// 保留原有字段，仅修改 JumiaSku
	UserID   uint   `gorm:"index:idx_user_product_user_id;not null" json:"user_id"`
	UserName string `gorm:"size:36" json:"user_name"`
	NameEn   string `gorm:"type:text;not null" json:"name_en"`
	NameZh   string `gorm:"type:varchar(1000);default:'未编辑'" json:"name_zh"`

	// ✅ SellerSku 保持索引
	SellerSku string `gorm:"size:100;index:idx_seller_sku;not null" json:"seller_sku"`

	// ❌ 移除 primaryKey，改为唯一索引（保持唯一性）
	JumiaSku string `gorm:"size:255;not null;uniqueIndex:idx_jumia_sku" json:"jumia_sku"`

	CountryName     string     `gorm:"type:text;not null" json:"country_name"`
	Inventory       int        `gorm:"default:0;not null" json:"inventory"`
	BuyUrl          string     `gorm:"size:255" json:"buy_url"`
	SellUrl         string     `gorm:"size:255" json:"sell_url"`
	LocalCurrency   string     `gorm:"size:10;not null" json:"local_currency"`
	LocalPriceValue float64    `gorm:"not null" json:"local_price_value"`
	PriceCurrency   string     `gorm:"size:10;not null" json:"price_currency"`
	PriceValue      float64    `gorm:"not null" json:"price_value"`
	SaleLocalValue  *float64   `json:"sale_local_value"`
	SaleValue       *float64   `json:"sale_value"`
	SaleStartAt     *time.Time `json:"sale_start_at"`
	SaleEndAt       *time.Time `json:"sale_end_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	//用于区分更新产品的账号来源
	Account string `gorm:"default:1;not null" json:"account"`
}
