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

type RateMasterService struct {
	Collection *mongo.Collection
}

type CreateRateRequest struct {
	RateCode      string    `json:"rate_code"`
	RateName      string    `json:"rate_name"`
	RateType      string    `json:"rate_type"`
	Category      string    `json:"category"`
	MaterialGrade string    `json:"material_grade"`
	MachineType   string    `json:"machine_type"`
	Rate          float64   `json:"rate"`
	Unit          string    `json:"unit"`
	Currency      string    `json:"currency"`
	EffectiveFrom time.Time `json:"effective_from"`
}

func (s *RateMasterService) CreateRate(c *gin.Context) {
	var request CreateRateRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	request.RateCode = strings.ToUpper(strings.TrimSpace(request.RateCode))
	request.RateName = strings.TrimSpace(request.RateName)

	if request.RateCode == "" || request.RateName == "" ||
		request.RateType == "" || request.Rate <= 0 || request.Unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "rate code, name, type, positive rate and unit are required",
		})
		return
	}

	if request.Currency == "" {
		request.Currency = "INR"
	}

	updatedBy, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var previous models.RateMaster

	err = s.Collection.FindOne(
		ctx,
		bson.M{"rate_code": request.RateCode},
	).Decode(&previous)

	version := 1

	if err == nil {
		version = previous.Version + 1

		_, err = s.Collection.UpdateOne(
			ctx,
			bson.M{"_id": previous.ID},
			bson.M{"$set": bson.M{
				"is_active":    false,
				"effective_to": request.EffectiveFrom,
				"updated_at":   time.Now(),
			}},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "could not close previous rate version",
			})
			return
		}
	} else if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not check previous rate",
		})
		return
	}

	now := time.Now()

	rate := models.RateMaster{
		RateCode:      request.RateCode,
		RateName:      request.RateName,
		RateType:      request.RateType,
		Category:      request.Category,
		MaterialGrade: request.MaterialGrade,
		MachineType:   request.MachineType,
		Rate:          request.Rate,
		Unit:          request.Unit,
		Currency:      request.Currency,
		Version:       version,
		EffectiveFrom: request.EffectiveFrom,
		IsActive:      true,
		UpdatedBy:     updatedBy,
	}

	rate.CreatedAt = now
	rate.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, rate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not create rate",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "rate created successfully",
		"data": gin.H{
			"id":      result.InsertedID,
			"version": version,
			"rate":    rate.Rate,
		},
	})
}

func (s *RateMasterService) ListRates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	filter := bson.M{}

	if rateType := c.Query("type"); rateType != "" {
		filter["rate_type"] = rateType
	}

	if c.Query("active") == "true" {
		filter["is_active"] = true
	}

	cursor, err := s.Collection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch rates",
		})
		return
	}
	defer cursor.Close(ctx)

	var rates []models.RateMaster

	if err := cursor.All(ctx, &rates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read rates",
		})
		return
	}

	if rates == nil {
		rates = []models.RateMaster{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(rates),
		"data":    rates,
	})
}
