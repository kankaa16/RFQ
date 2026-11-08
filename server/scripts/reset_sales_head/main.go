package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"rfq_costing/server/config"
	"rfq_costing/server/database"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.LoadConfig()

	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_EMAIL")))
	password := os.Getenv("SEED_PASSWORD")

	if email == "" || len(password) < 8 {
		log.Fatal("valid email and password are required")
	}

	client, err := database.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("mongodb connection failed: ", err)
	}
	defer client.Disconnect(context.Background())

	collection := client.Database(cfg.MongoDatabase).Collection("users")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatal("password hashing failed: ", err)
	}

	result, err := collection.UpdateOne(
		ctx,
		bson.M{
			"email": email,
			"role":  "sales_head",
		},
		bson.M{
			"$set": bson.M{
				"password_hash": string(hashedPassword),
			},
		},
	)

	if err != nil {
		log.Fatal("password reset failed: ", err)
	}

	if result.MatchedCount == 0 {
		log.Fatal("sales head account not found")
	}

	if result.ModifiedCount == 0 {
		log.Println("password hash was unchanged")
		return
	}

	log.Println("sales head password reset successfully")
}
