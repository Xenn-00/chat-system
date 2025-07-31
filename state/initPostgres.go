package state

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPostgres(dsn string) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Error().Msg(fmt.Errorf("failed to connect to database: %w", err).Error())
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error().Msg(fmt.Errorf("failed to get underlying sql.DB: %w", err).Error())
		return nil, nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxIdleTime(300 * time.Second)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	log.Info().Msg("Postgres database connection established successfully")
	return db, sqlDB, nil
}