package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/net/websocket"
)

var relayURLs = []string{
	"wss://offchain.pub",
	"wss://nos.lol",
	"wss://relay.damus.io",
	"wss://relay.mostr.pub",
	"wss://nostr.mom",
	"wss://relay.primal.net",
}

func CreateBadgeHandler(w http.ResponseWriter, r *http.Request) {
	var event nostr.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sendEventToRelays(event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "badge sent"})
}

func sendEventToRelays(event nostr.Event) {
	for _, relayURL := range relayURLs {
		go func(relayURL string) {
			ws, err := websocket.Dial(relayURL, "", "http://localhost/")
			if err != nil {
				log.Printf("Failed to connect to relay %s: %v", relayURL, err)
				return
			}
			defer ws.Close()

			// Prepare the event message
			eventMessage := []interface{}{"EVENT", event}

			// Marshal the event message to JSON
			eventMessageJSON, err := json.Marshal(eventMessage)
			if err != nil {
				log.Printf("Failed to marshal event message: %v", err)
				return
			}

			// Send the event message
			err = websocket.Message.Send(ws, string(eventMessageJSON))
			if err != nil {
				log.Printf("Failed to send event to relay %s: %v", relayURL, err)
				return
			}

			// Wait for a response from the relay
			var response string
			err = websocket.Message.Receive(ws, &response)
			if err != nil {
				log.Printf("Failed to receive response from relay %s: %v", relayURL, err)
				return
			}

			// Parse the response
			var responseArray []interface{}
			err = json.Unmarshal([]byte(response), &responseArray)
			if err != nil {
				log.Printf("Failed to parse response from relay %s: %v", relayURL, err)
				return
			}

			// Check if the response is valid and handle it
			if len(responseArray) >= 4 && responseArray[0] == "OK" {
				eventID := responseArray[1].(string)
				success := responseArray[2].(bool)
				message := responseArray[3].(string)

				if success {
					fmt.Printf("Event %s accepted by relay %s: %s\n", eventID, relayURL, message)
				} else {
					fmt.Printf("Event %s rejected by relay %s: %s\n", eventID, relayURL, message)
				}
			} else {
				log.Printf("Unexpected response from relay %s: %v", relayURL, responseArray)
			}
		}(relayURL)
	}
}