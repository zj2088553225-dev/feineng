package models

import "time"

// 自定义MODEL，没有用gorm的MODEL,因为我们不需要逻辑删除
type MODEL struct {
	ID        uint      `gorm:"primarykey" json:"id,select($any)" structs:"-"` // 主键ID
	CreatedAt time.Time `json:"created_at,select($any)" structs:"-"`           // 创建时间
	UpdatedAt time.Time `json:"-" structs:"-"`                                 // 更新时间
}

// 用于分页查询
type PageInfo struct {
	Page  int    `form:"page"`  //页码
	Key   string `form:"key"`   //搜索关键字
	Limit int    `form:"limit"` //每页显示多少条
	Sort  string `form:"sort"`  //排序
}
