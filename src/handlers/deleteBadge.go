package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

// DeleteBadgeHandler handles the deletion of a badge (constructs an unsigned deletion event)
func DeleteBadgeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	session, _ := User.Get(r, "session-name")
	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		log.Println("Error: User not authenticated")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Extract badge ID from request
	badgeID := r.URL.Query().Get("badge_id")
	if badgeID == "" {
		log.Println("Error: Badge ID is missing")
		http.Error(w, "Badge ID is required", http.StatusBadRequest)
		return
	}

	// Create an unsigned deletion event (NIP-09)
	deletionEvent := &nostr.Event{
		PubKey:    publicKey,
		CreatedAt: nostr.Timestamp(time.Now().Unix()),
		Kind:      5, // Deletion event kind (NIP-09)
		Tags: nostr.Tags{
			[]string{"e", badgeID}, // Reference the badge ID to delete
		},
		Content: "Badge deleted by user",
	}

	// Return the unsigned event to the client
	response, err := json.Marshal(deletionEvent)
	if err != nil {
		log.Printf("Failed to marshal deletion event: %v", err)
		http.Error(w, "Failed to create deletion event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}
