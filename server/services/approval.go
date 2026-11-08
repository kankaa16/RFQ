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

type ApprovalService struct {
	QuoteCollection    *mongo.Collection
	ApprovalCollection *mongo.Collection
}

type ReviewQuoteRequest struct {
	Action          string `json:"action"`
	Comments        string `json:"comments"`
	RejectionReason string `json:"rejection_reason"`
}

type ApprovalHistoryItem struct {
	Approval models.Approval `json:"approval"`
	Quote    *models.Quote   `json:"quote,omitempty"`
}

// list quotations and their approval history
func (s *ApprovalService) ListApprovals(c *gin.Context) {
	if c.GetString("role") != "sales_head" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "only the sales head can view approvals",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// fetch pending, approved and rejected quotations
	quoteFilter := bson.M{
		"status": bson.M{
			"$in": []string{
				models.QuoteStatusPending,
				models.QuoteStatusApproved,
				models.QuoteStatusRejected,
			},
		},
	}

	quoteCursor, err := s.QuoteCollection.Find(
		ctx,
		quoteFilter,
		options.Find().SetSort(bson.D{
			{Key: "created_at", Value: -1},
		}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch quotations",
		})
		return
	}
	defer quoteCursor.Close(ctx)

	quotes := make([]models.Quote, 0)

	if err := quoteCursor.All(ctx, &quotes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not decode quotations",
		})
		return
	}

	// fetch approval decisions
	historyFilter := bson.M{
		"action": bson.M{
			"$in": []string{
				models.ApprovalActionApproved,
				models.ApprovalActionRejected,
			},
		},
	}

	historyCursor, err := s.ApprovalCollection.Find(
		ctx,
		historyFilter,
		options.Find().SetSort(bson.D{
			{Key: "reviewed_at", Value: -1},
		}),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch approval history",
		})
		return
	}
	defer historyCursor.Close(ctx)

	approvals := make([]models.Approval, 0)

	if err := historyCursor.All(ctx, &approvals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not decode approval history",
		})
		return
	}

	// attach quotation details to each history record
	history := make([]ApprovalHistoryItem, 0, len(approvals))

	for _, approval := range approvals {
		var quote models.Quote

		err := s.QuoteCollection.FindOne(
			ctx,
			bson.M{"_id": approval.QuoteID},
		).Decode(&quote)

		if err != nil && err != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "could not fetch quotation details",
			})
			return
		}

		item := ApprovalHistoryItem{
			Approval: approval,
		}

		if err == nil {
			item.Quote = &quote
		}

		history = append(history, item)
	}

	// data remains compatible with the existing approval.jsx
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"count":         len(quotes),
		"pending_count": countQuotesByStatus(quotes, models.QuoteStatusPending),
		"history_count": len(history),
		"data":          quotes,
		"history":       history,
	})
}

func countQuotesByStatus(quotes []models.Quote, status string) int {
	count := 0

	for _, quote := range quotes {
		if quote.Status == status {
			count++
		}
	}

	return count
}

// approve or reject a pending quotation
func (s *ApprovalService) ReviewQuote(c *gin.Context) {
	if c.GetString("role") != "sales_head" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "only the sales head can review quotations",
		})
		return
	}

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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var quote models.Quote

	err = s.QuoteCollection.FindOne(
		ctx,
		bson.M{
			"_id":    quoteID,
			"status": models.QuoteStatusPending,
		},
	).Decode(&quote)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "pending quote not found or already reviewed",
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

	updateFields := bson.M{
		"status":     newStatus,
		"updated_at": now,
	}

	if request.Action == models.ApprovalActionApproved {
		updateFields["approved_by"] = reviewerID
		updateFields["approved_at"] = now
	}

	update := bson.M{
		"$set": updateFields,
	}

	if request.Action == models.ApprovalActionRejected {
		update["$unset"] = bson.M{
			"approved_by": "",
			"approved_at": "",
		}
	}

	// update only if the quote is still pending
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

	// save the review decision in approval history
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
		// restore the pending status if history could not be stored
		rollback := bson.M{
			"$set": bson.M{
				"status":     models.QuoteStatusPending,
				"updated_at": time.Now(),
			},
		}

		if request.Action == models.ApprovalActionApproved {
			rollback["$unset"] = bson.M{
				"approved_by": "",
				"approved_at": "",
			}
		}

		_, rollbackErr := s.QuoteCollection.UpdateOne(
			ctx,
			bson.M{
				"_id":        quoteID,
				"status":     newStatus,
				"updated_at": now,
			},
			rollback,
		)

		message := "approval history could not be saved"

		if rollbackErr != nil {
			message = "approval history could not be saved and quote rollback failed"
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "quote " + newStatus,
		"data": gin.H{
			"quote_id":     quoteID.Hex(),
			"quote_number": quote.QuoteNumber,
			"status":       newStatus,
			"reviewed_at":  now,
		},
	})
}
