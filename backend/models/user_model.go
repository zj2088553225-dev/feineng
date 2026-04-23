package models

import (
	"backend/models/ctype"
)

// UserModel 用户表
type UserModel struct {
	MODEL
	UserName string     `gorm:"size:36" json:"user_name"`                  // 用户名
	Password string     `gorm:"size:128" json:"pass_word"`                 // 密码
	Role     ctype.Role `gorm:"size:4;default:3" json:"role,select(info)"` // 权限  1 管理员  2 普通合伙人

	// 添加关联：一个用户有多个 SellerSku 绑定
	SellerSkus []UserSellerSkuModel `gorm:"foreignKey:UserID" json:"seller_skus,omitempty"`
}

// 合伙人与其绑定的sellersku的关系表
type UserSellerSkuModel struct {
	UserID    uint   `gorm:"index;primaryKey;autoIncrement:false" json:"user_id"` // 建议加 index 提升查询性能//用户id
	SellerSku string `gorm:"primaryKey;size:100;index" json:"seller_sku"`
}

//分公司管理员能够管理的用户
