package components

import (
	"badger/src/handlers"
	"badger/src/utils"
	"html/template"
	"net/http"
	"sync"
)

// Cache for storing badges without expiration
var badgesCache = struct {
	sync.RWMutex
	data map[string]utils.PageData
}{
	data: make(map[string]utils.PageData),
}

func RenderCreatedBadges(w http.ResponseWriter, r *http.Request) {
	// Retrieve session
	session, _ := handlers.User.Get(r, "session-name")

	// Retrieve publicKey from session
	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return
	}

	// Check if cache should be cleared
	clearCache := r.URL.Query().Get("clear_cache")

	if clearCache == "true" {
		// Clear the cache for this user
		badgesCache.Lock()
		delete(badgesCache.data, publicKey)
		badgesCache.Unlock()
	}

	// Check cache after potential clearing
	badgesCache.RLock()
	cachedData, found := badgesCache.data[publicKey]
	badgesCache.RUnlock()

	if found && clearCache != "true" {
		// Serve from cache
		renderCreatedBadges(w, cachedData)
		return
	}

	// Cache miss or cleared: Retrieve relays from session
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

	// Store in cache
	badgesCache.Lock()
	badgesCache.data[publicKey] = data
	badgesCache.Unlock()

	// Render the component
	renderCreatedBadges(w, data)
}

func renderCreatedBadges(w http.ResponseWriter, data utils.PageData) {
	tmpl := template.Must(template.ParseFiles("web/views/components/created-badges.html"))
	err := tmpl.ExecuteTemplate(w, "createdBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
