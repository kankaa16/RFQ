package main

import (
	"context"
	"log"
	"time"

	"rfq_costing/server/config"
	"rfq_costing/server/database"
	"rfq_costing/server/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// load configuration
	cfg := config.LoadConfig()

	// validate environment variables
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is missing from .env")
	}

	if cfg.MongoURI == "" {
		log.Fatal("MONGO_URI is missing from .env")
	}

	if cfg.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY is missing from .env")
	}

	if cfg.GeminiModel == "" {
		log.Fatal("GEMINI_MODEL is missing from .env")
	}

	gin.SetMode(cfg.GinMode)

	// connect to mongodb
	mongoClient, err := database.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongo connection failed: ", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Println("mongo disconnect failed:", err)
		}
	}()

	log.Println("mongo connected successfully")

	// select database
	db := mongoClient.Database(cfg.MongoDatabase)

	// create database indexes
	if err := database.CreateIndexes(db); err != nil {
		log.Fatal("database index creation failed: ", err)
	}

	log.Println("database indexes created successfully")

	// initialize routes
	router := routes.SetupRouter(
		db,
		cfg.JWTSecret,
		cfg.GeminiAPIKey,
		cfg.GeminiModel,
	)

	log.Printf("server working at http://localhost:%s", cfg.Port)

	// start server
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("server failed: ", err)
	}
}
