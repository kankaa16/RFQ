package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Quote struct {
	BaseModel `bson:",inline"`

	QuoteNumber string        `bson:"quote_number" json:"quote_number"`
	RFQID       bson.ObjectID `bson:"rfq_id" json:"rfq_id"`
	CustomerID  bson.ObjectID `bson:"customer_id" json:"customer_id"`

	CostingIDs []bson.ObjectID `bson:"costing_ids" json:"costing_ids"`

	Subtotal   float64 `bson:"subtotal" json:"subtotal"`
	GSTPercent float64 `bson:"gst_percent" json:"gst_percent"`
	GSTAmount  float64 `bson:"gst_amount" json:"gst_amount"`
	GrandTotal float64 `bson:"grand_total" json:"grand_total"`

	AnnualValue float64 `bson:"annual_value" json:"annual_value"`
	Currency    string  `bson:"currency" json:"currency"`

	PaymentTerms     string `bson:"payment_terms" json:"payment_terms"`
	DeliveryTerms    string `bson:"delivery_terms" json:"delivery_terms"`
	ValidityDays     int    `bson:"validity_days" json:"validity_days"`
	DeliveryLocation string `bson:"delivery_location" json:"delivery_location"`

	Status string `bson:"status" json:"status"`

	PreparedBy bson.ObjectID `bson:"prepared_by" json:"prepared_by"`
	ApprovedBy bson.ObjectID `bson:"approved_by,omitempty" json:"approved_by,omitempty"`
	ApprovedAt *time.Time    `bson:"approved_at,omitempty" json:"approved_at,omitempty"`

	SentAt  *time.Time `bson:"sent_at,omitempty" json:"sent_at,omitempty"`
	PDFPath string     `bson:"pdf_path,omitempty" json:"pdf_path,omitempty"`
}

const (
	QuoteStatusDraft    = "draft"
	QuoteStatusPending  = "pending_approval"
	QuoteStatusApproved = "approved"
	QuoteStatusRejected = "rejected"
	QuoteStatusSent     = "sent"
)
