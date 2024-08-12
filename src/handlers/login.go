package handlers

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var User = sessions.NewCookieStore([]byte("your-secret-key"))

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	publicKey := r.FormValue("publicKey")
	if publicKey == "" {
		http.Error(w, "Missing publicKey", http.StatusBadRequest)
		return
	}

	// Store the public key in session
	session, _ := User.Get(r, "session-name")
	session.Values["publicKey"] = publicKey

	// Ensure the session cookie expires when the browser is closed
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0, // Session expires when the browser is closed
		HttpOnly: true,
		Secure:   true, // Ensure this if you're using HTTPS
	}

	session.Save(r, w)

	// Redirect to the root ("/")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}