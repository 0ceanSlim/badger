package utils

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"

	"badger/src/types"
)

func FetchUserMetadata(publicKey string) (*types.NostrEvent, error) {
	url := "wss://purplepag.es" // Replace with actual relay URL

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Failed to connect to WebSocket: %v\n", err)
		return nil, err
	}
	defer conn.Close()

	// Subscribe to user's kind:0 events
	subRequest := types.SubscriptionRequest{
		Req:     "REQ",
		SubID:   "sub1",
		Authors: []string{publicKey},
		Kinds:   []int{0},
	}

	if err := conn.WriteJSON(subRequest); err != nil {
		log.Printf("Failed to send subscription request: %v\n", err)
		return nil, err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v\n", err)
			return nil, err
		}

		var response []interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("Failed to unmarshal response: %v\n", err)
			continue
		}

		if response[0] == "EVENT" {
			var event types.NostrEvent
			eventData, _ := json.Marshal(response[1])
			if err := json.Unmarshal(eventData, &event); err != nil {
				log.Printf("Failed to parse event data: %v\n", err)
				continue
			}

			if event.Kind == 0 {
				return &event, nil
			}
		}
	}
}
