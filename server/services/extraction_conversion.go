package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type convertedPart struct {
	PartNumber           string
	PartName             string
	DrawingNumber        string
	DrawingRevision      string
	MaterialGrade        string
	ManufacturingProcess string
	GrossWeight          float64
	NetWeight            float64
	AnnualVolume         int64
	TargetPrice          float64
}

func (s *AIExtractionService) ConvertToRFQ(c *gin.Context) {
	if s.Collection == nil ||
		s.RFQCollection == nil ||
		s.CustomerCollection == nil ||
		s.PartCollection == nil ||
		s.DocumentCollection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "conversion service is not configured",
		})
		return
	}

	extractionID, err := bson.ObjectIDFromHex(c.Param("extraction_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid extraction id",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	var extraction models.AIExtraction

	err = s.Collection.FindOne(ctx, bson.M{
		"_id": extractionID,
	}).Decode(&extraction)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "extraction not found",
			})
			return
		}

		log.Printf("extraction lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch extraction",
		})
		return
	}

	if extraction.Status != models.AIStatusReviewed {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "all extraction fields must be reviewed first",
		})
		return
	}

	if !extraction.CreatedRFQID.IsZero() {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "this extraction has already been converted",
			"data": gin.H{
				"rfq_id": extraction.CreatedRFQID.Hex(),
			},
		})
		return
	}

	var document models.Document

	err = s.DocumentCollection.FindOne(ctx, bson.M{
		"_id":         extraction.DocumentID,
		"uploaded_by": userID,
	}).Decode(&document)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "document not found or access denied",
			})
			return
		}

		log.Printf("document ownership lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not verify document access",
		})
		return
	}

	values := make(map[string]string)

	for _, field := range extraction.ExtractedFields {
		if !field.IsAccepted {
			continue
		}

		if field.ReviewStatus != models.AIFieldAccepted &&
			field.ReviewStatus != models.AIFieldEdited {
			continue
		}

		value := field.Value

		if field.ReviewStatus == models.AIFieldEdited {
			value = field.ReviewedValue
		}

		values[field.FieldName] = strings.TrimSpace(value)
	}

	customerName := values["customer_name"]
	rfqNumber := values["rfq_number"]

	if customerName == "" || rfqNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "accepted customer name and rfq number are required",
		})
		return
	}

	parts, err := buildAcceptedParts(values)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	if len(parts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "no accepted parts found",
		})
		return
	}

	customerPattern := "^" + regexp.QuoteMeta(customerName) + "$"

	var customer models.Customer

	err = s.CustomerCollection.FindOne(ctx, bson.M{
		"company_name": bson.M{
			"$regex":   customerPattern,
			"$options": "i",
		},
		"is_active": true,
	}).Decode(&customer)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "active customer not found; create the customer or correct the extracted name",
			})
			return
		}

		log.Printf("customer lookup failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not find customer",
		})
		return
	}

	now := time.Now()

	rfq := models.RFQ{
		RFQNumber:     rfqNumber,
		CustomerID:    customer.ID,
		CustomerName:  customer.CompanyName,
		OwnerID:       userID,
		PartCount:     len(parts),
		SourceChannel: models.RFQSourceManual,
		Priority:      models.RFQPriorityMedium,
		ReceivedAt:    now,
		Currency:      values["currency"],
		Status:        models.RFQStatusEstimating,
		InternalNotes: "created from reviewed ai extraction",
	}

	if rfq.Currency == "" {
		rfq.Currency = customer.PreferredCurrency
	}

	if rfq.Currency == "" {
		rfq.Currency = "INR"
	}

	if dueDate := values["due_date"]; dueDate != "" {
		parsedDate, err := parseExtractionDate(dueDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invalid accepted due date; use yyyy-mm-dd",
			})
			return
		}

		rfq.DueDate = parsedDate
	}

	var expectedAnnualValue float64

	for _, part := range parts {
		expectedAnnualValue += part.TargetPrice * float64(part.AnnualVolume)
	}

	rfq.ExpectedAnnualValue = expectedAnnualValue
	rfq.CreatedAt = now
	rfq.UpdatedAt = now

	session, err := s.Collection.Database().Client().StartSession()
	if err != nil {
		log.Printf("database session creation failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not start database transaction",
		})
		return
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(
		ctx,
		func(sc context.Context) (interface{}, error) {
			// reserve this extraction; missing and zero ids are both allowed
			reservation, err := s.Collection.UpdateOne(
				sc,
				bson.M{
					"_id":    extractionID,
					"status": models.AIStatusReviewed,
					"$or": []bson.M{
						{"created_rfq_id": bson.M{"$exists": false}},
						{"created_rfq_id": bson.ObjectID{}},
					},
				},
				bson.M{
					"$set": bson.M{
						"created_rfq_id": bson.ObjectID{},
						"updated_at":     now,
					},
				},
			)

			if err != nil {
				return nil, err
			}

			if reservation.MatchedCount == 0 {
				return nil, fmt.Errorf("extraction has already been converted or changed")
			}

			rfqResult, err := s.RFQCollection.InsertOne(sc, rfq)
			if err != nil {
				return nil, err
			}

			rfqID, ok := rfqResult.InsertedID.(bson.ObjectID)
			if !ok {
				return nil, fmt.Errorf("could not read created rfq id")
			}

			for _, partData := range parts {
				part := models.Part{
					RFQID:                rfqID,
					PartNumber:           partData.PartNumber,
					PartName:             partData.PartName,
					DrawingNumber:        partData.DrawingNumber,
					DrawingRevision:      partData.DrawingRevision,
					MaterialGrade:        partData.MaterialGrade,
					ManufacturingProcess: partData.ManufacturingProcess,
					GrossWeight:          partData.GrossWeight,
					NetWeight:            partData.NetWeight,
					ScrapWeight:          partData.GrossWeight - partData.NetWeight,
					AnnualVolume:         partData.AnnualVolume,
					MonthlyVolume:        partData.AnnualVolume / 12,
					CustomerTargetPrice:  partData.TargetPrice,
					MaterialConfirmed:    false,
					DrawingVerified:      false,
				}

				part.CreatedAt = now
				part.UpdatedAt = now

				_, err := s.PartCollection.InsertOne(sc, part)
				if err != nil {
					return nil, err
				}
			}

			updateResult, err := s.Collection.UpdateOne(
				sc,
				bson.M{
					"_id":            extractionID,
					"status":         models.AIStatusReviewed,
					"created_rfq_id": bson.ObjectID{},
				},
				bson.M{
					"$set": bson.M{
						"created_rfq_id": rfqID,
						"converted_at":   now,
						"updated_at":     now,
					},
				},
			)

			if err != nil {
				return nil, err
			}

			if updateResult.MatchedCount == 0 {
				return nil, fmt.Errorf("could not finalize extraction conversion")
			}

			return rfqID, nil
		},
	)

	if err != nil {
		log.Printf("extraction conversion failed: %v", err)

		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "conversion failed; check server logs for details",
		})
		return
	}

	rfqID, ok := result.(bson.ObjectID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read created rfq id",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "rfq and parts created successfully",
		"data": gin.H{
			"rfq_id":     rfqID.Hex(),
			"rfq_number": rfq.RFQNumber,
			"part_count": len(parts),
		},
	})
}

func buildAcceptedParts(values map[string]string) ([]convertedPart, error) {
	partsByIndex := make(map[int]*convertedPart)

	for fieldName, value := range values {
		if !strings.HasPrefix(fieldName, "parts[") {
			continue
		}

		closingBracket := strings.Index(fieldName, "].")
		if closingBracket == -1 {
			continue
		}

		indexText := fieldName[len("parts["):closingBracket]
		fieldNamePart := fieldName[closingBracket+2:]

		index, err := strconv.Atoi(indexText)
		if err != nil || index < 0 || index >= 100 {
			return nil, fmt.Errorf("invalid extracted part index")
		}

		if partsByIndex[index] == nil {
			partsByIndex[index] = &convertedPart{}
		}

		part := partsByIndex[index]

		switch fieldNamePart {
		case "part_number":
			part.PartNumber = value
		case "part_name":
			part.PartName = value
		case "drawing_number":
			part.DrawingNumber = value
		case "drawing_revision":
			part.DrawingRevision = value
		case "material_grade":
			part.MaterialGrade = value
		case "manufacturing_process":
			part.ManufacturingProcess = value
		case "gross_weight":
			part.GrossWeight, err = strconv.ParseFloat(value, 64)
		case "net_weight":
			part.NetWeight, err = strconv.ParseFloat(value, 64)
		case "annual_volume":
			part.AnnualVolume, err = strconv.ParseInt(value, 10, 64)
		case "target_price":
			part.TargetPrice, err = strconv.ParseFloat(value, 64)
		default:
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("invalid value for %s", fieldName)
		}
	}

	if len(partsByIndex) == 0 {
		return nil, nil
	}

	indices := make([]int, 0, len(partsByIndex))

	for index := range partsByIndex {
		indices = append(indices, index)
	}

	sort.Ints(indices)

	parts := make([]convertedPart, 0, len(indices))

	for expectedIndex, actualIndex := range indices {
		if expectedIndex != actualIndex {
			return nil, fmt.Errorf("extracted part list has a missing part")
		}

		part := partsByIndex[actualIndex]

		if part.PartNumber == "" ||
			part.PartName == "" ||
			part.MaterialGrade == "" ||
			part.GrossWeight <= 0 ||
			part.NetWeight <= 0 ||
			part.NetWeight > part.GrossWeight ||
			part.AnnualVolume <= 0 ||
			part.TargetPrice < 0 {
			return nil, fmt.Errorf(
				"part %d has missing or invalid accepted values",
				actualIndex+1,
			)
		}

		parts = append(parts, *part)
	}

	return parts, nil
}

func parseExtractionDate(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		time.RFC3339,
	}

	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported date format")
}
