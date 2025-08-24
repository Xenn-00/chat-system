package state

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/config"
	"go.mongodb.org/mongo-driver/v2/bson"
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

	db := client.Database("chat_collection")
	// init message collections
	if err := initMessageCollection(ctx, db); err != nil {
		return nil, fmt.Errorf("faield to create 'messages' collection: %w", err)
	}

	log.Info().Msg("MongoDB connection established successfully")
	return client, nil
}

func initMessageCollection(ctx context.Context, db *mongo.Database) error {
	collectionName := "messages"

	// check, is it already set or not
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": collectionName})
	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	if len(collections) == 0 {
		// create collection with optional JSON schema validator
		jsonSchema := bson.M{
			"bsonType": "object",
			"required": []string{"room_id", "sender_id", "content", "created_at"},
			"properties": bson.M{
				"room_id": bson.M{
					"bsonType":    "string",
					"description": "Room ID must be a string",
				},
				"sender_id": bson.M{
					"bsonType":    "string",
					"description": "ID sender user",
				},
				"content": bson.M{
					"bsonType":    "string",
					"description": "Chat's content",
				},
				"created_at": bson.M{
					"bsonType":    "date",
					"description": "Date send",
				},
			},
		}
		opts := options.CreateCollection().SetValidator(bson.M{
			"$jsonSchema": jsonSchema,
		})

		if err := db.CreateCollection(ctx, collectionName, opts); err != nil {
			return fmt.Errorf("faield to create collection: %w", err)
		}

		log.Info().Msg("Collection 'messages' created!")

		// add index (room_id + created_at) for easily query and sort
		indexes := db.Collection(collectionName).Indexes()
		_, err := indexes.CreateMany(ctx, []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "room_id", Value: 1}, {Key: "created_at", Value: -1}},
				Options: options.Index().SetName("room_time_idx"),
			},
			{
				Keys:    bson.D{{Key: "sender_id", Value: 1}},
				Options: options.Index().SetName("sender_idx"),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}

		log.Info().Msg("Indexes created for 'messages'")
	}

	log.Info().Msg("Already have 'messages' collection!")
	return nil
}
