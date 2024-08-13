package types

type NostrEvent struct {
	Content   string     `json:"content"`
	CreatedAt int64      `json:"created_at"`
	ID        string     `json:"id"`
	Kind      int        `json:"kind"`
	PubKey    string     `json:"pubkey"`
	Sig       string     `json:"sig"`
	Tags      [][]string `json:"tags"`
}
