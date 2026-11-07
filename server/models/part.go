package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Part struct {
	BaseModel `bson:",inline"`

	RFQID      bson.ObjectID `bson:"rfq_id" json:"rfq_id"`
	PartNumber string        `bson:"part_number" json:"part_number"`
	PartName   string        `bson:"part_name" json:"part_name"`

	DrawingNumber   string `bson:"drawing_number" json:"drawing_number"`
	DrawingRevision string `bson:"drawing_revision" json:"drawing_revision"`

	MaterialGrade        string `bson:"material_grade" json:"material_grade"`
	ManufacturingProcess string `bson:"manufacturing_process" json:"manufacturing_process"`

	GrossWeight float64 `bson:"gross_weight" json:"gross_weight"`
	NetWeight   float64 `bson:"net_weight" json:"net_weight"`
	ScrapWeight float64 `bson:"scrap_weight" json:"scrap_weight"`

	AnnualVolume  int64 `bson:"annual_volume" json:"annual_volume"`
	MonthlyVolume int64 `bson:"monthly_volume" json:"monthly_volume"`

	CustomerTargetPrice float64 `bson:"customer_target_price,omitempty" json:"customer_target_price,omitempty"`

	MaterialConfirmed bool `bson:"material_confirmed" json:"material_confirmed"`
	DrawingVerified   bool `bson:"drawing_verified" json:"drawing_verified"`
}
