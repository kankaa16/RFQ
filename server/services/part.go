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

type PartService struct {
	Collection    *mongo.Collection
	RFQCollection *mongo.Collection
}

type CreatePartRequest struct {
	RFQID                string  `json:"rfq_id"`
	PartNumber           string  `json:"part_number"`
	PartName             string  `json:"part_name"`
	DrawingNumber        string  `json:"drawing_number"`
	DrawingRevision      string  `json:"drawing_revision"`
	MaterialGrade        string  `json:"material_grade"`
	ManufacturingProcess string  `json:"manufacturing_process"`
	GrossWeight          float64 `json:"gross_weight"`
	NetWeight            float64 `json:"net_weight"`
	AnnualVolume         int64   `json:"annual_volume"`
	CustomerTargetPrice  float64 `json:"customer_target_price"`
}

func (s *PartService) CreatePart(c *gin.Context) {
	var request CreatePartRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request data",
		})
		return
	}

	request.PartNumber = strings.TrimSpace(request.PartNumber)
	request.PartName = strings.TrimSpace(request.PartName)

	if request.PartNumber == "" || request.PartName == "" || request.MaterialGrade == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "part number, part name and material grade are required",
		})
		return
	}

	if request.GrossWeight <= 0 || request.NetWeight <= 0 ||
		request.NetWeight > request.GrossWeight || request.AnnualVolume <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid part weight or annual volume",
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var rfq models.RFQ

	err = s.RFQCollection.FindOne(ctx, bson.M{
		"_id": rfqID,
	}).Decode(&rfq)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "rfq not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not verify rfq",
		})
		return
	}

	now := time.Now()

	part := models.Part{
		RFQID:                rfqID,
		PartNumber:           request.PartNumber,
		PartName:             request.PartName,
		DrawingNumber:        request.DrawingNumber,
		DrawingRevision:      request.DrawingRevision,
		MaterialGrade:        request.MaterialGrade,
		ManufacturingProcess: request.ManufacturingProcess,
		GrossWeight:          request.GrossWeight,
		NetWeight:            request.NetWeight,
		ScrapWeight:          request.GrossWeight - request.NetWeight,
		AnnualVolume:         request.AnnualVolume,
		MonthlyVolume:        request.AnnualVolume / 12,
		CustomerTargetPrice:  request.CustomerTargetPrice,
		MaterialConfirmed:    true,
		DrawingVerified:      false,
	}

	part.CreatedAt = now
	part.UpdatedAt = now

	result, err := s.Collection.InsertOne(ctx, part)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not create part",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "part created successfully",
		"data": gin.H{
			"id":          result.InsertedID,
			"part_number": part.PartNumber,
			"part_name":   part.PartName,
		},
	})
}

func (s *PartService) ListPartsByRFQ(c *gin.Context) {
	rfqID, err := bson.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid rfq id",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cursor, err := s.Collection.Find(ctx, bson.M{
		"rfq_id": rfqID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not fetch parts",
		})
		return
	}
	defer cursor.Close(ctx)

	var parts []models.Part

	if err := cursor.All(ctx, &parts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read parts",
		})
		return
	}

	if parts == nil {
		parts = []models.Part{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(parts),
		"data":    parts,
	})
}
