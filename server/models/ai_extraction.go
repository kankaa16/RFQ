package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type AIExtraction struct {
	BaseModel `bson:",inline"`

	RFQID      bson.ObjectID `bson:"rfq_id" json:"rfq_id"`
	DocumentID bson.ObjectID `bson:"document_id" json:"document_id"`

	ExtractedFields []ExtractedField `bson:"extracted_fields" json:"extracted_fields"`

	SuggestedRouting []string `bson:"suggested_routing" json:"suggested_routing"`

	SourceType string `bson:"source_type" json:"source_type"`
	ModelName  string `bson:"model_name" json:"model_name"`

	Status string `bson:"status" json:"status"`

	ReviewedBy bson.ObjectID `bson:"reviewed_by,omitempty" json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time    `bson:"reviewed_at,omitempty" json:"reviewed_at,omitempty"`
}

type ExtractedField struct {
	FieldName  string `bson:"field_name" json:"field_name"`
	Value      string `bson:"value" json:"value"`
	SourceText string `bson:"source_text" json:"source_text"`

	Confidence float64 `bson:"confidence" json:"confidence"`

	IsAccepted bool `bson:"is_accepted" json:"is_accepted"`
}

const (
	AIStatusPending  = "pending"
	AIStatusReviewed = "reviewed"
	AIStatusFailed   = "failed"

	AISourceEmail   = "email"
	AISourceExcel   = "excel"
	AISourceDrawing = "drawing"
)
