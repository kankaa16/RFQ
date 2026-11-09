package routes

import (
	"net/http"

	"rfq_costing/server/middleware"
	"rfq_costing/server/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func SetupRouter(
	db *mongo.Database,
	jwtSecret string,
	geminiAPIKey string,
	geminiModel string,
) *gin.Engine {
	router := gin.Default()

	// configure cors for local and deployed react frontends
	allowedOrigins := map[string]bool{
		"http://localhost:5173":                 true,
		"https://client-rho-sage-12.vercel.app": true,
	}

	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)
			c.Writer.Header().Set(
				"Access-Control-Allow-Headers",
				"Origin, Content-Type, Accept, Authorization",
			)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// initialize services
	authService := &services.AuthService{
		UserCollection: db.Collection("users"),
		JWTSecret:      jwtSecret,
	}

	rfqService := &services.RFQService{
		Collection: db.Collection("rfqs"),
	}

	rfqStatusService := &services.RFQStatusService{
		Collection: db.Collection("rfqs"),
	}

	customerService := &services.CustomerService{
		Collection: db.Collection("customers"),
	}

	partService := &services.PartService{
		Collection:    db.Collection("parts"),
		RFQCollection: db.Collection("rfqs"),
	}

	rateMasterService := &services.RateMasterService{
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
		Collection:         db.Collection("ai_extractions"),
		DocumentCollection: db.Collection("documents"),
		RFQCollection:      db.Collection("rfqs"),
		CustomerCollection: db.Collection("customers"),
		PartCollection:     db.Collection("parts"),
	}

	geminiExtractionService := &services.GeminiExtractionService{
		DocumentCollection:   db.Collection("documents"),
		ExtractionCollection: db.Collection("ai_extractions"),
		Gemini: services.NewGeminiService(
			geminiAPIKey,
			geminiModel,
		),
	}

	pdfQuoteService := &services.PDFQuoteService{
		Collection: db.Collection("quotes"),
	}

	// public routes
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "rfq costing api is running",
		})
	})

	router.POST("/api/auth/login", authService.Login)

	// protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(jwtSecret))

	// profile
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

	// customer routes
	protected.POST("/customers", customerService.CreateCustomer)
	protected.GET("/customers", customerService.ListCustomers)

	// rfq routes
	protected.POST("/rfqs", rfqService.CreateRFQ)
	protected.GET("/rfqs", rfqService.ListRFQs)
	protected.PATCH("/rfqs/:rfq_id/status", rfqStatusService.UpdateStatus)

	// part routes
	protected.POST("/parts", partService.CreatePart)
	protected.GET("/rfqs/:rfq_id/parts", partService.ListPartsByRFQ)

	// rate routes
	protected.POST("/rates", rateMasterService.CreateRate)
	protected.GET("/rates", rateMasterService.ListRates)
	protected.GET("/rates/:rate_code/history", rateMasterService.GetRateHistory)

	// routing routes
	protected.POST("/parts/:part_id/routing", routingService.SaveRouting)
	protected.GET("/parts/:part_id/routing", routingService.GetRouting)

	// costing routes
	protected.POST("/costings/calculate", costingService.CalculateCosting)

	// quote routes
	protected.POST("/quotes", quoteService.CreateQuote)
	protected.GET("/quotes", quoteService.ListQuotes)

	// quote approval routes
	protected.GET("/approvals", approvalService.ListApprovals)
	protected.POST("/quotes/:id/review", approvalService.ReviewQuote)

	// pdf quote route
	protected.GET("/quotes/:id/pdf", pdfQuoteService.GenerateQuotePDF)

	// document routes
	protected.POST("/documents/upload", documentService.UploadDocument)

	// historical quote routes
	protected.POST(
		"/historical-quotes/import",
		historicalQuoteService.ImportHistoricalQuotes,
	)
	protected.GET(
		"/historical-quotes",
		historicalQuoteService.SearchHistoricalQuotes,
	)

	// ai extraction routes
	protected.POST(
		"/ai-extractions/:document_id/extract",
		geminiExtractionService.ExtractDocument,
	)

	protected.GET(
		"/ai-extractions/:document_id",
		aiExtractionService.GetExtraction,
	)

	protected.POST(
		"/extractions/:extraction_id/review",
		aiExtractionService.ReviewExtraction,
	)

	protected.POST(
		"/extractions/:extraction_id/convert",
		aiExtractionService.ConvertToRFQ,
	)

	return router
}
