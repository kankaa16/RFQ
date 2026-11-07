package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type HistoricalQuote struct {
	BaseModel `bson:",inline"`

	OriginalQuoteNumber string `bson:"original_quote_number" json:"original_quote_number"`

	CustomerName string `bson:"customer_name" json:"customer_name"`
	PartNumber   string `bson:"part_number" json:"part_number"`
	PartName     string `bson:"part_name" json:"part_name"`

	MaterialGrade        string `bson:"material_grade" json:"material_grade"`
	ManufacturingProcess string `bson:"manufacturing_process" json:"manufacturing_process"`

	GrossWeight  float64 `bson:"gross_weight" json:"gross_weight"`
	AnnualVolume int64   `bson:"annual_volume" json:"annual_volume"`

	MaterialCost   float64 `bson:"material_cost" json:"material_cost"`
	ConversionCost float64 `bson:"conversion_cost" json:"conversion_cost"`
	TotalCost      float64 `bson:"total_cost" json:"total_cost"`
	SellingPrice   float64 `bson:"selling_price" json:"selling_price"`

	Currency string `bson:"currency" json:"currency"`

	QuotedAt time.Time `bson:"quoted_at" json:"quoted_at"`

	SourceFile string        `bson:"source_file" json:"source_file"`
	ImportedBy bson.ObjectID `bson:"imported_by" json:"imported_by"`
}
