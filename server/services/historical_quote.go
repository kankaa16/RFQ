package services

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"rfq_costing/server/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type HistoricalQuoteService struct {
	Collection *mongo.Collection
}

func (s *HistoricalQuoteService) SearchHistoricalQuotes(c *gin.Context) {
	search := c.Query("search")
	material := c.Query("material")
	process := c.Query("process")
	customer := c.Query("customer")

	filter := bson.M{}

	if search != "" {
		pattern := bson.Regex{
			Pattern: regexp.QuoteMeta(search),
			Options: "i",
		}

		filter["$or"] = bson.A{
			bson.M{"part_name": pattern},
			bson.M{"part_number": pattern},
			bson.M{"original_quote_number": pattern},
		}
	}

	if material != "" {
		filter["material_grade"] = bson.Regex{
			Pattern: regexp.QuoteMeta(material),
			Options: "i",
		}
	}

	if process != "" {
		filter["manufacturing_process"] = bson.Regex{
			Pattern: regexp.QuoteMeta(process),
			Options: "i",
		}
	}

	if customer != "" {
		filter["customer_name"] = bson.Regex{
			Pattern: regexp.QuoteMeta(customer),
			Options: "i",
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	findOptions := options.Find().
		SetSort(bson.D{
			{Key: "quoted_at", Value: -1},
		}).
		SetLimit(100)

	cursor, err := s.Collection.Find(ctx, filter, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not search historical quotes",
		})
		return
	}
	defer cursor.Close(ctx)

	var quotes []models.HistoricalQuote

	if err := cursor.All(ctx, &quotes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "could not read historical quotes",
		})
		return
	}

	if quotes == nil {
		quotes = []models.HistoricalQuote{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(quotes),
		"data":    quotes,
	})
}
