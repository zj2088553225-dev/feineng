package models

import (
	"time"
)

type ServiceStatus struct {
	ID        uint      `gorm:"primaryKey"`
	Service   string    `gorm:"type:varchar(100);uniqueIndex;not null;comment:服务名称"`
	Status    string    `gorm:"type:varchar(20);default:'未知';comment:服务状态"`
	Message   string    `gorm:"type:text;comment:状态描述"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
