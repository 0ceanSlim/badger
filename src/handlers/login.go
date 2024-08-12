package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LoginRequest struct {
	PublicKey string `json:"publicKey"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Use the public key for session management or authentication
	fmt.Printf("Received Public Key: %s\n", loginReq.PublicKey)

	// Respond to the client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
