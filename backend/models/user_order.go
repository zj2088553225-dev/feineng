package models

import "time"

type Order struct {
	ID                       string    `gorm:"primaryKey;size:36" json:"id"`
	ShopIDs                  string    `json:"shopIds"`
	TotalItems               int       `json:"totalItems"`
	PackedItems              int       `json:"packedItems"`
	IsPrepayment             bool      `json:"isPrepayment"`
	HasMultipleStatus        bool      `json:"hasMultipleStatus"`
	HasItemsFulfilledByJumia bool      `json:"hasItemsFulfilledByJumia"`
	PendingSince             *string   `json:"pendingSince"`
	Status                   string    `json:"status"`
	DeliveryOption           *string   `json:"deliveryOption"`
	Number                   string    `json:"number"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`

	// 嵌套字段展开
	TotalAmountCurrency      string  `gorm:"column:total_amount_currency" json:"totalAmountCurrency"`
	TotalAmountValue         float64 `gorm:"column:total_amount_value" json:"totalAmountValue"`
	TotalAmountLocalCurrency string  `gorm:"column:total_amount_local_currency" json:"totalAmountLocalCurrency"`
	TotalAmountLocalValue    float64 `gorm:"column:total_amount_local_value" json:"totalAmountLocalValue"`
	TotalShippingCost        float64 `gorm:"column:total_shipping_cost" json:"totalShippingCost"`
	NetProfit                float64 `gorm:"column:net_profit" json:"netProfit"`

	CountryCode     string `gorm:"column:country_code"`
	CountryName     string `gorm:"column:country_name"`
	CountryCurrency string `gorm:"column:country_currency"`

	// 配送地址
	ShippingFirstName   string `gorm:"column:shipping_first_name"`
	ShippingLastName    string `gorm:"column:shipping_last_name"`
	ShippingAddress     string `gorm:"column:shipping_address"`
	ShippingCity        string `gorm:"column:shipping_city"`
	ShippingPostalCode  string `gorm:"column:shipping_postal_code"`
	ShippingWard        string `gorm:"column:shipping_ward"`
	ShippingRegion      string `gorm:"column:shipping_region"`
	ShippingCountryName string `gorm:"column:shipping_country_name"`
	// ✅ 新增：订单项列表（只用于返回，GORM 忽略）
	OrderItems []OrderItem `json:"orderItems" gorm:"-"`
}

type OrderItem struct {
	ID                 string `gorm:"column:id;primaryKey" json:"id"`
	OrderID            string `gorm:"column:order_id;index" json:"orderId"`
	OrderNumber        string `gorm:"column:order_number" json:"orderNumber"`
	Status             string `gorm:"column:status" json:"status"`
	TrackingNumber     string `gorm:"column:tracking_number;index" json:"trackingNumber"`
	TrackingURL        string `gorm:"column:tracking_url" json:"trackingUrl"`
	ShipmentType       string `gorm:"column:shipment_type" json:"shipmentType"`
	DeliveryOption     string `gorm:"column:delivery_option" json:"deliveryOption"`
	IsFulfilledByJumia bool   `gorm:"column:is_fulfilled_by_jumia" json:"isFulfilledByJumia"`
	ShopID             string `gorm:"column:shop_id" json:"shopId"`

	// 价格字段
	ItemPrice           float64 `gorm:"column:item_price" json:"itemPrice"`
	PaidPrice           float64 `gorm:"column:paid_price" json:"paidPrice"`
	ShippingAmount      float64 `gorm:"column:shipping_amount" json:"shippingAmount"`
	ItemPriceLocal      float64 `gorm:"column:item_price_local" json:"itemPriceLocal"`
	PaidPriceLocal      float64 `gorm:"column:paid_price_local" json:"paidPriceLocal"`
	ShippingAmountLocal float64 `gorm:"column:shipping_amount_local" json:"shippingAmountLocal"`
	ExchangeRate        float64 `gorm:"column:exchange_rate" json:"exchangeRate"`
	TaxAmount           float64 `gorm:"column:tax_amount" json:"taxAmount"`
	VoucherAmount       float64 `gorm:"column:voucher_amount" json:"voucherAmount"`

	// 国家
	CountryCode     string `gorm:"column:country_code" json:"countryCode"`
	CountryName     string `gorm:"column:country_name" json:"countryName"`
	CountryCurrency string `gorm:"column:country_currency" json:"countryCurrency"`

	// 商品信息
	ProductName string `gorm:"column:product_name" json:"productName"`
	SellerSKU   string `gorm:"column:seller_sku;index" json:"sellerSku"`
	JumiaSKU    string `gorm:"column:jumia_sku;index" json:"jumia_sku"` // 新增字段
	ImageURL    string `gorm:"column:image_url" json:"imageUrl"`

	// 收货地址
	ShippingFirstName   string `gorm:"column:shipping_first_name" json:"shippingFirstName"`
	ShippingLastName    string `gorm:"column:shipping_last_name" json:"shippingLastName"`
	ShippingAddress     string `gorm:"column:shipping_address" json:"shippingAddress"`
	ShippingCity        string `gorm:"column:shipping_city" json:"shippingCity"`
	ShippingPostalCode  string `gorm:"column:shipping_postal_code" json:"shippingPostalCode"`
	ShippingWard        string `gorm:"column:shipping_ward" json:"shippingWard"`
	ShippingRegion      string `gorm:"column:shipping_region" json:"shippingRegion"`
	ShippingCountryName string `gorm:"column:shipping_country_name" json:"shippingCountryName"`

	CreatedAt time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt"`
}
