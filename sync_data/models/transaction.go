package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"
)

// 从csv中读取数据存储到数据库
type Transaction struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	TransactionDate    time.Time  `json:"transaction_date"`
	TransactionType    string     `gorm:"size:100" json:"transaction_type"`
	TransactionNumber  string     `gorm:"column:transaction_number;size:255;uniqueIndex;not null" json:"transaction_number"`
	TransactionState   string     `gorm:"size:100" json:"transaction_state"`
	Details            string     `gorm:"type:text" json:"details"`
	SellerSKU          string     `gorm:"column:seller_sku;size:255" json:"seller_sku"`
	JumiaSKU           string     `gorm:"column:jumia_sku;size:255" json:"jumia_sku"`
	Amount             float64    `json:"amount"`
	StatementStartDate *time.Time `json:"statement_start_date"`
	StatementEndDate   *time.Time `json:"statement_end_date"`
	PaidStatus         bool       `json:"paid_status"`
	OrderNo            NullString `gorm:"column:order_no;size:255" json:"order_no"`
	OrderItemNo        NullString `gorm:"column:order_item_no;size:255" json:"order_item_no"`
	OrderItemStatus    NullString `gorm:"column:order_item_status;size:100" json:"order_item_status"`
	ShippingProvider   NullString `gorm:"column:shipping_provider;size:100" json:"shipping_provider"`
	TrackingNumber     NullString `gorm:"column:tracking_number;size:255" json:"tracking_number"`
	Comment            NullString `gorm:"column:comment;type:text" json:"comment"`
	LocalExchangeRate  float64    `json:"local_exchange_rate"`
	CountryCode        string     `gorm:"size:10" json:"country_code"`
	StatementNumber    string     `gorm:"size:255" json:"statement_number"`
}

type NullString struct {
	sql.NullString
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

func (ns *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ns.String, ns.Valid = "", false
		return nil
	}
	err := json.Unmarshal(data, &ns.String)
	ns.Valid = err == nil
	return err
}

func (ns *NullString) Scan(value interface{}) error {
	return (&ns.NullString).Scan(value)
}

func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

// StringToNullString converts string to NullString
func StringToNullString(s string) NullString {
	trimmed := strings.TrimSpace(s)
	return NullString{
		NullString: sql.NullString{
			String: trimmed,
			Valid:  s != "", // 如果原字符串为空，视为 NULL
		},
	}
}
