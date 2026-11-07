package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Approval struct {
	BaseModel `bson:",inline"`

	QuoteID bson.ObjectID `bson:"quote_id" json:"quote_id"`

	RequestedBy bson.ObjectID `bson:"requested_by" json:"requested_by"`
	ReviewedBy  bson.ObjectID `bson:"reviewed_by,omitempty" json:"reviewed_by,omitempty"`

	Action string `bson:"action" json:"action"`

	Comments        string `bson:"comments,omitempty" json:"comments,omitempty"`
	RejectionReason string `bson:"rejection_reason,omitempty" json:"rejection_reason,omitempty"`

	RequestedAt time.Time  `bson:"requested_at" json:"requested_at"`
	ReviewedAt  *time.Time `bson:"reviewed_at,omitempty" json:"reviewed_at,omitempty"`
}

const (
	ApprovalActionRequested = "requested"
	ApprovalActionApproved  = "approved"
	ApprovalActionRejected  = "rejected"
)
