package handlers

import (
	"badger/src/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nbd-wtf/go-nostr"
)

// DeleteSignedBadgeHandler processes the signed deletion event and sends it to relays
func DeleteSignedBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the signed event from the client
	var signedEvent nostr.Event
	err := json.NewDecoder(r.Body).Decode(&signedEvent)
	if err != nil {
		log.Printf("Failed to decode signed deletion event: %v", err)
		http.Error(w, "Invalid signed event data", http.StatusBadRequest)
		return
	}

	// Get the relay list from session
	session, _ := User.Get(r, "session-name")
	relayList, ok := session.Values["relays"].(utils.RelayList)
	if !ok {
		log.Println("Error: No relay list found in session or incorrect type")
		http.Error(w, "No relay list found", http.StatusInternalServerError)
		return
	}

	// Combine all relays (Read, Write, Both) into a single slice
	allRelays := append(relayList.Read, relayList.Write...)
	allRelays = append(allRelays, relayList.Both...)

	// Send the signed deletion event to all relays
	for _, relay := range allRelays {
		err := utils.SendToRelay(relay, signedEvent)
		if err != nil {
			log.Printf("Failed to send deletion event to relay %s: %v", relay, err)
			http.Error(w, fmt.Sprintf("Failed to broadcast deletion event to relay: %s", relay), http.StatusInternalServerError)
			return
		}
	}

	// Respond with success
	response := map[string]string{"status": "success", "message": "Signed badge deletion event broadcasted successfully"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
