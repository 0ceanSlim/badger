package utils

import (
	"encoding/json"
	"log"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type RelayList struct {
	Read  []string
	Write []string
	Both  []string
}

func FetchUserRelays(publicKey string) (*RelayList, error) {
	url := "wss://purplepag.es" // Replace with actual relay URL

	log.Printf("Connecting to WebSocket: %s\n", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("Failed to connect to WebSocket: %v\n", err)
		return nil, err
	}
	defer conn.Close()

	filter := types.SubscriptionFilter{
		Authors: []string{publicKey},
		Kinds:   []int{10002}, // Kind 10002 corresponds to relay list (NIP-65)
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
			return nil, err
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

			relayList := &RelayList{}

			for _, tag := range event.Tags {
				if len(tag) > 1 && tag[0] == "r" {
					relayURL := tag[1]
					if len(tag) == 3 {
						switch tag[2] {
						case "read":
							relayList.Read = append(relayList.Read, relayURL)
						case "write":
							relayList.Write = append(relayList.Write, relayURL)
						}
					} else {
						relayList.Both = append(relayList.Both, relayURL)
					}
				}
			}
			return relayList, nil
		}
	}
}
