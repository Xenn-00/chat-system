package state

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

type JwtSecret struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

type AppState struct {
	Ctx       context.Context
	Cancel    context.CancelFunc
	DB        *gorm.DB
	Redis     *redis.Client
	Mongo     *mongo.Client
	JwtSecret *JwtSecret
}

func InitAppState(ctx context.Context, cancel context.CancelFunc) (*AppState, error) {
	dbUrl := config.Conf.DATABASE.Postgres.DSN
	rAddr := config.Conf.DATABASE.Redis.Addr
	rPass := config.Conf.DATABASE.Redis.Password

	db, _, err := InitPostgres(dbUrl)
	if err != nil {
		return nil, err
	}

	mongoClient, err := InitMongo(context.Background())
	if err != nil {
		return nil, err
	}

	rdb, err := InitRedis(rAddr, rPass, 0)
	if err != nil {
		return nil, err
	}

	jwtSecret, err := InitSecret()
	if err != nil {
		return nil, err
	}

	return &AppState{
		Ctx:       ctx,
		Cancel:    cancel,
		DB:        db,
		Mongo:     mongoClient,
		Redis:     rdb,
		JwtSecret: jwtSecret,
	}, nil
}

func (a *AppState) Close() {
	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err == nil {
			log.Info().Msg("Closing PostgreSQL database connection...")
			sqlDB.Close()
		}
	}

	if a.Mongo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		log.Info().Msg("Closing MongoDB client...")
		defer cancel()
		if err := a.Mongo.Disconnect(ctx); err != nil {
			log.Error().Err(err).Msg("failed to disconnect MongoDB client")
		}
	}

	if a.Redis != nil {
		log.Info().Msg("Closing Redis client...")
		if err := a.Redis.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close Redis client")
		}
	}
}
