package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitPostgres_Integration_Success(t *testing.T) {
	testDsn := "postgres://chat_admin:e55361929bc8bd1d9faa96df0a89916a@localhost:5432/chat_db_test?sslmode=disable" // testing db dsn

	db, sqlDB, err := InitPostgres(testDsn)

	require.NoError(t, err)
	require.NotNil(t, db)
	require.NotNil(t, sqlDB)

	// Test connection pool settings
	stats := sqlDB.Stats()
	assert.Equal(t, 1, stats.OpenConnections)
	assert.Equal(t, 100, stats.MaxOpenConnections)

	// Test basic functionality
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	// Clean close
	defer sqlDB.Close()
}

func TestInitPostgres_InvalidDSN(t *testing.T) {
	// test with completely invalid dsn
	invalidDSN := "invalid-dsn-format"

	db, sqlDB, err := InitPostgres(invalidDSN)

	// Assertions
	assert.Error(t, err, "InitPostgres should return error with invalid_DSN")
	assert.Nil(t, db, "GORM DB should be nil on error")
	assert.Nil(t, sqlDB, "SQL DB should be nil on error")
}

func TestInitPostgres_DatabaseConnectionFailure(t *testing.T) {
	// Test with valid dsn format but non-existent database
	nonExistentDSN := "host=nonexistent-host user=test password=test dbname=test port=5432 sslmode=disable"

	db, sqlDB, err := InitPostgres(nonExistentDSN)

	// Assertions
	assert.Error(t, err, "InitPostgres should return error when database is unreachable")
	assert.Nil(t, db, "GORM DB should be nil on error")
	assert.Nil(t, sqlDB, "SQL DB should be nil on error")

	assert.Contains(t, err.Error(), "failed to connect")
}
