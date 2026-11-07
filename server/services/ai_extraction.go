package services

import (
	"context"
	"net/http"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AIExtractionService struct {
	Collection *mongo.Collection
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

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

type ReviewExtractionRequest struct {
	AcceptedFields []string `json:"accepted_fields"`
}

func (s *AIExtractionService) ReviewExtraction(c *gin.Context) {
	extractionID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid extraction id",
		})
		return
	}

	var request ReviewExtractionRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var extraction models.AIExtraction

	err = s.Collection.FindOne(ctx, bson.M{
		"_id": extractionID,
	}).Decode(&extraction)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "extraction not found",
		})
		return
	}

	accepted := make(map[string]bool)

	for _, field := range request.AcceptedFields {
		accepted[field] = true
	}

	for i := range extraction.ExtractedFields {
		extraction.ExtractedFields[i].IsAccepted =
			accepted[extraction.ExtractedFields[i].FieldName]
	}

	reviewerID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	now := time.Now()

	_, err = s.Collection.UpdateOne(ctx,
		bson.M{"_id": extractionID},
		bson.M{"$set": bson.M{
			"extracted_fields": extraction.ExtractedFields,
			"status":           models.AIStatusReviewed,
			"reviewed_by":      reviewerID,
			"reviewed_at":      now,
			"updated_at":       now,
		}},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save extraction review",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "extraction review saved",
	})
}
