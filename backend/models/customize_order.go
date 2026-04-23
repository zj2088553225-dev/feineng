package models

import (
	"time"
)

// 自定义订单加纳
type CustomizeOrderGH struct {
	GHID           int        `gorm:"column:gh_id;primaryKey;autoIncrement:false;comment:自定义自增ID" json:"gh_id"`
	Week           string     `gorm:"column:week;size:10;index;comment:Week编号" json:"week"`
	Date           *time.Time `gorm:"column:date;type:date;index:idx_date_status,priority:3;comment:订单日期" json:"date"`
	OrderNumb      string     `gorm:"column:order_numb;size:50;comment:原始订单编号（唯一键）" json:"order_numb"`
	Qty            string     `gorm:"column:qty;size:10;comment:数量（字符串存储）" json:"qty"`       // 原 int
	Amount         string     `gorm:"column:amount;size:20;comment:金额（字符串存储）" json:"amount"` // 原 float64
	OrderShop      string     `gorm:"column:order_shop;size:255;comment:店铺URL或名称" json:"order_shop"`
	ProductName    string     `gorm:"column:product_name;size:500;comment:商品名称" json:"product_name"`
	FirstName      string     `gorm:"column:first_name;size:100;comment:收件人名" json:"first_name"`
	LastName       string     `gorm:"column:last_name;size:100;comment:收件人姓" json:"last_name"`
	PhoneNumber    string     `gorm:"column:phone_number;size:50;index;comment:电话号码" json:"phone_number"`
	EmailAddr      string     `gorm:"column:email_addr;size:255;comment:邮箱地址" json:"email_addr"`
	Address        string     `gorm:"column:address;type:text;comment:完整地址" json:"address"`
	City           string     `gorm:"column:city;size:100;index;comment:城市" json:"city"`
	JumiaSKU       string     `gorm:"column:jumia_sku;size:50;index;comment:Jumia平台SKU" json:"jumia_sku"`
	Agents         string     `gorm:"column:agents;size:100;index;comment:代理人员" json:"agents"`
	CALLED         string     `gorm:"column:called;size:100;index;comment:是否已致电（原始值存储）" json:"called"`         // 原 bool
	OrderDone      string     `gorm:"column:order_done;size:10;index;comment:订单是否完成（原始值存储）" json:"order_done"` // 原 bool
	CallComment    string     `gorm:"column:call_comment;type:text;comment:电话沟通备注" json:"call_comment"`
	ClosestPUS     string     `gorm:"column:closest_pus;size:100;comment:最近提货点" json:"closest_pus"`
	OrderNumber    string     `gorm:"column:order_number;size:50;comment:系统订单号" json:"order_number"`
	AgentComments  string     `gorm:"column:agent_comments;type:text;comment:代理备注" json:"agent_comments"`
	SellerComments string     `gorm:"column:seller_comments;type:text;comment:卖家备注" json:"seller_comments"`
	WAContactMade  string     `gorm:"column:wa_contact_made;size:10;index;comment:是否通过WhatsApp联系（原始值）" json:"wa_contact_made"` // 原 bool
	Person         string     `gorm:"column:person;size:100;comment:对接人" json:"person"`
	Status         string     `gorm:"column:status;size:50;index:idx_date_status,priority:2;comment:订单状态" json:"status"`
	TrackingURL    string     `gorm:"column:tracking_url;size:500;comment:物流追踪链接" json:"tracking_url"`
}

// 自定义订单肯尼亚
type CustomizeOrderKE struct {
	// KEID 作为业务主键和唯一索引，用于 upsert
	KEID                int        `gorm:"column:ke_id;primaryKey;autoIncrement:false" json:"ke_id"`
	First               string     `gorm:"column:first;size:50" json:"first"`
	CallDate            *time.Time `gorm:"column:call_date" json:"call_date"`
	OrderDate           *time.Time `gorm:"column:order_date" json:"order_date"`
	ID                  string     `gorm:"column:id" json:"id"`
	ItemName            string     `gorm:"column:item_name" json:"item_name"`
	Price               string     `gorm:"column:price" json:"price"` // 原始价格字符串（含逗号、货币符号）
	Qty                 string     `gorm:"column:qty" json:"qty"`     // 数量（字符串）
	CustomerName        string     `gorm:"column:customer_name" json:"customer_name"`
	PhoneNumber         string     `gorm:"column:phone_number" json:"phone_number"`
	PhoneNumber2        string     `gorm:"column:phone_number_2" json:"phone_number_2"`
	Address             string     `gorm:"column:address" json:"address"`
	City                string     `gorm:"column:city" json:"city"`
	Region              string     `gorm:"column:region" json:"region"`
	Email               string     `gorm:"column:email" json:"email"`
	JumiaSKU            string     `gorm:"column:jumia_sku" json:"jumia_sku"`
	PickUpStations      string     `gorm:"column:pick_up_stations" json:"pick_up_stations"`
	SellerAgent         string     `gorm:"column:seller_agent" json:"seller_agent"`
	Called              string     `gorm:"column:called" json:"called"`   // "Yes", "No"
	Reached             string     `gorm:"column:reached" json:"reached"` // "Yes", "No"
	OrderStatus         string     `gorm:"column:order_status" json:"order_status"`
	ShippingMethod      string     `gorm:"column:shipping_method" json:"shipping_method"`
	JumiaSalesAgentName string     `gorm:"column:jumia_sales_agent_name" json:"jumia_sales_agent_name"`
	OrderPlaced         string     `gorm:"column:order_placed" json:"order_placed"`
	OrderNumber         string     `gorm:"column:order_number" json:"order_number"`
	SellerComment       string     `gorm:"column:seller_comment" json:"seller_comment"`
	Ordered             string     `gorm:"column:ordered" json:"ordered"`
	Person              string     `gorm:"column:person" json:"person"`
	Status              string     `gorm:"column:status" json:"status"`
	TrackingURL         string     `gorm:"column:tracking_url" json:"tracking_url"`
}

// 自定义订单尼日利亚
type CustomizeOrderNG struct {
	NGID           int        `gorm:"column:ng_id;primaryKey;autoIncrement:false" json:"ng_id"`
	Week           string     `gorm:"column:week;size:10" json:"week"`
	Date           *time.Time `gorm:"column:date" json:"date"`
	ID             string     `gorm:"column:id;size:50" json:"id"` // 可能是订单 ID，非主键
	Time           *time.Time `gorm:"column:time" json:"time"`
	ItemName       string     `gorm:"column:item_name;size:500" json:"item_name"`
	Price          string     `gorm:"column:price;size:20" json:"price"` // 保留 string 防止科学计数法
	Qty            string     `gorm:"column:qty;size:10" json:"qty"`
	CustomerName   string     `gorm:"column:customer_name;size:100" json:"customer_name"`
	PhoneNumber    string     `gorm:"column:phone_number;size:50" json:"phone_number"`
	PhoneNumber2   string     `gorm:"column:phone_number_2;size:50" json:"phone_number_2"`
	Address        string     `gorm:"column:address;size:200" json:"address"`
	City           string     `gorm:"column:city;size:50" json:"city"`
	Region         string     `gorm:"column:region;size:50" json:"region"`
	Email          string     `gorm:"column:email;size:100" json:"email"`
	JumiaSKU       string     `gorm:"column:jumia_sku;size:50" json:"jumia_sku"`
	PusAddress     string     `gorm:"column:pus_address;size:200" json:"pus_address"`
	SellerAgent    string     `gorm:"column:seller_agent;size:50" json:"seller_agent"`
	Called         string     `gorm:"column:called;size:100" json:"called"`
	OrderStatus    string     `gorm:"column:order_status;size:50" json:"order_status"`
	Reached        string     `gorm:"column:reached;size:20" json:"reached"`
	ShippingMethod string     `gorm:"column:shipping_method;size:50" json:"shipping_method"`
	JumiaAgentName string     `gorm:"column:jumia_agent_name;size:50" json:"jumia_agent_name"`
	OrderPlaced    string     `gorm:"column:order_placed" json:"order_placed"`
	OrderNumber    string     `gorm:"column:order_number;size:50" json:"order_number"`
	SellerComment  string     `gorm:"column:seller_comment;size:200" json:"seller_comment"`
	Person         string     `gorm:"column:person;size:50" json:"person"`
	Status         string     `gorm:"column:status;size:50" json:"status"`
	TrackingURL    string     `gorm:"column:tracking_url;size:200" json:"tracking_url"`
}
