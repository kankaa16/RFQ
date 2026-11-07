package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Routing struct {
	BaseModel `bson:",inline"`

	PartID bson.ObjectID `bson:"part_id" json:"part_id"`

	Operations []RoutingOperation `bson:"operations" json:"operations"`

	Source string `bson:"source" json:"source"`

	IsVerified bool          `bson:"is_verified" json:"is_verified"`
	VerifiedBy bson.ObjectID `bson:"verified_by,omitempty" json:"verified_by,omitempty"`
}

type RoutingOperation struct {
	Sequence int    `bson:"sequence" json:"sequence"`
	Name     string `bson:"name" json:"name"`

	MachineType string `bson:"machine_type" json:"machine_type"`

	CycleTimeMinutes float64 `bson:"cycle_time_minutes" json:"cycle_time_minutes"`
	SetupTimeMinutes float64 `bson:"setup_time_minutes" json:"setup_time_minutes"`

	HourRate float64 `bson:"hour_rate" json:"hour_rate"`

	RateMasterID bson.ObjectID `bson:"rate_master_id,omitempty" json:"rate_master_id,omitempty"`
}

const (
	RoutingSourceManual     = "manual"
	RoutingSourceHistorical = "historical"
	RoutingSourceAI         = "ai"
)
