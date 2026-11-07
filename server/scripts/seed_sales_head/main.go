package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"rfq_costing/server/config"
	"rfq_costing/server/database"
	"rfq_costing/server/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.MongoURI == "" {
		log.Fatal("MONGO_URI is missing from .env")
	}

	// read user details from environment
	name := strings.TrimSpace(os.Getenv("SEED_NAME"))
	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_EMAIL")))
	password := os.Getenv("SEED_PASSWORD")
	role := strings.TrimSpace(os.Getenv("SEED_ROLE"))

	// validate input
	if name == "" || email == "" || len(password) < 8 {
		log.Fatal("valid name, email and password are required")
	}

	if role != models.RoleEstimator && role != models.RoleSalesHead {
		log.Fatal("role must be estimator or sales_head")
	}

	// connect to mongodb
	client, err := database.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongodb connection failed: ", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(cfg.MongoDatabase).Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// check whether user already exists
	count, err := collection.CountDocuments(ctx, bson.M{
		"email": email,
	})

	if err != nil {
		log.Fatal("could not check existing user: ", err)
	}

	if count > 0 {
		log.Fatal("user with this email already exists")
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		log.Fatal("password hashing failed: ", err)
	}

	// create user
	user := models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		CreatedAt:    time.Now(),
	}

	_, err = collection.InsertOne(ctx, user)

	if err != nil {
		log.Fatal("user creation failed: ", err)
	}

	fmt.Println("user created successfully")
	fmt.Println("name:", name)
	fmt.Println("email:", email)
	fmt.Println("role:", role)
}
