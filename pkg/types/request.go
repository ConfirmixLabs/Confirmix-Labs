package types

// SignedRequest represents a request signed by an admin
type SignedRequest struct {
	Action       string            `json:"action"`
	Data         map[string]string `json:"data"`
	AdminAddress string            `json:"adminAddress"`
	Signature    string            `json:"signature"`
	Timestamp    int64             `json:"timestamp"`
} 