package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"badger/src/utils" // Import the utils package to use RelayList

	"github.com/nbd-wtf/go-nostr"
	"golang.org/x/net/websocket"
)

func CreateBadgeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := User.Get(r, "session-name")

	// Fetch the relay list from the session
	relays, ok := session.Values["relays"].(utils.RelayList)
	if !ok {
		log.Println("No relay list found in session")
		http.Error(w, "Relay list not found", http.StatusInternalServerError)
		return
	}

	// Combine all user relays (read, write, both)
	allRelays := append(relays.Read, relays.Write...)
	allRelays = append(allRelays, relays.Both...)

	var event nostr.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Send the event to the user's relays
	sendEventToRelays(event, allRelays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "badge sent"})
}

func sendEventToRelays(event nostr.Event, relayURLs []string) {
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
