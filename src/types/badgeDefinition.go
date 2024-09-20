package types

type BadgeDefinition struct {
	NostrEvent
	Name        string
	Description string
	ImageURL    string
	ThumbURL    string
	DTag        string
}
