package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const approvalThreshold = 2500000.0

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

	request.RFQID = strings.TrimSpace(request.RFQID)
	request.CustomerID = strings.TrimSpace(request.CustomerID)

	if request.RFQID == "" ||
		request.CustomerID == "" ||
		len(request.CostingIDs) == 0 ||
		request.GSTPercent < 0 ||
		request.GSTPercent > 100 ||
		request.ValidityDays <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rfq, customer, costings, gst or validity days",
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

	costingIDs := make([]bson.ObjectID, 0, len(request.CostingIDs))
	seen := make(map[bson.ObjectID]bool)

	var subtotal float64
	var annualValue float64

	// validate costings and calculate annual quotation value
	for _, id := range request.CostingIDs {
		costingID, err := bson.ObjectIDFromHex(strings.TrimSpace(id))
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

		if costing.SellingPrice <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "costing selling price must be greater than zero",
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

	// send quotations above the annual value threshold for sales approval
	status := models.QuoteStatusDraft

	if annualValue > approvalThreshold {
		status = models.QuoteStatusPending
	}

	now := time.Now()

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
		PaymentTerms:     strings.TrimSpace(request.PaymentTerms),
		DeliveryTerms:    strings.TrimSpace(request.DeliveryTerms),
		ValidityDays:     request.ValidityDays,
		DeliveryLocation: strings.TrimSpace(request.DeliveryLocation),
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
			"id":                result.InsertedID,
			"quote_number":      quote.QuoteNumber,
			"subtotal":          subtotal,
			"gst_amount":        gstAmount,
			"grand_total":       grandTotal,
			"annual_value":      annualValue,
			"approval_required": status == models.QuoteStatusPending,
			"status":            status,
		},
	})
}

func (s *QuoteService) ListQuotes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	findOptions := options.Find().SetSort(bson.D{
		{Key: "created_at", Value: -1},
	})

	cursor, err := s.Collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch quotes",
		})
		return
	}
	defer cursor.Close(ctx)

	quotes := make([]models.Quote, 0)

	if err := cursor.All(ctx, &quotes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read quotes",
		})
		return
	}

	data := make([]gin.H, 0, len(quotes))

	for _, quote := range quotes {
		var rfq models.RFQ
		var customer models.Customer

		rfqNumber := ""
		customerName := ""

		// fetch associated rfq
		err := s.RFQCollection.FindOne(ctx, bson.M{
			"_id": quote.RFQID,
		}).Decode(&rfq)

		if err == nil {
			rfqNumber = rfq.RFQNumber
			customerName = rfq.CustomerName
		}

		// fetch associated customer
		err = s.CustomerCollection.FindOne(ctx, bson.M{
			"_id": quote.CustomerID,
		}).Decode(&customer)

		if err == nil {
			customerName = customer.CompanyName
		}

		data = append(data, gin.H{
			"id":                quote.ID.Hex(),
			"quote_number":      quote.QuoteNumber,
			"rfq_id":            quote.RFQID.Hex(),
			"rfq_number":        rfqNumber,
			"customer_id":       quote.CustomerID.Hex(),
			"customer_name":     customerName,
			"subtotal":          quote.Subtotal,
			"gst_percent":       quote.GSTPercent,
			"gst_amount":        quote.GSTAmount,
			"grand_total":       quote.GrandTotal,
			"annual_value":      quote.AnnualValue,
			"currency":          quote.Currency,
			"payment_terms":     quote.PaymentTerms,
			"delivery_terms":    quote.DeliveryTerms,
			"validity_days":     quote.ValidityDays,
			"delivery_location": quote.DeliveryLocation,
			"status":            quote.Status,
			"prepared_by":       quote.PreparedBy.Hex(),
			"approved_by":       objectIDHex(quote.ApprovedBy),
			"created_at":        quote.CreatedAt,
			"updated_at":        quote.UpdatedAt,
			"approved_at":       quote.ApprovedAt,
			"sent_at":           quote.SentAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(data),
		"data":    data,
	})
}

func objectIDHex(id bson.ObjectID) string {
	if id.IsZero() {
		return ""
	}

	return id.Hex()
}
