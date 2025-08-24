package types

type RefreshSession struct {
	UserId      string `json:"userId"`
	JTI         string `json:"jti"`
	Fingerprint string `json:"fingerprint"`
	IssueAt     int64  `json:"issue_at"`
	ExpireAt    int64  `json:"expires_refresh"`
	Status      string `json:"status"`
}
