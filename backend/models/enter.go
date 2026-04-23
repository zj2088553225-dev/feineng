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
	//订单查询使用
	// 新增：合伙人 ID（可选）
	PartnerID uint `json:"partner_id" form:"partner_id"`
	//新增国家名查询
	CountryName string `json:"country_name" form:"country_name"`
	//状态查询
	Status string `json:"status" form:"status"`

	//交易查询使用
	//交易国家码
	CountryCode string `json:"country_code" form:"country_code"`
	//支付状态
	PaidStatus string `json:"paid_status" form:"paid_status"`
	//增加日期查找
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
	//交易类型分类
	TransactionType string `json:"transaction_type" form:"transaction_type"`

	//社媒订单使用
	//时间筛选，jumiasku搜索，Order  Done 订单状态筛选，合伙人筛选，订单详情状态筛选，
	//获取不同的国家的社媒订单
	CustomizeOrderType string `json:"customize_order_type" form:"customize_order_type"`
	//筛选合伙人
	Person string `json:"person" form:"person"`
	//订单拨打电话的回复状态
	OrderStatus string `json:"order_status" form:"order_status"`
}
