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
	BadgeType   string // Badge type from the "a" tag
	AwardedBy   string // Awarding pubkey from the "a" tag
	EventID     string // Event ID from the "e" tag (badge award event)
	ThumbURL    string // Badge thumbnail URL fetched from the 30009 event
	CreatedAt   int64  // Event creation time
	Description string // Badge description fetched from the 30009 event
}

// FetchCollectedBadges fetches badges from multiple relays.
func FetchCollectedBadges(publicKey string, relays []string) ([]CollectedBadge, error) {
	var collectedBadges []CollectedBadge
	latestBadges := make(map[string]CollectedBadge)
	eventOrder := []string{}

	for _, relayURL := range relays {
		log.Printf("Connecting to WebSocket: %s\n", relayURL)
		conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
		if err != nil {
			log.Printf("Failed to connect to WebSocket: %v\n", err)
			continue
		}

		filter := types.SubscriptionFilter{
			Authors: []string{publicKey},
			Kinds:   []int{30008}, // Badge receipt events
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

		if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
			log.Printf("Failed to send subscription request: %v\n", err)
			return nil, err
		}

		for {
			log.Println("Waiting for WebSocket message...")
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading WebSocket message: %v\n", err)
				break
			}

			log.Printf("Received WebSocket message: %s\n", message)

			var response []interface{}
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("Failed to unmarshal response: %v\n", err)
				continue
			}

			if response[0] == "EVENT" {
				// Process the event message
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

				// Process badge events with "profile_badges" tag
				if !containsTag(event.Tags, "d", "profile_badges") {
					continue
				}

				// Process pairs of "a" and "e" tags
				for i := 0; i < len(event.Tags); i++ {
					tag := event.Tags[i]
					if tag[0] == "a" {
						// Ensure the next tag is "e"
						if i+1 < len(event.Tags) && event.Tags[i+1][0] == "e" {
							badgeType := tag[1]
							eventID := event.Tags[i+1][1]
							relay := ""

							if len(tag) > 2 {
								relay = tag[2]
								log.Printf("Using relay: %s\n", relay)
							}

							if relay == "" {
								relay = relayURL // Fallback to the current relay
								log.Printf("Fallback relay URL used: %s\n", relay)
							}

							// Fetch badge details
							badgeDetails, err := fetchBadgeDetails(relay, eventID, badgeType)
							if err != nil {
								log.Printf("Failed to fetch badge details: %v\n", err)
								continue
							}

							// Update the latest badge
							if existingBadge, exists := latestBadges[eventID]; !exists || badgeDetails.CreatedAt > existingBadge.CreatedAt {
								latestBadges[eventID] = badgeDetails
							}

							if !contains(eventOrder, eventID) {
								eventOrder = append(eventOrder, eventID)
							}
						}
					}
				}
			} else if response[0] == "EOSE" {
				// End of Stored Events - finish processing this relay
				log.Println("End of subscription signal received")
				break
			}
		}
		defer conn.Close()
	}

	// Collect badges in order of their event IDs
	for _, eventID := range eventOrder {
		if badge, exists := latestBadges[eventID]; exists {
			collectedBadges = append(collectedBadges, badge)
		}
	}

	return collectedBadges, nil
}

// fetchBadgeDetails retrieves badge details using the relay and event ID
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
		Kinds: []int{30009}, // Badge definition event
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

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return badge, fmt.Errorf("failed to read response: %v", err)
		}

		log.Printf("Received response: %s\n", string(message))

		var response []interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			return badge, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		if response[0] == "EVENT" {
			eventData, err := json.Marshal(response[2])
			if err != nil {
				return badge, fmt.Errorf("failed to marshal event data: %v", err)
			}

			var event types.NostrEvent
			if err := json.Unmarshal(eventData, &event); err != nil {
				return badge, fmt.Errorf("failed to parse event data: %v", err)
			}

			// Extract badge details from the event tags
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
			return badge, nil // Return badge after successful parsing
		} else if response[0] == "EOSE" {
			// End of Stored Events, no valid event found
			log.Printf("End of subscription received while fetching badge details")
			break
		}
	}

	return badge, errors.New("no valid badge event found")
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

