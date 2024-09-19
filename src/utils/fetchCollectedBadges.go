package utils

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type CollectedBadge struct {
	BadgeType   string // Badge type from the "a" tag
	AwardedBy   string // Awarding pubkey from the "a" tag
	EventID     string // Event ID from the "e" tag (badge award event)
	Name        string // Badge name from the 30009 event
	Description string // Badge description from the 30009 event
	ImageURL    string // Full-size badge image URL fetched from the 30009 event
	ThumbURL    string // Badge thumbnail URL fetched from the 30009 event
	CreatedAt   int64  // Event creation time
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
		defer conn.Close()

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

							// Fetch badge details from the relay
							badgeDetails, err := fetchBadgeDetails(relay, eventID, badgeType)
							if err != nil {
								log.Printf("Failed to fetch badge details: %v\n", err)
								continue
							}

							// Update the latest badge if newer
							if existingBadge, exists := latestBadges[eventID]; !exists || badgeDetails.CreatedAt > existingBadge.CreatedAt {
								latestBadges[eventID] = badgeDetails
							}

							// Maintain order of events
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
	}

	// Collect badges in order of their event IDs
	for _, eventID := range eventOrder {
		if badge, exists := latestBadges[eventID]; exists {
			collectedBadges = append(collectedBadges, badge)
		}
	}

	return collectedBadges, nil
}

// fetchBadgeDetails fetches the badge definition details from the relay.
func fetchBadgeDetails(relayURL, eventID, badgeType string) (CollectedBadge, error) {
	conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		log.Printf("Failed to connect to WebSocket for badge details: %v\n", err)
		return CollectedBadge{}, err
	}
	defer conn.Close()

	// Badge type sometimes has a format like `30009:xxx:badgeName`. Only extract the event ID (the middle part)
	parts := strings.Split(badgeType, ":")
	if len(parts) < 2 {
		log.Printf("Invalid badgeType format: %s\n", badgeType)
		return CollectedBadge{}, errors.New("invalid badge type format")
	}
	badgeEventID := parts[1]

	// Subscription filter for badge definition (kind 30009)
	filter := types.SubscriptionFilter{
		Kinds:   []int{30009},
		Authors: []string{badgeEventID},
	}

	subRequest := []interface{}{
		"REQ",
		"sub3",
		filter,
	}

	requestJSON, err := json.Marshal(subRequest)
	if err != nil {
		log.Printf("Failed to marshal subscription request: %v\n", err)
		return CollectedBadge{}, err
	}

	if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
		log.Printf("Failed to send subscription request for badge details: %v\n", err)
		return CollectedBadge{}, err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v\n", err)
			return CollectedBadge{}, err
		}

		log.Printf("Received WebSocket message for badge details: %s\n", message)

		var response []interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("Failed to unmarshal badge details response: %v\n", err)
			continue
		}

		// Handle NOTICE messages indicating bad request
		if response[0] == "NOTICE" {
			log.Printf("NOTICE from WebSocket: %v\n", response)
			return CollectedBadge{}, errors.New("error fetching badge details: " + response[1].(string))
		}

		if response[0] == "EVENT" {
			eventData, err := json.Marshal(response[2])
			if err != nil {
				log.Printf("Failed to marshal badge details event data: %v\n", err)
				continue
			}

			var event types.NostrEvent
			if err := json.Unmarshal(eventData, &event); err != nil {
				log.Printf("Failed to parse badge details event data: %v\n", err)
				continue
			}

			// Parse badge details from the event
			var badge CollectedBadge
			badge.BadgeType = badgeType
			badge.EventID = eventID
			badge.CreatedAt = event.CreatedAt

			for _, tag := range event.Tags {
				switch tag[0] {
				case "name":
					badge.Name = tag[1]
				case "description":
					badge.Description = tag[1]
				case "image":
					badge.ImageURL = tag[1]
				case "thumb":
					badge.ThumbURL = tag[1]
				}
			}

			return badge, nil
		} else if response[0] == "EOSE" {
			break
		}
	}

	return CollectedBadge{}, errors.New("badge details not found")
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
