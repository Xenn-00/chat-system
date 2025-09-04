package types

import "time"

type DLQRetryConfig struct {
	BatchSize      int           `json:"batch_size"`
	RetryInterval  time.Duration `json:"retry_interval"`
	MaxRetryCount  int           `json:"max_retry_count"`
	BackoffFactor  float64       `json:"backoff_factor"`
	DatabaseName   string        `json:"database_name"`
	CollectionName string        `json:"collection_name"`
}
