package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type CollectedBadge struct {
	BadgeType  string // Badge type from the "a" tag
	AwardedBy  string // Awarding pubkey from the "a" tag
	EventID    string // Event ID from the "e" tag (badge award event)
	ThumbURL   string // Badge thumbnail URL fetched from the 30009 event
	CreatedAt  int64  // Event creation time
	Description string // Badge description fetched from the 30009 event
}

// FetchCollectedBadges fetches all badges collected by a user from their relays, filtering duplicates and ensuring the latest events are used.
func FetchCollectedBadges(publicKey string, relays []string) ([]CollectedBadge, error) {
	var collectedBadges []CollectedBadge
	latestBadges := make(map[string]CollectedBadge) // Track the latest badge per event ID
	eventOrder := []string{}                        // Track the order of "e" tags for the badges

	for _, relayURL := range relays {
		log.Printf("Connecting to WebSocket: %s\n", relayURL)
		conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
		if err != nil {
			log.Printf("Failed to connect to WebSocket: %v\n", err)
			continue
		}
		// defer conn.Close() - moved inside the loop to avoid premature closure

		filter := types.SubscriptionFilter{
			Authors: []string{publicKey},
			Kinds:   []int{30008}, // Badge receipt event
		}

		subRequest := []interface{}{
			"REQ",
			"sub2",
			filter,
		}

		requestJSON, err := json.Marshal(subRequest)
		if err != nil {
			log.Printf("Failed to marshal subscription request: %v\n", err)
			return nil, err
		}

		log.Printf("Sending subscription request: %s\n", requestJSON)

		if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
			log.Printf("Failed to send subscription request: %v\n", err)
			return nil, err
		}

		for {
			log.Println("Waiting for WebSocket message...")
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading WebSocket message: %v\n", err)
				break // Move to the next relay if there's an error
			}

			log.Printf("Received WebSocket message: %s\n", message)

			var response []interface{}
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("Failed to unmarshal response: %v\n", err)
				continue
			}

			if response[0] == "EVENT" {
				// Parse the event data
				eventData, err := json.Marshal(response[2])
				if err != nil {
					log.Printf("Failed to marshal event data: %v\n", err)
					continue
				}

				var event types.NostrEvent
				if err := json.Unmarshal(eventData, &event); err != nil {
					log.Printf("Failed to parse event data: %v\n", err)
					continue
				}

				// Check if it's a valid badge collection event
				if !containsTag(event.Tags, "d", "profile_badges") {
					continue // Ignore events that don't match the "profile_badges" identifier
				}

				// Process pairs of "a" and "e" tags
				for i := 0; i < len(event.Tags); i++ {
					tag := event.Tags[i]
					if tag[0] == "a" {
						// Ensure there's a corresponding "e" tag
						if i+1 < len(event.Tags) && event.Tags[i+1][0] == "e" {
							badgeType := tag[1]  // Badge type from "a" tag
							eventID := event.Tags[i+1][1] // Badge award event ID from "e" tag
							relay := ""          // Default relay
							if len(tag) > 2 {
								relay = tag[2] // Relay to fetch the badge definition
								log.Printf("Using relay: %s\n", relay) // Log the relay URL
							}
				
							// Fetch the 30009 event details for this badge
							badgeDetails, err := fetchBadgeDetails(relay, eventID, badgeType)
							if err != nil {
								log.Printf("Failed to fetch badge details: %v\n", err)
								continue
							}
				
							// Check if a badge with the same event ID exists, only replace if this one is newer
							if existingBadge, exists := latestBadges[eventID]; !exists || badgeDetails.CreatedAt > existingBadge.CreatedAt {
								latestBadges[eventID] = badgeDetails
							}
				
							// Maintain the order of badge events
							if !contains(eventOrder, eventID) {
								eventOrder = append(eventOrder, eventID)
							}
						}
					}
				}
			} else if response[0] == "EOSE" {
				log.Println("End of subscription signal received")
				break // No more events, move to the next relay
			}
		}
		defer conn.Close() // Moved here
	}

	// Collect the badges in the order of the event IDs from the "e" tags
	for _, eventID := range eventOrder {
		if badge, exists := latestBadges[eventID]; exists {
			collectedBadges = append(collectedBadges, badge)
		}
	}

	return collectedBadges, nil // Return all collected badges in the correct order
}

// fetchBadgeDetails fetches the 30009 badge details from the given relay and event ID
func fetchBadgeDetails(relay string, eventID string, badgeType string) (CollectedBadge, error) {
    var badge CollectedBadge
    badge.BadgeType = badgeType

    relay = strings.TrimSpace(relay)
    log.Printf("Relay URL for fetching badge details: %s\n", relay)

    if relay == "" || (!strings.HasPrefix(relay, "ws://") && !strings.HasPrefix(relay, "wss://")) {
        log.Printf("Invalid or empty relay URL: %s\n", relay)
        return badge, errors.New("invalid or empty relay URL")
    }

    conn, _, err := websocket.DefaultDialer.Dial(relay, nil)
    if err != nil {
        log.Printf("Failed to connect to relay %s: %v\n", relay, err)
        return badge, fmt.Errorf("failed to connect to relay %s: %v", relay, err)
    }
    defer conn.Close()

    filter := types.SubscriptionFilter{
        IDs:   []string{eventID},
        Kinds: []int{30009},
    }

    subRequest := []interface{}{
        "REQ",
        "sub3",
        filter,
    }

    requestJSON, err := json.Marshal(subRequest)
    if err != nil {
        return badge, fmt.Errorf("failed to marshal request: %v", err)
    }

    if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
        return badge, fmt.Errorf("failed to send subscription request: %v", err)
    }

    _, message, err := conn.ReadMessage()
    if err != nil {
        return badge, fmt.Errorf("failed to read response: %v", err)
    }

    log.Printf("Received response: %s\n", string(message))

    var response []interface{}
    if err := json.Unmarshal(message, &response); err != nil {
        return badge, fmt.Errorf("failed to unmarshal response: %v", err)
    }

    if len(response) < 3 || response[0] != "EVENT" {
        log.Printf("Unexpected response format: %v\n", response)
        return badge, errors.New("no valid EVENT found in response")
    }

    eventData, err := json.Marshal(response[2])
    if err != nil {
        return badge, fmt.Errorf("failed to marshal event data: %v", err)
    }

    var event types.NostrEvent
    if err := json.Unmarshal(eventData, &event); err != nil {
        return badge, fmt.Errorf("failed to parse event data: %v", err)
    }

    for _, tag := range event.Tags {
        log.Printf("Processing tag: %v\n", tag)
        switch tag[0] {
        case "name":
            badge.BadgeType = tag[1]
        case "description":
            badge.Description = tag[1]
        case "image":
            badge.ThumbURL = tag[1]
        }
    }

    return badge, nil
}



// Helper function to check if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, str := range slice {
		if str == item {
			return true
		}
	}
	return false
}

// Helper function to check if a tag exists with a specific key and value
func containsTag(tags [][]string, key, value string) bool {
	for _, tag := range tags {
		if tag[0] == key && tag[1] == value {
			return true
		}
	}
	return false
}
