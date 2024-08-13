package types

type SubscriptionRequest struct {
	Command        string               `json:"-"` // Not marshaled, manually included in the JSON array
	SubscriptionID string               `json:"-"` // Not marshaled, manually included in the JSON array
	Filters        []SubscriptionFilter `json:"filters"`
}