package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// shared fields for all database models
type BaseModel struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}

type RFQ struct {
	BaseModel `bson:",inline"`

	RFQNumber string `bson:"rfq_number" json:"rfq_number"`

	CustomerID   bson.ObjectID `bson:"customer_id" json:"customer_id"`
	CustomerName string        `bson:"customer_name" json:"customer_name"`

	OwnerID bson.ObjectID `bson:"owner_id" json:"owner_id"`

	PartCount int `bson:"part_count" json:"part_count"`

	SourceChannel string    `bson:"source_channel" json:"source_channel"`
	Priority      string    `bson:"priority" json:"priority"`
	ReceivedAt    time.Time `bson:"received_at" json:"received_at"`
	DueDate       time.Time `bson:"due_date" json:"due_date"`
	TargetSOPDate time.Time `bson:"target_sop_date,omitempty" json:"target_sop_date,omitempty"`

	ExpectedAnnualValue float64 `bson:"expected_annual_value" json:"expected_annual_value"`
	Currency            string  `bson:"currency" json:"currency"`

	Status     string `bson:"status" json:"status"`
	LostReason string `bson:"lost_reason,omitempty" json:"lost_reason,omitempty"`

	// latest quotation linked to this rfq
	LatestQuoteID          bson.ObjectID `bson:"latest_quote_id,omitempty" json:"latest_quote_id,omitempty"`
	LatestQuoteNumber      string        `bson:"latest_quote_number,omitempty" json:"latest_quote_number,omitempty"`
	LatestQuoteStatus      string        `bson:"latest_quote_status,omitempty" json:"latest_quote_status,omitempty"`
	LatestQuoteGrandTotal  float64       `bson:"latest_quote_grand_total,omitempty" json:"latest_quote_grand_total,omitempty"`
	LatestQuoteAnnualValue float64       `bson:"latest_quote_annual_value,omitempty" json:"latest_quote_annual_value,omitempty"`
	LatestQuotedAt         *time.Time    `bson:"latest_quoted_at,omitempty" json:"latest_quoted_at,omitempty"`

	InternalNotes string `bson:"internal_notes,omitempty" json:"internal_notes,omitempty"`
}

const (
	RFQStatusEstimating = "estimating"
	RFQStatusSent       = "sent"
	RFQStatusWon        = "won"
	RFQStatusLost       = "lost"

	RFQPriorityLow    = "low"
	RFQPriorityMedium = "medium"
	RFQPriorityHigh   = "high"

	RFQSourceEmail  = "email"
	RFQSourceManual = "manual"
	RFQSourceImport = "import"
)
