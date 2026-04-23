package models

import (
	"time"
)

type ServiceStatus struct {
	ID        uint      `gorm:"primaryKey"`
	Service   string    // 服务名称，如 "jumia_order_sync"
	Status    string    // 状态
	Message   string    // 状态描述
	UpdatedAt time.Time // GORM 会自动在更新时填充此字段
}
