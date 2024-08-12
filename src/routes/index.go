package routes

import (
	"badger/src/handlers"
	"badger/src/types"
	"badger/src/utils"
	"net/http"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session
	session, _ := handlers.User.Get(r, "session-name")

	// Check if the user is logged in (i.e., has a public key)
	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		// If not logged in, redirect to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// If logged in, render the dashboard
	data := types.PageData{
		Title:     "Dashboard",
		PublicKey: publicKey,
	}
	utils.RenderTemplate(w, data, "index.html")
}
