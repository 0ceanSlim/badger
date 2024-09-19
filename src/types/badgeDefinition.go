package types

type BadgeDefinition struct {
	//add pubkey to complete struct
	Name        string
	Description string
	ImageURL    string
	ThumbURL    string
	EventID     string // Track event ID to avoid duplicates
	DTag        string // Add DTag field for the unique identifier
}
