package state

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitRedis_Success(t *testing.T) {
	// Setup mock Redis server
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()

	// Test the function
	client, err := InitRedis(mockRedis.Addr(), "", 0)

	// Assertions
	require.NoError(t, err, "InitRedis should not return an error")
	require.NotNil(t, client, "Redis client should not be nil")

	// Test that we can actually use the client
	ctx := context.Background()
	pong := client.Ping(ctx)
	assert.NoError(t, pong.Err(), "Should be able to ping Redis")

	// Cleanup
	client.Close()
}

func TestInitRedis_WithPassword(t *testing.T) {
	// Setup mock Redis with password
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()

	password := "testpassword"
	mockRedis.RequireAuth(password)

	// Test with correct password
	client, err := InitRedis(mockRedis.Addr(), password, 0)

	require.NoError(t, err, "InitRedis should work with correct password")
	require.NotNil(t, client, "Redis client should not be nil")

	client.Close()
}

func TestInitRedis_WithWrongPassword(t *testing.T) {
	// Setup mock Redis with password
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()

	mockRedis.RequireAuth("correctPassword")

	// Test with correct password
	client, err := InitRedis(mockRedis.Addr(), "wrongpassword", 0)

	assert.Error(t, err, "InitRedis should return error with wrong password")
	assert.Nil(t, client, "Redis client should be nil on error")
	assert.Contains(t, err.Error(), "failed to connect to Redis", "Error message should be descriptive")
}

func TestInitRedis_InvalidAddress(t *testing.T) {
	// Test with invalid address
	client, err := InitRedis("invalid-address:6379", "", 0)

	assert.Error(t, err, "InitRedis should return error with invalid address")
	assert.Nil(t, client, "Redis client should be nil on error")
	assert.Contains(t, err.Error(), "failed to connect to Redis", "Error message should be descriptive")
}

func TestInitRedis_ConnectionTimeout(t *testing.T) {
	// Test with non-existent but valid-format address (will timeout)
	client, err := InitRedis("127.0.0.1:16379", "", 0) // Port that's likely not running Redis

	assert.Error(t, err, "InitRedis should return error when connection times out")
	assert.Nil(t, client, "Redis client should be nil on error")
}

func TestInitRedis_ClientConfiguration(t *testing.T) {
	// Setup mock Redis
	mockRedis := miniredis.RunT(t)
	defer mockRedis.Close()

	client, err := InitRedis(mockRedis.Addr(), "", 5) // Different DB number

	require.NoError(t, err)
	require.NotNil(t, client)

	// Check client options (this is a bit tricky to test, but we can verify it works)
	ctx := context.Background()

	// Set a key in DB 5
	err = client.Set(ctx, "testkey", "testvalue", time.Minute).Err()
	assert.NoError(t, err, "Should be able to set key in specified DB")

	// Get the key back
	val, err := client.Get(ctx, "testkey").Result()
	assert.NoError(t, err, "Should be able to get key from specified DB")
	assert.Equal(t, "testvalue", val, "Retrieved value should match set value")

	client.Close()
}

// Benchmark test (optional tapi bagus buat performance testing)
func BenchmarkInitRedis(b *testing.B) {
	mockRedis := miniredis.RunT(b)
	defer mockRedis.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := InitRedis(mockRedis.Addr(), "", 0)
		if err != nil {
			b.Fatal(err)
		}
		client.Close()
	}
}
