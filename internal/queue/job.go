package queue

import "encoding/json"

type Job struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Priority  int             `json:"priority"`
	Retry     int             `json:"retry"`
	MaxRetry  int             `json:"max_retry"`
	ErrorMsg  string          `json:"error_msg,omitempty"`
	CreatedAt int64           `json:"created_at"`
	ExpireAt  int64           `json:"expired_at"`
}

func MustMarshal(payload any) json.RawMessage {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	return b
}
