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

type ApprovalService struct {
	QuoteCollection    *mongo.Collection
	ApprovalCollection *mongo.Collection
}

type ReviewQuoteRequest struct {
	Action          string `json:"action"`
	Comments        string `json:"comments"`
	RejectionReason string `json:"rejection_reason"`
}

func (s *ApprovalService) ReviewQuote(c *gin.Context) {
	quoteID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid quote id",
		})
		return
	}

	var request ReviewQuoteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	request.Action = strings.TrimSpace(request.Action)
	request.Comments = strings.TrimSpace(request.Comments)
	request.RejectionReason = strings.TrimSpace(request.RejectionReason)

	if request.Action != models.ApprovalActionApproved &&
		request.Action != models.ApprovalActionRejected {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "action must be approved or rejected",
		})
		return
	}

	if request.Action == models.ApprovalActionRejected &&
		request.RejectionReason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "rejection reason is required",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var quote models.Quote

	err = s.QuoteCollection.FindOne(ctx, bson.M{
		"_id":    quoteID,
		"status": models.QuoteStatusPending,
	}).Decode(&quote)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "pending quote not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch quote",
		})
		return
	}

	now := time.Now()

	newStatus := models.QuoteStatusApproved

	if request.Action == models.ApprovalActionRejected {
		newStatus = models.QuoteStatusRejected
	}

	update := bson.M{
		"$set": bson.M{
			"status":     newStatus,
			"updated_at": now,
		},
	}

	// only approved quotes receive approval details
	if request.Action == models.ApprovalActionApproved {
		update["$set"].(bson.M)["approved_by"] = reviewerID
		update["$set"].(bson.M)["approved_at"] = now
	} else {
		update["$unset"] = bson.M{
			"approved_by": "",
			"approved_at": "",
		}
	}

	result, err := s.QuoteCollection.UpdateOne(
		ctx,
		bson.M{
			"_id":    quoteID,
			"status": models.QuoteStatusPending,
		},
		update,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not update quote",
		})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "quote was already reviewed",
		})
		return
	}

	approval := models.Approval{
		QuoteID:         quoteID,
		RequestedBy:     quote.PreparedBy,
		ReviewedBy:      reviewerID,
		Action:          request.Action,
		Comments:        request.Comments,
		RejectionReason: request.RejectionReason,
		RequestedAt:     quote.CreatedAt,
		ReviewedAt:      &now,
	}

	approval.CreatedAt = now
	approval.UpdatedAt = now

	_, err = s.ApprovalCollection.InsertOne(ctx, approval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "quote status updated but approval history could not be saved",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "quote " + newStatus,
		"data": gin.H{
			"quote_id": quoteID.Hex(),
			"status":   newStatus,
		},
	})
}
