package utils

import (
	"encoding/json"
	"log"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type NostrContent struct {
	DisplayName string `json:"display_name"`
	Picture     string `json:"picture"`
	About       string `json:"about"`
}

func FetchUserMetadata(publicKey string, relays []string) (*NostrContent, error) {
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
			Kinds:   []int{0},
		}

		subRequest := []interface{}{
			"REQ",
			"sub1",
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
				// The third element in the array is the actual event data
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

				log.Printf("Received Nostr event: %+v\n", event)

				var content NostrContent
				// Now parse the Content field, which is a JSON string
				if err := json.Unmarshal([]byte(event.Content), &content); err != nil {
					log.Printf("Failed to parse content JSON: %v\n", err)
					continue
				}
				return &content, nil
			} else if response[0] == "EOSE" {
				log.Println("End of subscription signal received")
				break // No more events, move to the next relay
			}
		}
	}
	return nil, nil // Return nil if no metadata was found
}
