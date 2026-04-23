package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// ------------------------- 数据结构 -------------------------

// 订单表
type CesFbjOrder struct {
	ID                  string     `gorm:"type:char(255);primaryKey" json:"id"` // 全局唯一业务ID
	CreateBy            string     `gorm:"type:varchar(255)" json:"createBy"`
	CreateTime          CustomTime `gorm:"type:datetime" json:"createTime"`
	UpdateBy            string     `gorm:"type:varchar(255)" json:"updateBy"`
	UpdateTime          CustomTime `gorm:"type:datetime" json:"updateTime"`
	HBL                 string     `gorm:"type:varchar(255)" json:"hbl"`
	Type                string     `gorm:"type:varchar(255)" json:"type"`
	TypeName            string     `gorm:"type:varchar(255)" json:"typeName"`
	ChannelID           string     `gorm:"type:varchar(255)" json:"channelId"`
	ChannelCode         string     `gorm:"type:varchar(255)" json:"channelCode"`
	ChannelName         string     `gorm:"type:varchar(255)" json:"channelName"`
	WhID                string     `gorm:"type:varchar(255)" json:"whId"`
	WhCode              string     `gorm:"type:varchar(255)" json:"whCode"`
	WhName              string     `gorm:"type:varchar(255)" json:"whName"`
	WhAddress           string     `gorm:"type:varchar(500)" json:"whAddress"`
	PickingType         string     `gorm:"type:varchar(255)" json:"pickingType"`
	ChoiceWhID          string     `gorm:"type:varchar(255)" json:"choiceWhId"`
	ChoiceWhContact     string     `gorm:"type:varchar(255)" json:"choiceWhContact"`
	ChoiceWhPhone       string     `gorm:"type:varchar(255)" json:"choiceWhPhone"`
	ChoiceWhAddress     string     `gorm:"type:varchar(500)" json:"choiceWhAddress"`
	DeclareService      string     `gorm:"type:varchar(255)" json:"declareService"`
	ServiceType         string     `gorm:"type:varchar(255)" json:"serviceType"`
	CustEmail           string     `gorm:"type:varchar(255)" json:"custEmail"`
	ExpeStorageTime     string     `gorm:"type:varchar(255)" json:"expeStorageTime"`
	RealStorageTime     string     `gorm:"type:varchar(255)" json:"realStorageTime"`
	DispatchDocuments   string     `gorm:"type:varchar(500)" json:"dispatchDocuments"`
	OrderTracking       string     `gorm:"type:text" json:"orderTracking"`
	ReceiptID           string     `gorm:"type:varchar(255)" json:"receiptId"`
	Status              string     `gorm:"type:varchar(255)" json:"status"`
	StatusName          string     `gorm:"type:varchar(255)" json:"statusName"`
	TotalCount          int        `gorm:"type:int" json:"totalCount"`
	TotalNetWeight      float64    `gorm:"type:decimal(10,3)" json:"totalNetWeight"`
	TotalRoughWeight    float64    `gorm:"type:decimal(10,3)" json:"totalRoughWeight"`
	TotalCBM            float64    `gorm:"type:decimal(10,3)" json:"totalCbm"`
	TotalCommodityCount int        `gorm:"type:int" json:"totalCommodityCount"`

	//Storage entry
	//入仓重量和入仓体积
	TotalRoughWeightStorage float64 `gorm:"type:decimal(10,3)" json:"totalRoughWeightStorage"`
	TotalCBMStorage         float64 `gorm:"type:decimal(10,3)" json:"totalCBMStorage"`

	Trajectories []*CesFbjOrderTrajectory `gorm:"foreignKey:OrderID" json:"trajectories"`
	Cargos       []*CesFbjCargoInfo       `gorm:"foreignKey:OrderID" json:"cargos"`
}

// 货物信息表
type CesFbjCargoInfo struct {
	ID         string     `gorm:"type:char(255);primaryKey" json:"id"`
	CreateBy   string     `gorm:"type:varchar(255)" json:"createBy"`
	CreateTime CustomTime `gorm:"type:datetime" json:"createTime"`
	ShopID     string     `gorm:"type:varchar(255)" json:"shopId"`
	SellerID   string     `gorm:"type:varchar(255)" json:"sellerId"`
	ShopName   string     `gorm:"type:varchar(255)" json:"shopName"`
	PO         string     `gorm:"type:varchar(255)" json:"po"`
	Amends     string     `gorm:"type:varchar(255)" json:"amends"`
	OrderID    string     `gorm:"type:char(255);index" json:"orderId"` // 关联订单表

	Order    *CesFbjOrder     `gorm:"foreignKey:OrderID" json:"order"`
	Packages []*CesFbjPackage `gorm:"foreignKey:CargoID" json:"packages"`
}

// 包裹信息表
type CesFbjPackage struct {
	ID                  string     `gorm:"type:char(255);primaryKey" json:"id"`
	CreateBy            string     `gorm:"type:varchar(255)" json:"createBy"`
	CreateTime          CustomTime `gorm:"type:datetime" json:"createTime"`
	BoxNo               string     `gorm:"type:varchar(255)" json:"boxNo"`
	Length              float64    `gorm:"type:decimal(10,3)" json:"length"`
	High                float64    `gorm:"type:decimal(10,3)" json:"high"`
	Width               float64    `gorm:"type:decimal(10,3)" json:"width"`
	RoughWeight         float64    `gorm:"type:decimal(10,3)" json:"roughWeight"`
	TotalRoughWeight    float64    `gorm:"type:decimal(10,3)" json:"totalRoughWeight"`
	NetWeight           float64    `gorm:"type:decimal(10,3)" json:"netWeight"`
	TotalNetWeight      float64    `gorm:"type:decimal(10,3)" json:"totalNetWeight"`
	Count               int        `gorm:"type:int" json:"count"`
	TotalCommodityCount int        `gorm:"type:int" json:"totalCommodityCount"`
	CargoID             string     `gorm:"type:char(255);index" json:"cargoId"` // 关联货物表
	Value               string     `gorm:"type:varchar(255)" json:"value"`

	Cargo       *CesFbjCargoInfo       `gorm:"foreignKey:CargoID" json:"cargo"`
	Commodities []*CesFbjCommodityInfo `gorm:"foreignKey:PackageID" json:"commodities"`
}

// 商品信息表
type CesFbjCommodityInfo struct {
	ID                     string `gorm:"type:char(255);primaryKey" json:"id"`
	CommodityID            string `gorm:"type:varchar(255)" json:"commodityId"`
	CommodityName          string `gorm:"type:varchar(255)" json:"commodityName"`
	CommoditySku           string `gorm:"type:varchar(255)" json:"commoditySku"`
	ShopSku                string `gorm:"type:varchar(255)" json:"shopSku"`
	CommodityCname         string `gorm:"type:varchar(255)" json:"commodityCname"`
	CommodityEname         string `gorm:"type:varchar(255)" json:"commodityEname"`
	CommodityAttribute     string `gorm:"type:varchar(255)" json:"commodityAttribute"`
	CommodityAttributeName string `gorm:"type:varchar(255)" json:"commodityAttributeName"`
	CommodityCount         int    `gorm:"type:int" json:"commodityCount"`
	PackageID              string `gorm:"type:char(255);index" json:"packageId"` // 关联包裹表
	BrandName              string `gorm:"type:varchar(255)" json:"brandName"`
	IsBrand                int    `gorm:"type:tinyint" json:"isBrand"`

	HBL        string         `gorm:"type:varchar(255)" json:"hbl"`
	CreateBy   string         `gorm:"type:varchar(255)" json:"createBy"` // 创建人
	CreateTime CustomTime     `gorm:"type:datetime" json:"createTime"`   // 创建时间
	UpdateBy   string         `gorm:"type:varchar(255)" json:"updateBy"` // 更新人
	UpdateTime CustomTime     `gorm:"type:datetime" json:"updateTime"`   // 更新时间
	Package    *CesFbjPackage `gorm:"foreignKey:PackageID" json:"package"`
}

// 订单轨迹表
type CesFbjOrderTrajectory struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID   string     `gorm:"index;comment:对应CesFbjOrder ID" json:"order_id"`
	SO        string     `gorm:"index;comment:物流单号" json:"so"`
	OpLink    string     `gorm:"size:255;comment:操作节点" json:"op_link"`
	Timestamp *time.Time `gorm:"comment:轨迹时间" json:"timestamp"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ------------------------- CustomTime -------------------------

type CustomTime struct {
	time.Time
}

var timeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02",
	time.RFC3339,
}

// JSON 解析
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	str := strings.TrimSpace(strings.Trim(string(b), `"`))
	if str == "" || str == "null" {
		ct.Time = time.Time{}
		return nil
	}

	var parseErr error
	for _, layout := range timeLayouts {
		t, err := time.Parse(layout, str)
		if err == nil {
			ct.Time = t
			return nil
		}
		parseErr = err
	}
	return fmt.Errorf("解析时间失败: %w", parseErr)
}

// JSON 序列化
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.IsZero() {
		return []byte(`null`), nil
	}
	return []byte(`"` + ct.Format("2006-01-02 15:04:05") + `"`), nil
}

// 实现 GORM driver.Valuer 接口（写入数据库）
func (ct CustomTime) Value() (driver.Value, error) {
	if ct.IsZero() {
		return nil, nil
	}
	return ct.Format("2006-01-02 15:04:05"), nil
}

// 实现 sql.Scanner 接口（从数据库读取）
func (ct *CustomTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		return nil
	case []byte:
		return ct.UnmarshalJSON(v)
	case string:
		return ct.UnmarshalJSON([]byte(v))
	default:
		return fmt.Errorf("cannot scan type %T into CustomTime", value)
	}
}
