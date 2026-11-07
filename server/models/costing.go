package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Costing struct {
	BaseModel `bson:",inline"`

	PartID bson.ObjectID `bson:"part_id" json:"part_id"`
	RFQID  bson.ObjectID `bson:"rfq_id" json:"rfq_id"`

	MaterialCost  float64 `bson:"material_cost" json:"material_cost"`
	MachiningCost float64 `bson:"machining_cost" json:"machining_cost"`
	ScrapCost     float64 `bson:"scrap_cost" json:"scrap_cost"`
	SetupCost     float64 `bson:"setup_cost" json:"setup_cost"`

	OverheadCost float64 `bson:"overhead_cost" json:"overhead_cost"`
	PackingCost  float64 `bson:"packing_cost" json:"packing_cost"`
	FreightCost  float64 `bson:"freight_cost" json:"freight_cost"`

	TotalCost    float64 `bson:"total_cost" json:"total_cost"`
	MarginRate   float64 `bson:"margin_rate" json:"margin_rate"`
	SellingPrice float64 `bson:"selling_price" json:"selling_price"`

	Scenarios []VolumeScenario `bson:"scenarios" json:"scenarios"`

	RateSnapshots []RateSnapshot `bson:"rate_snapshots" json:"rate_snapshots"`

	FormulaVersion string `bson:"formula_version" json:"formula_version"`
	Assumptions    string `bson:"assumptions" json:"assumptions"`

	PreparedBy   bson.ObjectID `bson:"prepared_by" json:"prepared_by"`
	CalculatedAt time.Time     `bson:"calculated_at" json:"calculated_at"`
}

type VolumeScenario struct {
	AnnualVolume int64 `bson:"annual_volume" json:"annual_volume"`

	MaterialCost  float64 `bson:"material_cost" json:"material_cost"`
	MachiningCost float64 `bson:"machining_cost" json:"machining_cost"`
	OverheadCost  float64 `bson:"overhead_cost" json:"overhead_cost"`
	OtherCost     float64 `bson:"other_cost" json:"other_cost"`

	TotalCost    float64 `bson:"total_cost" json:"total_cost"`
	MarginRate   float64 `bson:"margin_rate" json:"margin_rate"`
	SellingPrice float64 `bson:"selling_price" json:"selling_price"`
}

type RateSnapshot struct {
	RateID   bson.ObjectID `bson:"rate_id" json:"rate_id"`
	RateCode string        `bson:"rate_code" json:"rate_code"`
	Version  int           `bson:"version" json:"version"`
	Rate     float64       `bson:"rate" json:"rate"`
	Unit     string        `bson:"unit" json:"unit"`
}
