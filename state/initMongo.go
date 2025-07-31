package state

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func InitMongo(ctx context.Context) (*mongo.Client, error) {
	uri := config.Conf.DATABASE.Mongo.Url
	if uri == "" {
		return nil, fmt.Errorf("mongo url is empty") // No MongoDB URI provided, skip initialization
	}

	log.Info().Msgf("Connecting to MongoDB at %s", uri)

	clientOpts := options.Client().ApplyURI(uri)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Info().Msg("MongoDB connection established successfully")
	return client, nil
}