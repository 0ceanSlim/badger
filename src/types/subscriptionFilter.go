package types

type SubscriptionFilter struct {
	Authors []string `json:"authors,omitempty"`
	Kinds   []int    `json:"kinds,omitempty"`
	IDs     []string `json:"ids,omitempty"`
	// Additional fields can be added as needed.
}