package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// CreateDeleteEvent constructs a Nostr delete event (kind 5)
func CreateDeleteEvent(pubkey string, badgeID string, privateKey string) (*nostr.Event, error) {
	// Create a Nostr event struct
	event := &nostr.Event{
		PubKey:    pubkey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      5, // Kind 5 is the deletion event
		Tags: nostr.Tags{
			[]string{"e", badgeID}, // Deleting the badge event
		},
		Content: "Badge deleted by user",
	}

	// Serialize and hash the event to generate the ID
	event.ID = generateEventID(event)

	// Sign the event using the user's private key
	err := event.Sign(privateKey)
	if err != nil {
		return nil, err
	}

	return event, nil
}

// GenerateEventID computes the SHA256 hash of the serialized event data
func generateEventID(event *nostr.Event) string {
	eventSerialized := event.Serialize() // Use Nostr library serialization method
	hash := sha256.Sum256([]byte(eventSerialized))
	return hex.EncodeToString(hash[:])
}
