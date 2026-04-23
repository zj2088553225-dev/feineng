package models

import (
	"time"
)

type ServiceStatus struct {
	ID        uint      `gorm:"primaryKey"`                                           // 主键，自增
	Service   string    `gorm:" type:varchar(100);uniqueIndex;not null;comment:服务名称"` // 服务名称，唯一
	Status    string    `gorm:"type:varchar(20);default:'未知';comment:服务状态"`           // 状态，如 正常/错误
	Message   string    `gorm:"type:text;comment:状态描述"`                               // 状态描述，错误原因等
	UpdatedAt time.Time `gorm:"autoUpdateTime"`                                       // 更新时间
}
