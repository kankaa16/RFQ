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

	request.PartID = strings.TrimSpace(request.PartID)

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

	if request.Source == "" {
		request.Source = models.RoutingSourceManual
	}

	if request.Source != models.RoutingSourceManual {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "only manual routing can be verified through this endpoint",
		})
		return
	}

	for i := range request.Operations {
		operation := &request.Operations[i]

		operation.Name = strings.TrimSpace(operation.Name)
		operation.MachineType = strings.TrimSpace(operation.MachineType)

		if operation.Name == "" || operation.MachineType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "each operation needs a name and machine type",
			})
			return
		}

		if operation.CycleTimeMinutes <= 0 ||
			operation.SetupTimeMinutes < 0 ||
			operation.BatchSize <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "each operation needs valid cycle time, setup time and batch size",
			})
			return
		}

		if operation.RateMasterID.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "select a valid machine rate master for operation: " + operation.Name,
			})
			return
		}

		if operation.HourRate <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "machine rate must be greater than zero for operation: " + operation.Name,
			})
			return
		}

		operation.Sequence = i + 1
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// verify that the part exists
	err = s.PartCollection.FindOne(ctx, bson.M{
		"_id": partID,
	}).Err()

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
			"message": "could not verify part",
		})
		return
	}

	// verify the authenticated user
	verifiedBy, err := bson.ObjectIDFromHex(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "valid authenticated user is required to verify routing",
		})
		return
	}

	now := time.Now()

	update := bson.M{
		"part_id":     partID,
		"operations":  request.Operations,
		"source":      request.Source,
		"is_verified": true,
		"verified_by": verifiedBy,
		"updated_at":  now,
	}

	updateDocument := bson.M{
		"$set": update,
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	// update every existing routing record for this part.
	// this also repairs older duplicate routing records.
	result, err := s.Collection.UpdateMany(
		ctx,
		bson.M{"part_id": partID},
		updateDocument,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not update routing",
		})
		return
	}

	// create a routing if this part does not have one yet
	if result.MatchedCount == 0 {
		_, err = s.Collection.UpdateOne(
			ctx,
			bson.M{"part_id": partID},
			updateDocument,
			options.UpdateOne().SetUpsert(true),
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "could not create routing",
			})
			return
		}
	}

	// retrieve the updated routing
	var savedRouting models.Routing

	err = s.Collection.FindOne(
		ctx,
		bson.M{"part_id": partID},
		options.FindOne().SetSort(bson.D{
			{Key: "updated_at", Value: -1},
		}),
	).Decode(&savedRouting)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "routing was saved but could not be retrieved",
		})
		return
	}

	// verify that every operation retained its rate master reference
	for _, operation := range savedRouting.Operations {
		if operation.RateMasterID.IsZero() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "saved routing is missing a rate master for operation: " + operation.Name,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "routing saved and verified successfully",
		"data":    savedRouting,
	})
}

func (s *RoutingService) GetRouting(c *gin.Context) {
	partID, err := bson.ObjectIDFromHex(c.Param("part_id"))
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
		options.FindOne().SetSort(bson.D{
			{Key: "updated_at", Value: -1},
		}),
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
