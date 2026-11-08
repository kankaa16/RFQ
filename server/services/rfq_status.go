package services

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type RFQStatusService struct {
	Collection *mongo.Collection
}

type UpdateRFQStatusRequest struct {
	Status     string `json:"status" binding:"required"`
	LostReason string `json:"lost_reason"`
}

func (s *RFQStatusService) UpdateStatus(c *gin.Context) {
	rfqID, err := bson.ObjectIDFromHex(c.Param("rfq_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rfq id",
		})
		return
	}

	var request UpdateRFQStatusRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "status is required",
		})
		return
	}

	request.Status = strings.ToLower(strings.TrimSpace(request.Status))
	request.LostReason = strings.TrimSpace(request.LostReason)

	if request.Status != "sent" &&
		request.Status != "won" &&
		request.Status != "lost" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid target status",
		})
		return
	}

	role := c.GetString("role")

	if (request.Status == "won" || request.Status == "lost") &&
		role != "sales_head" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "only the sales head can mark an rfq as won or lost",
		})
		return
	}

	if request.Status == "lost" && request.LostReason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "loss reason is required",
		})
		return
	}

	now := time.Now()

	filter := bson.M{"_id": rfqID}
	updateFields := bson.M{
		"status":     request.Status,
		"updated_at": now,
	}

	switch request.Status {
	case "sent":
		filter["status"] = "estimating"
		updateFields["sent_at"] = now
		updateFields["sent_by"] = c.GetString("user_id")

	case "won":
		filter["status"] = "sent"
		updateFields["won_at"] = now
		updateFields["won_by"] = c.GetString("user_id")

	case "lost":
		filter["status"] = "sent"
		updateFields["lost_at"] = now
		updateFields["lost_by"] = c.GetString("user_id")
		updateFields["lost_reason"] = request.LostReason
	}

	result, err := s.Collection.UpdateOne(
		c.Request.Context(),
		filter,
		bson.M{"$set": updateFields},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not update rfq status",
		})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "rfq not found or status transition is not allowed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "rfq status updated successfully",
		"data": gin.H{
			"rfq_id": rfqID.Hex(),
			"status": request.Status,
		},
	})
}
