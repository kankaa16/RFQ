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
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func normalizeRateType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validRateType(value string) bool {
	switch value {
	case models.RateTypeMaterial,
		models.RateTypeMachine,
		models.RateTypeLabour,
		models.RateTypeOverhead,
		models.RateTypePacking,
		models.RateTypeFreight:
		return true
	default:
		return false
	}
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
	request.RateType = normalizeRateType(request.RateType)
	request.Category = strings.TrimSpace(request.Category)
	request.Unit = strings.TrimSpace(request.Unit)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.MaterialGrade = strings.TrimSpace(request.MaterialGrade)
	request.MachineType = strings.TrimSpace(request.MachineType)

	if request.RateCode == "" ||
		request.RateName == "" ||
		request.Rate <= 0 ||
		request.Unit == "" ||
		!validRateType(request.RateType) ||
		request.EffectiveFrom.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "provide a valid rate code, name, type, positive rate, unit and effective-from date",
		})
		return
	}

	if request.Currency == "" {
		request.Currency = "INR"
	}

	if request.Currency != "INR" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "only INR currency is currently supported",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Find the latest version for this rate code.
	var previous models.RateMaster

	err = s.Collection.FindOne(
		ctx,
		bson.M{"rate_code": request.RateCode},
		options.FindOne().SetSort(bson.D{
			{Key: "version", Value: -1},
		}),
	).Decode(&previous)

	version := 1

	if err == nil {
		version = previous.Version + 1

		// A new version must not begin before the previous version.
		if request.EffectiveFrom.Before(previous.EffectiveFrom) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "new rate effective date cannot be earlier than the previous version",
			})
			return
		}

		if request.EffectiveFrom.Equal(previous.EffectiveFrom) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "new rate must have a different effective-from date",
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

	newRate := models.RateMaster{
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
		UpdatedBy:     userID,
	}

	newRate.CreatedAt = now
	newRate.UpdatedAt = now

	// Use a transaction so the previous version is not closed
	// unless the new version is successfully inserted.
	client := s.Collection.Database().Client()

	session, err := client.StartSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not start rate version transaction",
		})
		return
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (interface{}, error) {
		if version > 1 {
			// Close the previous version on the day the new one starts.
			_, updateErr := s.Collection.UpdateOne(
				txCtx,
				bson.M{"_id": previous.ID},
				bson.M{
					"$set": bson.M{
						"is_active":    false,
						"effective_to": request.EffectiveFrom,
						"updated_at":   now,
					},
				},
			)

			if updateErr != nil {
				return nil, updateErr
			}
		}

		result, insertErr := s.Collection.InsertOne(txCtx, newRate)
		if insertErr != nil {
			return nil, insertErr
		}

		return result.InsertedID, nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save new rate version: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "rate version created successfully",
		"data": gin.H{
			"id":             newRate.ID,
			"rate_code":      newRate.RateCode,
			"rate":           newRate.Rate,
			"unit":           newRate.Unit,
			"version":        newRate.Version,
			"effective_from": newRate.EffectiveFrom,
			"is_active":      newRate.IsActive,
		},
	})
}

func (s *RateMasterService) ListRates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	filter := bson.M{}

	if rateType := normalizeRateType(c.Query("type")); rateType != "" {
		filter["rate_type"] = rateType
	}

	if category := strings.TrimSpace(c.Query("category")); category != "" {
		filter["category"] = category
	}

	if code := strings.ToUpper(strings.TrimSpace(c.Query("rate_code"))); code != "" {
		filter["rate_code"] = code
	}

	if c.Query("active") == "true" {
		filter["is_active"] = true
	} else if c.Query("active") == "false" {
		filter["is_active"] = false
	}

	findOptions := options.Find().SetSort(bson.D{
		{Key: "rate_code", Value: 1},
		{Key: "version", Value: -1},
	})

	cursor, err := s.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch rates",
		})
		return
	}
	defer cursor.Close(ctx)

	rates := []models.RateMaster{}

	if err := cursor.All(ctx, &rates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read rates",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(rates),
		"data":    rates,
	})
}

func (s *RateMasterService) GetRateHistory(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("rate_code")))

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "rate code is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := s.Collection.Find(
		ctx,
		bson.M{"rate_code": code},
		options.Find().SetSort(bson.D{
			{Key: "version", Value: -1},
		}),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch rate history",
		})
		return
	}
	defer cursor.Close(ctx)

	rates := []models.RateMaster{}

	if err := cursor.All(ctx, &rates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read rate history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(rates),
		"data":    rates,
	})
}
