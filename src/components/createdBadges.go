package components

import (
	"badger/src/handlers"
	"badger/src/utils"
	"html/template"
	"net/http"
)

// Assuming you have initialized a session store somewhere globally
//var User = sessions.NewCookieStore([]byte("your-secret-key"))

func RenderCreatedBadges(w http.ResponseWriter, r *http.Request) {
	// Retrieve session
	session, _ := handlers.User.Get(r, "session-name")

	// Retrieve publicKey from session
	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	// Retrieve relays from session
	relays, ok := session.Values["relays"].(utils.RelayList)
	if !ok {
		http.Error(w, "No relays found in session", http.StatusInternalServerError)
		return
	}

	// Combine all relays into a single slice
	allRelays := append(relays.Read, relays.Write...)
	allRelays = append(allRelays, relays.Both...)

	// Fetch the created badges from the relays
	badges, err := utils.FetchCreatedBadges(publicKey, allRelays)
	if err != nil {
		http.Error(w, "Failed to fetch badges", http.StatusInternalServerError)
		return
	}

	// Prepare data for the template
	data := utils.PageData{
		CreatedBadges: badges,
	}

	// Render the component
	tmpl := template.Must(template.ParseFiles("web/views/components/created-badges.html"))
	err = tmpl.ExecuteTemplate(w, "createdBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
