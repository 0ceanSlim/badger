package types

type SubscriptionFilter struct {
	Authors []string `json:"authors,omitempty"`
	Kinds   []int    `json:"kinds,omitempty"`
	// Additional fields can be added as needed.
}