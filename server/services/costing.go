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

type CostingService struct {
	Collection        *mongo.Collection
	PartCollection    *mongo.Collection
	RateCollection    *mongo.Collection
	RoutingCollection *mongo.Collection
}

type CalculateCostingRequest struct {
	PartID       string  `json:"part_id"`
	MaterialRate float64 `json:"material_rate"`
	ScrapRate    float64 `json:"scrap_rate"`
	OverheadRate float64 `json:"overhead_rate"`
	PackingCost  float64 `json:"packing_cost"`
	FreightCost  float64 `json:"freight_cost"`
	MarginRate   float64 `json:"margin_rate"`
	Volumes      []int64 `json:"volumes"`
}

func (s *CostingService) CalculateCosting(c *gin.Context) {
	var request CalculateCostingRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	partID, err := bson.ObjectIDFromHex(request.PartID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid part id",
		})
		return
	}

	if request.MaterialRate <= 0 || request.ScrapRate < 0 ||
		request.OverheadRate < 0 || request.MarginRate < 0 ||
		request.MarginRate >= 100 || len(request.Volumes) == 0 ||
		len(request.Volumes) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rates, margin or volume scenarios",
		})
		return
	}

	for _, volume := range request.Volumes {
		if volume <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "production volumes must be positive",
			})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var part models.Part

	err = s.PartCollection.FindOne(ctx, bson.M{
		"_id": partID,
	}).Decode(&part)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "part not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch part",
		})
		return
	}

	var routing models.Routing

	err = s.RoutingCollection.FindOne(ctx, bson.M{
		"part_id":     partID,
		"is_verified": true,
	}).Decode(&routing)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "verified routing not found for this part",
		})
		return
	}

	// calculate machining cost per part
	var machiningCost float64

	for _, operation := range routing.Operations {
		machiningCost += (operation.CycleTimeMinutes / 60) * operation.HourRate
	}

	// calculate material and scrap cost per part
	materialCost := part.GrossWeight * request.MaterialRate
	scrapCost := part.ScrapWeight * request.ScrapRate

	// calculate cost scenarios
	var scenarios []models.VolumeScenario

	for _, volume := range request.Volumes {
		overheadCost := (materialCost + machiningCost) * request.OverheadRate / 100

		otherCost := scrapCost + request.PackingCost + request.FreightCost

		totalCost := materialCost + machiningCost + overheadCost + otherCost

		sellingPrice := totalCost / (1 - request.MarginRate/100)

		scenarios = append(scenarios, models.VolumeScenario{
			AnnualVolume:  volume,
			MaterialCost:  materialCost,
			MachiningCost: machiningCost,
			OverheadCost:  overheadCost,
			OtherCost:     otherCost,
			TotalCost:     totalCost,
			MarginRate:    request.MarginRate,
			SellingPrice:  sellingPrice,
		})
	}

	userID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	now := time.Now()

	costing := models.Costing{
		PartID:         partID,
		MaterialCost:   materialCost,
		MachiningCost:  machiningCost,
		ScrapCost:      scrapCost,
		OverheadCost:   (materialCost + machiningCost) * request.OverheadRate / 100,
		PackingCost:    request.PackingCost,
		FreightCost:    request.FreightCost,
		TotalCost:      scenarios[0].TotalCost,
		MarginRate:     request.MarginRate,
		SellingPrice:   scenarios[0].SellingPrice,
		Scenarios:      scenarios,
		FormulaVersion: "1.0",
		PreparedBy:     userID,
		CalculatedAt:   now,
	}

	costing.CreatedAt = now
	costing.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, costing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save costing",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "costing calculated successfully",
		"data": gin.H{
			"id":        result.InsertedID,
			"scenarios": scenarios,
		},
	})
}
