package utils

import (
	"encoding/json"
	"log"
	"strings"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type CollectedBadge struct {
    BadgeType  string // Name or type of the badge
    AwardedBy  string // Who awarded the badge
    EventID    string
    ThumbURL   string // Add this if it exists
}

// FetchCollectedBadges fetches all badges collected by a user from their relays, filtering duplicates
func FetchCollectedBadges(publicKey string, relays []string) ([]CollectedBadge, error) {
	var badges []CollectedBadge
	seenEventIDs := make(map[string]bool)

	for _, url := range relays {
		log.Printf("Connecting to WebSocket: %s\n", url)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			log.Printf("Failed to connect to WebSocket: %v\n", err)
			continue
		}
		defer conn.Close()

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

				// Check if we've already seen this event ID
				if seenEventIDs[event.ID] {
					log.Printf("Duplicate event ID found: %s, skipping...", event.ID)
					continue
				}
				seenEventIDs[event.ID] = true

				// Parse collected badge data
				var badge CollectedBadge
				badge.EventID = event.ID
				for _, tag := range event.Tags {
					switch tag[0] {
					case "a":
						// "a" tag has the badge type and awarding user's pubkey (e.g., "30009:alice:bravery")
						badgeParts := strings.Split(tag[1], ":")
						if len(badgeParts) >= 3 {
							badge.AwardedBy = badgeParts[1]
							badge.BadgeType = badgeParts[2]
						}
					}
				}
				badges = append(badges, badge)
			} else if response[0] == "EOSE" {
				log.Println("End of subscription signal received")
				break // No more events, move to the next relay
			}
		}
	}

	return badges, nil // Return all collected badges
}
