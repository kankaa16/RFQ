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

	if request.MaterialRate <= 0 ||
		request.ScrapRate < 0 ||
		request.OverheadRate < 0 ||
		request.OverheadRate > 100 ||
		request.PackingCost < 0 ||
		request.FreightCost < 0 ||
		request.MarginRate < 0 ||
		request.MarginRate >= 100 ||
		len(request.Volumes) == 0 ||
		len(request.Volumes) > 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rates, costs, margin or volume scenarios",
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

	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		15*time.Second,
	)
	defer cancel()

	// fetch part
	var part models.Part

	err = s.PartCollection.FindOne(
		ctx,
		bson.M{"_id": partID},
	).Decode(&part)

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

	// fetch verified routing
	var routing models.Routing

	err = s.RoutingCollection.FindOne(
		ctx,
		bson.M{
			"part_id":     partID,
			"is_verified": true,
		},
	).Decode(&routing)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "verified routing not found for this part",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch routing",
		})
		return
	}

	if len(routing.Operations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "routing has no operations",
		})
		return
	}

	now := time.Now().UTC()

	var machiningCost float64
	var setupCostPerPart float64
	var rateSnapshots []models.RateSnapshot

	seenRates := make(map[bson.ObjectID]bool)

	// calculate machining and setup costs using rate masters
	for _, operation := range routing.Operations {
		if operation.CycleTimeMinutes <= 0 ||
			operation.SetupTimeMinutes < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "routing contains invalid operation values",
			})
			return
		}

		if operation.SetupTimeMinutes > 0 && operation.BatchSize <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "batch size must be positive when setup time is specified",
			})
			return
		}

		if operation.RateMasterID == bson.NilObjectID {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "rate master is missing for operation: " + operation.Name,
			})
			return
		}

		var rate models.RateMaster

		err := s.RateCollection.FindOne(
			ctx,
			bson.M{"_id": operation.RateMasterID},
		).Decode(&rate)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "rate master not found for operation: " + operation.Name,
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "could not fetch machine rate master",
			})
			return
		}

		if rate.RateType != models.RateTypeMachine || rate.Rate <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invalid machine rate master for operation: " + operation.Name,
			})
			return
		}

		// allow a rate only within its effective date range
		if rate.EffectiveFrom.After(now) ||
			(!rate.EffectiveTo.IsZero() && !now.Before(rate.EffectiveTo)) {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "machine rate is not effective today: " + rate.RateCode,
			})
			return
		}

		// use the rate master value, not the routing's stored hourly rate
		hourRate := rate.Rate

		machiningCost +=
			(operation.CycleTimeMinutes / 60) * hourRate

		if operation.SetupTimeMinutes > 0 {
			setupCostPerPart +=
				(operation.SetupTimeMinutes * hourRate) /
					(60 * float64(operation.BatchSize))
		}

		// save each distinct rate version once
		if !seenRates[operation.RateMasterID] {
			rateSnapshots = append(rateSnapshots, models.RateSnapshot{
				RateID:   rate.ID,
				RateCode: rate.RateCode,
				Version:  rate.Version,
				Rate:     rate.Rate,
				Unit:     rate.Unit,
			})

			seenRates[operation.RateMasterID] = true
		}
	}

	// calculate material and scrap costs per part
	materialCost := part.GrossWeight * request.MaterialRate
	scrapCost := part.ScrapWeight * request.ScrapRate

	// calculate overhead on material, machining and setup
	overheadCost := (materialCost +
		machiningCost +
		setupCostPerPart) * request.OverheadRate / 100

	// calculate additional costs per part
	otherCost := scrapCost +
		request.PackingCost +
		request.FreightCost

	// calculate independent volume scenarios
	var scenarios []models.VolumeScenario

	for _, volume := range request.Volumes {
		totalCost := materialCost +
			machiningCost +
			setupCostPerPart +
			overheadCost +
			otherCost

		sellingPrice := totalCost / (1 - request.MarginRate/100)

		scenarios = append(scenarios, models.VolumeScenario{
			AnnualVolume:  volume,
			MaterialCost:  materialCost,
			MachiningCost: machiningCost,
			OverheadCost:  overheadCost,
			OtherCost:     setupCostPerPart + otherCost,
			TotalCost:     totalCost,
			MarginRate:    request.MarginRate,
			SellingPrice:  sellingPrice,
		})
	}

	// get logged-in user
	userID, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "invalid logged-in user",
		})
		return
	}

	rfqID := part.RFQID

	costing := models.Costing{
		PartID:         partID,
		RFQID:          rfqID,
		MaterialCost:   materialCost,
		MachiningCost:  machiningCost,
		ScrapCost:      scrapCost,
		SetupCost:      setupCostPerPart,
		OverheadCost:   overheadCost,
		PackingCost:    request.PackingCost,
		FreightCost:    request.FreightCost,
		TotalCost:      scenarios[0].TotalCost,
		MarginRate:     request.MarginRate,
		SellingPrice:   scenarios[0].SellingPrice,
		Scenarios:      scenarios,
		RateSnapshots:  rateSnapshots,
		FormulaVersion: "1.2",
		Assumptions:    "machine hourly rates fetched from effective rate master versions; material rate, scrap rate, overhead rate, packing cost and freight cost supplied in costing request; setup allocated using routing batch size",
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
			"id":             result.InsertedID,
			"part_id":        partID,
			"rfq_id":         rfqID,
			"material_cost":  materialCost,
			"machining_cost": machiningCost,
			"setup_cost":     setupCostPerPart,
			"overhead_cost":  overheadCost,
			"scrap_cost":     scrapCost,
			"rate_snapshots": rateSnapshots,
			"scenarios":      scenarios,
		},
	})
}
