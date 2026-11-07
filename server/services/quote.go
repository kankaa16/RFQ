package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type QuoteService struct {
	Collection         *mongo.Collection
	CostingCollection  *mongo.Collection
	RFQCollection      *mongo.Collection
	CustomerCollection *mongo.Collection
	PartCollection     *mongo.Collection
}

type CreateQuoteRequest struct {
	RFQID            string   `json:"rfq_id"`
	CustomerID       string   `json:"customer_id"`
	CostingIDs       []string `json:"costing_ids"`
	GSTPercent       float64  `json:"gst_percent"`
	PaymentTerms     string   `json:"payment_terms"`
	DeliveryTerms    string   `json:"delivery_terms"`
	ValidityDays     int      `json:"validity_days"`
	DeliveryLocation string   `json:"delivery_location"`
}

func (s *QuoteService) CreateQuote(c *gin.Context) {
	var request CreateQuoteRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	if len(request.CostingIDs) == 0 ||
		request.GSTPercent < 0 ||
		request.GSTPercent > 100 ||
		request.ValidityDays <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid costings, gst or validity days",
		})
		return
	}

	rfqID, err := bson.ObjectIDFromHex(request.RFQID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rfq id",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// verify rfq
	var rfq models.RFQ

	err = s.RFQCollection.FindOne(ctx, bson.M{
		"_id": rfqID,
	}).Decode(&rfq)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "rfq not found",
		})
		return
	}

	// verify customer
	var customer models.Customer

	err = s.CustomerCollection.FindOne(ctx, bson.M{
		"_id": customerID,
	}).Decode(&customer)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "customer not found",
		})
		return
	}

	if rfq.CustomerID != customerID {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "customer does not belong to this rfq",
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

	costingIDs := make([]bson.ObjectID, 0, len(request.CostingIDs))
	seen := make(map[bson.ObjectID]bool)

	var subtotal float64
	var annualValue float64

	// validate each costing and calculate quote values
	for _, id := range request.CostingIDs {
		costingID, err := bson.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invalid costing id",
			})
			return
		}

		if seen[costingID] {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "duplicate costing ids are not allowed",
			})
			return
		}
		seen[costingID] = true

		var costing models.Costing

		err = s.CostingCollection.FindOne(ctx, bson.M{
			"_id":    costingID,
			"rfq_id": rfqID,
		}).Decode(&costing)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "costing not found for this rfq",
			})
			return
		}

		var part models.Part

		err = s.PartCollection.FindOne(ctx, bson.M{
			"_id":    costing.PartID,
			"rfq_id": rfqID,
		}).Decode(&part)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "part not found for this costing",
			})
			return
		}

		if part.AnnualVolume <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "part annual volume must be greater than zero",
			})
			return
		}

		subtotal += costing.SellingPrice
		annualValue += costing.SellingPrice * float64(part.AnnualVolume)

		costingIDs = append(costingIDs, costingID)
	}

	// calculate gst on the unit-price subtotal
	gstAmount := subtotal * request.GSTPercent / 100
	grandTotal := subtotal + gstAmount

	// require approval when annual value exceeds 25 lakh
	status := models.QuoteStatusDraft

	if annualValue > 2500000 {
		status = models.QuoteStatusPending
	}

	now := time.Now()

	// use an object id suffix to avoid timestamp-only collisions
	quoteNumber := fmt.Sprintf(
		"QT-%s-%s",
		now.Format("20060102-150405"),
		bson.NewObjectID().Hex()[:8],
	)

	quote := models.Quote{
		QuoteNumber:      quoteNumber,
		RFQID:            rfqID,
		CustomerID:       customerID,
		CostingIDs:       costingIDs,
		Subtotal:         subtotal,
		GSTPercent:       request.GSTPercent,
		GSTAmount:        gstAmount,
		GrandTotal:       grandTotal,
		AnnualValue:      annualValue,
		Currency:         customer.PreferredCurrency,
		PaymentTerms:     request.PaymentTerms,
		DeliveryTerms:    request.DeliveryTerms,
		ValidityDays:     request.ValidityDays,
		DeliveryLocation: request.DeliveryLocation,
		Status:           status,
		PreparedBy:       userID,
	}

	quote.CreatedAt = now
	quote.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, quote)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not create quote",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "quote created successfully",
		"data": gin.H{
			"id":           result.InsertedID,
			"quote_number": quote.QuoteNumber,
			"subtotal":     subtotal,
			"gst_amount":   gstAmount,
			"grand_total":  grandTotal,
			"annual_value": annualValue,
			"status":       status,
		},
	})
}
