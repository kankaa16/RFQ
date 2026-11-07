package services

import (
	"context"
	"net/http"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type RFQService struct {
	Collection *mongo.Collection
}

type CreateRFQRequest struct {
	RFQNumber           string    `json:"rfq_number"`
	CustomerID          string    `json:"customer_id"`
	CustomerName        string    `json:"customer_name"`
	PartCount           int       `json:"part_count"`
	SourceChannel       string    `json:"source_channel"`
	Priority            string    `json:"priority"`
	DueDate             time.Time `json:"due_date"`
	TargetSOPDate       time.Time `json:"target_sop_date"`
	ExpectedAnnualValue float64   `json:"expected_annual_value"`
	InternalNotes       string    `json:"internal_notes"`
}

func (s *RFQService) CreateRFQ(c *gin.Context) {
	var request CreateRFQRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	request.RFQNumber = strings.TrimSpace(request.RFQNumber)
	request.CustomerName = strings.TrimSpace(request.CustomerName)

	if request.RFQNumber == "" || request.CustomerName == "" || request.PartCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "rfq number, customer name and valid part count are required",
		})
		return
	}

	customerID, err := bson.ObjectIDFromHex(request.CustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid customer id",
		})
		return
	}

	ownerID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	count, err := s.Collection.CountDocuments(ctx, bson.M{
		"rfq_number": request.RFQNumber,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not check rfq number",
		})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "rfq number already exists",
		})
		return
	}

	now := time.Now()

	rfq := models.RFQ{
		RFQNumber:           request.RFQNumber,
		CustomerID:          customerID,
		CustomerName:        request.CustomerName,
		OwnerID:             ownerID,
		PartCount:           request.PartCount,
		SourceChannel:       request.SourceChannel,
		Priority:            request.Priority,
		ReceivedAt:          now,
		DueDate:             request.DueDate,
		TargetSOPDate:       request.TargetSOPDate,
		ExpectedAnnualValue: request.ExpectedAnnualValue,
		Currency:            "INR",
		Status:              models.RFQStatusEstimating,
		InternalNotes:       request.InternalNotes,
	}

	rfq.CreatedAt = now
	rfq.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, rfq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not create rfq",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "rfq created successfully",
		"data": gin.H{
			"id":         result.InsertedID,
			"rfq_number": rfq.RFQNumber,
			"status":     rfq.Status,
		},
	})
}

func (s *RFQService) ListRFQs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := s.Collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch rfqs",
		})
		return
	}
	defer cursor.Close(ctx)

	var rfqs []models.RFQ

	if err := cursor.All(ctx, &rfqs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read rfqs",
		})
		return
	}

	if rfqs == nil {
		rfqs = []models.RFQ{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(rfqs),
		"data":    rfqs,
	})
}
