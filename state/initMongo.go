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
			"required": []string{"room_id", "sender_id", "receiver_id", "content", "is_read", "created_at"},
			"properties": bson.M{
				"room_id": bson.M{
					"bsonType":    "string",
					"description": "Room ID must be a string",
				},
				"sender_id": bson.M{
					"bsonType":    "string",
					"description": "ID sender user",
				},
				"receiver_id": bson.M{
					"bsonType":    "string",
					"description": "ID receiver user",
				},
				"content": bson.M{
					"bsonType":    "string",
					"description": "Chat's content",
				},
				"is_read": bson.M{
					"bsonType":    "bool",
					"description": "Is message already read",
				},
				"reply_to": bson.M{
					"bsonType": []string{"object", "null"},
					"required": []string{"message_id", "content", "sender_id"},
					"properties": bson.M{
						"message_id": bson.M{
							"bsonType":    "objectId",
							"description": "ID of the replied message",
						},
						"content": bson.M{
							"bsonType":    "string",
							"description": "Content of the replied message",
						},
						"sender_id": bson.M{
							"bsonType":    "string",
							"description": "Sender of the original message",
						},
					},
				},
				"attachments": bson.M{
					"bsonType": []string{"array", "null"},
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"url", "type"},
						"properties": bson.M{
							"type": bson.M{
								"bsonType":    "string",
								"description": "File type, e.g., image, video, docs, etc.",
							},
							"url": bson.M{
								"bsonType":    "string",
								"description": "File storage URL",
							},
						},
					},
				},
				"is_edited": bson.M{
					"bsonType":    []string{"bool", "null"},
					"description": "Whether the message has been edited",
				},
				"created_at": bson.M{
					"bsonType":    "date",
					"description": "Message creation timestamp",
				},
				"updated_at": bson.M{
					"bsonType":    []string{"date", "null"},
					"description": "Message last update timestamp",
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
			{
				Keys:    bson.D{{Key: "receiver_id", Value: 1}},
				Options: options.Index().SetName("receiver_idx"),
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
