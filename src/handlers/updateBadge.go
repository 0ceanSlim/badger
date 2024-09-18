package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"badger/src/utils" // Import the utils package to use RelayList

	"github.com/nbd-wtf/go-nostr"
)

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

	var updatedEvent nostr.Event
	if err := json.NewDecoder(r.Body).Decode(&updatedEvent); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Send the updated event to the user's relays
	sendEventToRelays(updatedEvent, allRelays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "badge updated"})
}
