package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type GeminiExtractionService struct {
	DocumentCollection   *mongo.Collection
	ExtractionCollection *mongo.Collection
	Gemini               *GeminiService
}

func (s *GeminiExtractionService) ExtractDocument(c *gin.Context) {
	if s.Gemini == nil ||
		s.DocumentCollection == nil ||
		s.ExtractionCollection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "extraction service is not configured",
		})
		return
	}

	documentID, err := bson.ObjectIDFromHex(c.Param("document_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid document id",
		})
		return
	}

	userID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	var document models.Document

	err = s.DocumentCollection.FindOne(ctx, bson.M{
		"_id":         documentID,
		"uploaded_by": userID,
	}).Decode(&document)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "document not found or access denied",
			})
			return
		}

		log.Printf("document lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch document",
		})
		return
	}

	if strings.ToLower(filepath.Ext(document.FileName)) != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "gemini extraction currently supports pdf files only",
		})
		return
	}

	// Return an existing extraction instead of creating a duplicate.
	var existing models.AIExtraction

	err = s.ExtractionCollection.FindOne(ctx, bson.M{
		"document_id": documentID,
	}).Decode(&existing)

	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "existing extraction returned",
			"data": gin.H{
				"extraction_id": existing.ID.Hex(),
				"document_id":   documentID.Hex(),
				"field_count":   len(existing.ExtractedFields),
				"extraction":    existing,
			},
		})
		return
	}

	if err != mongo.ErrNoDocuments {
		log.Printf("existing extraction lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not check existing extraction",
		})
		return
	}

	pdfBytes, err := os.ReadFile(document.StoragePath)
	if err != nil {
		log.Printf("pdf read failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read stored pdf",
		})
		return
	}

	if len(pdfBytes) == 0 || len(pdfBytes) > 20*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "pdf is empty or exceeds the 20 mb limit",
		})
		return
	}

	extraction, err := s.Gemini.ExtractPDF(
		ctx,
		document.FileName,
		pdfBytes,
	)

	if err != nil {
		log.Printf("gemini extraction error: %v", err)

		_, updateErr := s.DocumentCollection.UpdateOne(ctx,
			bson.M{"_id": documentID},
			bson.M{"$set": bson.M{
				"extraction_status": models.ExtractionFailed,
				"updated_at":        time.Now(),
			}},
		)

		if updateErr != nil {
			log.Printf("document status update failed: %v", updateErr)
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "gemini extraction failed",
		})
		return
	}

	fields := make([]models.ExtractedField, 0)

	addField := func(name, value string, confidence float64) {
		fields = append(fields, models.ExtractedField{
			FieldName:      name,
			Value:          value,
			OriginalValue:  value,
			ReviewedValue:  "",
			SourceText:     "",
			SourceLocation: "",
			Confidence:     confidence,
			ReviewStatus:   models.AIFieldPending,
			IsAccepted:     false,
		})
	}

	addField("customer_name", extraction.CustomerName, 0)
	addField("rfq_number", extraction.RFQNumber, 0)
	addField("due_date", extraction.DueDate, 0)
	addField("currency", extraction.Currency, 0)

	for i, part := range extraction.Parts {
		prefix := fmt.Sprintf("parts[%d].", i)

		addField(prefix+"part_number", part.PartNumber, part.Confidence)
		addField(prefix+"part_name", part.PartName, part.Confidence)
		addField(prefix+"drawing_number", part.DrawingNumber, part.Confidence)
		addField(prefix+"drawing_revision", part.DrawingRevision, part.Confidence)
		addField(prefix+"material_grade", part.MaterialGrade, part.Confidence)
		addField(prefix+"manufacturing_process", part.ManufacturingProcess, part.Confidence)
		addField(prefix+"gross_weight", fmt.Sprintf("%g", part.GrossWeight), part.Confidence)
		addField(prefix+"net_weight", fmt.Sprintf("%g", part.NetWeight), part.Confidence)
		addField(prefix+"annual_volume", fmt.Sprintf("%d", part.AnnualVolume), part.Confidence)
		addField(prefix+"target_price", fmt.Sprintf("%g", part.TargetPrice), part.Confidence)
	}

	now := time.Now()

	record := models.AIExtraction{
		RFQID:            document.RFQID,
		DocumentID:       documentID,
		ExtractedFields:  fields,
		SuggestedRouting: []string{},
		SourceType:       models.AISourcePDF,
		ModelName:        s.Gemini.Model,
		Status:           models.AIStatusPending,
	}

	record.CreatedAt = now
	record.UpdatedAt = now

	result, err := s.ExtractionCollection.InsertOne(ctx, record)
	if err != nil {
		log.Printf("extraction insert failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save extraction",
		})
		return
	}

	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read saved extraction id",
		})
		return
	}

	// Keep the returned extraction object consistent with its MongoDB record.
	record.ID = insertedID

	_, err = s.DocumentCollection.UpdateOne(ctx,
		bson.M{"_id": documentID},
		bson.M{"$set": bson.M{
			"extraction_status": models.ExtractionCompleted,
			"updated_at":        now,
		}},
	)

	if err != nil {
		log.Printf("document status update failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "extraction saved but document status could not be updated",
			"data": gin.H{
				"extraction_id": insertedID.Hex(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "pdf extraction completed",
		"data": gin.H{
			"extraction_id": insertedID.Hex(),
			"document_id":   documentID.Hex(),
			"field_count":   len(fields),
			"extraction":    record,
		},
	})
}
