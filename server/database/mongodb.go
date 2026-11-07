package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func Connect(uri string) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(20 * time.Second).
		SetConnectTimeout(15 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	// verify connection with the primary node
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return client, nil
}
