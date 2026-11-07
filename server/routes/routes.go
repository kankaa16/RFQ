package routes

import (
	"net/http"

	"rfq_costing/server/middleware"
	"rfq_costing/server/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupRouter(db *mongo.Database, jwtSecret string) *gin.Engine {
	router := gin.Default()

	// initialize services
	authService := &services.AuthService{
		UserCollection: db.Collection("users"),
		JWTSecret:      jwtSecret,
	}

	rfqService := &services.RFQService{
		Collection: db.Collection("rfqs"),
	}

	customerService := &services.CustomerService{
		Collection: db.Collection("customers"),
	}

	partService := &services.PartService{
		Collection:    db.Collection("parts"),
		RFQCollection: db.Collection("rfqs"),
	}

	rateService := &services.RateMasterService{
		Collection: db.Collection("rate_masters"),
	}

	routingService := &services.RoutingService{
		Collection:     db.Collection("routings"),
		PartCollection: db.Collection("parts"),
	}

	costingService := &services.CostingService{
		Collection:        db.Collection("costings"),
		PartCollection:    db.Collection("parts"),
		RateCollection:    db.Collection("rate_masters"),
		RoutingCollection: db.Collection("routings"),
	}

	quoteService := &services.QuoteService{
		Collection:         db.Collection("quotes"),
		CostingCollection:  db.Collection("costings"),
		RFQCollection:      db.Collection("rfqs"),
		CustomerCollection: db.Collection("customers"),
		PartCollection:     db.Collection("parts"),
	}

	approvalService := &services.ApprovalService{
		QuoteCollection:    db.Collection("quotes"),
		ApprovalCollection: db.Collection("approvals"),
	}

	documentService := &services.DocumentService{
		Collection: db.Collection("documents"),
	}

	historicalQuoteService := &services.HistoricalQuoteService{
		Collection: db.Collection("historical_quotes"),
	}

	aiExtractionService := &services.AIExtractionService{
		Collection: db.Collection("ai_extractions"),
	}

	pdfQuoteService := &services.PDFQuoteService{
		Collection: db.Collection("quotes"),
	}

	api := router.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "rfq costing api is running",
			})
		})

		api.POST("/auth/login", authService.Login)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtSecret))

		{
			protected.GET("/profile", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"user_id": c.GetString("user_id"),
						"email":   c.GetString("email"),
						"role":    c.GetString("role"),
					},
				})
			})

			// rfq routes
			protected.POST("/rfqs", rfqService.CreateRFQ)
			protected.GET("/rfqs", rfqService.ListRFQs)

			// customer routes
			protected.POST("/customers", customerService.CreateCustomer)
			protected.GET("/customers", customerService.ListCustomers)

			// part routes
			protected.POST("/parts", partService.CreatePart)
			protected.GET("/rfqs/:id/parts", partService.ListPartsByRFQ)

			// rate master routes
			protected.POST("/rates", rateService.CreateRate)
			protected.GET("/rates", rateService.ListRates)

			// routing routes
			protected.POST("/routings", routingService.SaveRouting)
			protected.GET("/parts/:id/routing", routingService.GetRouting)

			// costing routes
			protected.POST("/costings/calculate", costingService.CalculateCosting)

			// quote routes
			protected.POST("/quotes", quoteService.CreateQuote)

			// approval routes
			protected.POST(
				"/quotes/:id/review",
				middleware.RequireRole("sales_head"),
				approvalService.ReviewQuote,
			)

			// document routes
			protected.POST("/documents/upload", documentService.UploadDocument)

			// historical quote routes
			protected.GET(
				"/historical-quotes",
				historicalQuoteService.SearchHistoricalQuotes,
			)

			// ai extraction routes
			protected.GET(
				"/ai-extractions/:document_id",
				aiExtractionService.GetExtraction,
			)

			protected.POST(
				"/ai-extractions/:id/review",
				aiExtractionService.ReviewExtraction,
			)

			// pdf quotation route
			protected.GET("/quotes/:id/pdf", pdfQuoteService.GenerateQuotePDF)
		}
	}

	return router
}
