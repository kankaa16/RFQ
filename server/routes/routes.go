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

	// initialize authentication service
	authService := &services.AuthService{
		UserCollection: db.Collection("users"),
		JWTSecret:      jwtSecret,
	}

	api := router.Group("/api")
	{
		// public routes
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "rfq costing api is running",
			})
		})

		api.POST("/auth/login", authService.Login)

		// protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(jwtSecret))

		{
			// test route to check logged-in user
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

			// sales head only
			protected.POST(
				"/quotes/:id/approve",
				middleware.RequireRole("sales_head"),
				func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{
						"success": true,
						"message": "approval access granted",
					})
				},
			)
		}
	}

	return router
}
