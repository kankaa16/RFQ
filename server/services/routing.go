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

type RoutingService struct {
	Collection     *mongo.Collection
	PartCollection *mongo.Collection
}

type SaveRoutingRequest struct {
	PartID     string                    `json:"part_id"`
	Source     string                    `json:"source"`
	Operations []models.RoutingOperation `json:"operations"`
}

func (s *RoutingService) SaveRouting(c *gin.Context) {
	var request SaveRoutingRequest

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

	if len(request.Operations) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "at least one operation is required",
		})
		return
	}

	for i, operation := range request.Operations {
		if operation.Name == "" || operation.MachineType == "" ||
			operation.CycleTimeMinutes <= 0 || operation.HourRate <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "each operation needs a name, machine, cycle time and hour rate",
			})
			return
		}

		request.Operations[i].Sequence = i + 1
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := s.PartCollection.FindOne(ctx, bson.M{
		"_id": partID,
	}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "part not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not verify part",
		})
		return
	}

	now := time.Now()

	routing := models.Routing{
		PartID:     partID,
		Operations: request.Operations,
		Source:     request.Source,
		IsVerified: request.Source == models.RoutingSourceManual,
	}

	if routing.Source == "" {
		routing.Source = models.RoutingSourceManual
		routing.IsVerified = true
	}

	if routing.IsVerified {
		routing.VerifiedBy, _ = bson.ObjectIDFromHex(c.GetString("user_id"))
	}

	routing.CreatedAt = now
	routing.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, routing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not save routing",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "routing saved successfully",
		"data": gin.H{
			"id":          result.InsertedID,
			"part_id":     partID.Hex(),
			"operations":  routing.Operations,
			"is_verified": routing.IsVerified,
		},
	})
}

func (s *RoutingService) GetRouting(c *gin.Context) {
	partID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid part id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var routing models.Routing

	err = s.Collection.FindOne(
		ctx,
		bson.M{"part_id": partID},
	).Decode(&routing)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "routing not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch routing",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    routing,
	})
}
