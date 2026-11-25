package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"bansos-backend/config"
)

const (
	CollectionModuleContents = "module_contents"
	CollectionProgress       = "progress"
	CollectionReviews        = "module_reviews"
)

func NewMongoClient(ctx context.Context) (*mongo.Client, error) {
	uri := config.GetEnv("MONGO_URI", "mongodb://localhost:27017")

	opts := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(uint64(config.GetEnvInt("MONGO_MAX_POOL", 20)))

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client Mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB tidak dapat dihubungi: %w", err)
	}

	return client, nil
}
