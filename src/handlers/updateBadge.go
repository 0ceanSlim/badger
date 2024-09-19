package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"badger/src/utils"

	"github.com/nbd-wtf/go-nostr"
)

// UpdateBadgeHandler handles the event of updating a badge
func UpdateBadgeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Decode the updated badge event from the request body
	var updatedEvent nostr.Event
	err := json.NewDecoder(r.Body).Decode(&updatedEvent)
	if err != nil {
		log.Printf("Failed to decode the request body: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log the updated event for debugging
	log.Printf("Received updated event: %+v", updatedEvent)

	// Send the updated event to the user's relays
	sendEventToRelays(updatedEvent, allRelays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "badge updated"})
}
