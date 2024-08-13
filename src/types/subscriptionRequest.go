package types

type SubscriptionRequest struct {
	Req     string   `json:"REQ"`
	SubID   string   `json:"subscription_id"`
	Authors []string `json:"authors,omitempty"`
	Kinds   []int    `json:"kinds,omitempty"`
}
