package models

import "time"

// 合营合伙人表
type CooperationPartner struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"` // 关联用户 ID
	Rate      float64   `gorm:"default:0.8" json:"rate"`             // 合营比例（默认 0.8 = 80%）
	Note      string    `gorm:"size:255" json:"note"`                // 备注（可选）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      UserModel `gorm:"foreignKey:UserID" json:"user"` // 关联用户信息
}
