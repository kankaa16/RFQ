package services

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AIExtractionService struct {
	Collection         *mongo.Collection
	DocumentCollection *mongo.Collection
	RFQCollection      *mongo.Collection
	CustomerCollection *mongo.Collection
	PartCollection     *mongo.Collection
}

func (s *AIExtractionService) GetExtraction(c *gin.Context) {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
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

	var extraction models.AIExtraction

	err = s.Collection.FindOne(ctx, bson.M{
		"document_id": documentID,
	}).Decode(&extraction)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "extraction not found",
			})
			return
		}

		log.Printf("extraction lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch extraction",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    extraction,
	})
}

type FieldReviewRequest struct {
	FieldName string `json:"field_name" binding:"required"`
	Action    string `json:"action" binding:"required"`
	Value     string `json:"value"`
}

type ReviewExtractionRequest struct {
	Fields []FieldReviewRequest `json:"fields" binding:"required,min=1,dive"`
}

func (s *AIExtractionService) ReviewExtraction(c *gin.Context) {
	extractionID, err := bson.ObjectIDFromHex(c.Param("extraction_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid extraction id",
		})
		return
	}

	reviewerID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	var request ReviewExtractionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid review request",
		})
		return
	}

	if len(request.Fields) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "too many fields in one review request",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var extraction models.AIExtraction

	err = s.Collection.FindOne(ctx, bson.M{
		"_id": extractionID,
	}).Decode(&extraction)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "extraction not found",
			})
			return
		}

		log.Printf("extraction lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch extraction",
		})
		return
	}

	if extraction.Status != models.AIStatusPending {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "extraction is not awaiting review",
		})
		return
	}

	if !extraction.CreatedRFQID.IsZero() {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "this extraction has already been converted",
		})
		return
	}

	if s.DocumentCollection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "document service is not configured",
		})
		return
	}

	var document models.Document

	err = s.DocumentCollection.FindOne(ctx, bson.M{
		"_id":         extraction.DocumentID,
		"uploaded_by": reviewerID,
	}).Decode(&document)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "you cannot review this document",
			})
			return
		}

		log.Printf("document ownership lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not verify document access",
		})
		return
	}

	fieldIndexes := make(map[string]int)

	for i, field := range extraction.ExtractedFields {
		fieldIndexes[field.FieldName] = i
	}

	processed := make(map[string]bool)

	for _, review := range request.Fields {
		fieldName := strings.TrimSpace(review.FieldName)
		action := strings.ToLower(strings.TrimSpace(review.Action))

		if fieldName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "field name cannot be empty",
			})
			return
		}

		if processed[fieldName] {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "duplicate field in review request: " + fieldName,
			})
			return
		}

		processed[fieldName] = true

		index, exists := fieldIndexes[fieldName]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "unknown extraction field: " + fieldName,
			})
			return
		}

		field := &extraction.ExtractedFields[index]

		if field.ReviewStatus != "" &&
			field.ReviewStatus != models.AIFieldPending {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "field has already been reviewed: " + fieldName,
			})
			return
		}

		switch action {
		case "accept":
			field.ReviewStatus = models.AIFieldAccepted
			field.IsAccepted = true
			field.ReviewedValue = field.Value

		case "edit":
			value := strings.TrimSpace(review.Value)

			if value == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "edited value cannot be empty",
				})
				return
			}

			field.ReviewStatus = models.AIFieldEdited
			field.IsAccepted = true
			field.ReviewedValue = value

		case "reject":
			field.ReviewStatus = models.AIFieldRejected
			field.IsAccepted = false
			field.ReviewedValue = ""

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "action must be accept, edit or reject",
			})
			return
		}
	}

	allReviewed := true

	for i := range extraction.ExtractedFields {
		field := &extraction.ExtractedFields[i]

		if field.ReviewStatus == "" {
			field.ReviewStatus = models.AIFieldPending
		}

		if field.ReviewStatus == models.AIFieldPending {
			allReviewed = false
		}
	}

	now := time.Now()

	status := models.AIStatusPending

	if allReviewed {
		status = models.AIStatusReviewed
	}

	update := bson.M{
		"extracted_fields": extraction.ExtractedFields,
		"updated_at":       now,
		"status":           status,
	}

	if allReviewed {
		update["reviewed_by"] = reviewerID
		update["reviewed_at"] = now
	}

	result, err := s.Collection.UpdateOne(
		ctx,
		bson.M{
			"_id":    extractionID,
			"status": models.AIStatusPending,
			"$or": []bson.M{
				{"created_rfq_id": bson.M{"$exists": false}},
				{"created_rfq_id": bson.ObjectID{}},
			},
		},
		bson.M{
			"$set": update,
		},
	)

	if err != nil {
		log.Printf("extraction review update failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save extraction review",
		})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "extraction was already reviewed, converted or changed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "extraction review saved",
		"data": gin.H{
			"status":       status,
			"all_reviewed": allReviewed,
		},
	})
}
