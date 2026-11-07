package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func Connect(uri string) (*mongo.Client, error) {

	//created mongo client
	client, err := mongo.Connect(
		options.Client().ApplyURI(uri),
	)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()
	//verify mongo conncetion
	err = client.Ping(ctx, readpref.Primary())

	if err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return client, nil
}
