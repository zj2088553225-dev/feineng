package models

import "time"

type CostSheet struct {
	ID                     string     `gorm:"column:id;primaryKey;size:128" json:"id"`
	TransactionOrderNumber string     `gorm:"column:transaction_order_number;size:100;index" json:"transactionOrderNumber"`
	TrackingNumber         string     `gorm:"column:tracking_number;size:100;index" json:"trackingNumber"`
	CostType               string     `gorm:"column:cost_type;size:100;index" json:"costType"`
	ChargeWeight           float64    `gorm:"column:charge_weight" json:"chargeWeight"`
	Currency               string     `gorm:"column:currency;size:20" json:"currency"`
	Amount                 float64    `gorm:"column:amount" json:"amount"`
	DeductionStatus        string     `gorm:"column:deduction_status;size:50;index" json:"deductionStatus"`
	RawStatus              string     `gorm:"column:raw_status;size:100" json:"rawStatus"`
	OccurredAt             *time.Time `gorm:"column:occurred_at" json:"occurredAt"`
	CreatedAt              time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt              time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}
