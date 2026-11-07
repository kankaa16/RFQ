package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func CreateIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := map[string][]mongo.IndexModel{
		"users": {
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"customers": {
			{
				Keys:    bson.D{{Key: "customer_code", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
		"rfqs": {
			{
				Keys:    bson.D{{Key: "rfq_number", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{{Key: "due_date", Value: 1}},
			},
		},
		"rate_masters": {
			{
				Keys: bson.D{
					{Key: "rate_code", Value: 1},
					{Key: "version", Value: 1},
				},
				Options: options.Index().SetUnique(true),
			},
		},
		"parts": {
			{
				Keys: bson.D{{Key: "rfq_id", Value: 1}},
			},
		},
		"routings": {
			{
				Keys: bson.D{{Key: "part_id", Value: 1}},
			},
		},
		"costings": {
			{
				Keys: bson.D{{Key: "part_id", Value: 1}},
			},
		},
		"quotes": {
			{
				Keys:    bson.D{{Key: "quote_number", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{{Key: "rfq_id", Value: 1}},
			},
		},
		"documents": {
			{
				Keys: bson.D{{Key: "rfq_id", Value: 1}},
			},
		},
		"historical_quotes": {
			{
				Keys: bson.D{{Key: "part_number", Value: 1}},
			},
			{
				Keys: bson.D{{Key: "quoted_at", Value: -1}},
			},
		},
		"ai_extractions": {
			{
				Keys:    bson.D{{Key: "document_id", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
		},
	}

	for collectionName, models := range indexes {
		_, err := db.Collection(collectionName).Indexes().CreateMany(ctx, models)
		if err != nil {
			return err
		}
	}

	return nil
}
