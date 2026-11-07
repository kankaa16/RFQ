package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type RateMaster struct {
	BaseModel `bson:",inline"`

	RateCode string `bson:"rate_code" json:"rate_code"`
	RateName string `bson:"rate_name" json:"rate_name"`

	RateType string `bson:"rate_type" json:"rate_type"`
	Category string `bson:"category" json:"category"`

	MaterialGrade string `bson:"material_grade,omitempty" json:"material_grade,omitempty"`
	MachineType   string `bson:"machine_type,omitempty" json:"machine_type,omitempty"`

	Rate     float64 `bson:"rate" json:"rate"`
	Unit     string  `bson:"unit" json:"unit"`
	Currency string  `bson:"currency" json:"currency"`

	Version       int       `bson:"version" json:"version"`
	EffectiveFrom time.Time `bson:"effective_from" json:"effective_from"`
	EffectiveTo   time.Time `bson:"effective_to,omitempty" json:"effective_to,omitempty"`

	IsActive bool `bson:"is_active" json:"is_active"`

	UpdatedBy bson.ObjectID `bson:"updated_by" json:"updated_by"`
}

const (
	RateTypeMaterial = "material"
	RateTypeMachine  = "machine"
	RateTypeLabour   = "labour"
	RateTypeOverhead = "overhead"
	RateTypePacking  = "packing"
	RateTypeFreight  = "freight"
)
