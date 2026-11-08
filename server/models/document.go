package models

import "go.mongodb.org/mongo-driver/v2/bson"

type Document struct {
	BaseModel `bson:",inline"`

	RFQID  bson.ObjectID `bson:"rfq_id,omitempty" json:"rfq_id,omitempty"`
	PartID bson.ObjectID `bson:"part_id,omitempty" json:"part_id,omitempty"`

	FileName   string `bson:"file_name" json:"file_name"`
	StoredName string `bson:"stored_name" json:"stored_name"`
	FileType   string `bson:"file_type" json:"file_type"`
	FileSize   int64  `bson:"file_size" json:"file_size"`

	StoragePath string `bson:"storage_path" json:"storage_path"`
	FileHash    string `bson:"file_hash" json:"file_hash"`

	Category string `bson:"category" json:"category"`

	UploadedBy bson.ObjectID `bson:"uploaded_by" json:"uploaded_by"`

	ExtractionStatus string `bson:"extraction_status" json:"extraction_status"`
	IsVerified       bool   `bson:"is_verified" json:"is_verified"`
}

const (
	DocumentTypeDrawing   = "drawing"
	DocumentTypePartList  = "part_list"
	DocumentTypeQuotation = "quotation"
	DocumentTypeOther     = "other"

	ExtractionPending   = "pending"
	ExtractionCompleted = "completed"
	ExtractionFailed    = "failed"
)
