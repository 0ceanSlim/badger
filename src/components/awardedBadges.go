package components

import (
	"badger/src/handlers"
	"badger/src/utils"
	"html/template"
	"net/http"
	"sync"
)

// Cache for storing awarded badges without expiration
var awardedBadgesCache = struct {
	sync.RWMutex
	data map[string]utils.PageData
}{
	data: make(map[string]utils.PageData),
}

func RenderAwardedBadges(w http.ResponseWriter, r *http.Request) {
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
		awardedBadgesCache.Lock()
		delete(awardedBadgesCache.data, publicKey)
		awardedBadgesCache.Unlock()
	}

	// Check cache after potential clearing
	awardedBadgesCache.RLock()
	cachedData, found := awardedBadgesCache.data[publicKey]
	awardedBadgesCache.RUnlock()

	if found && clearCache != "true" {
		// Serve from cache
		renderAwardedBadges(w, cachedData)
		return
	}

	// Cache miss or cleared: Retrieve the public relays
	// (Assuming a predefined list of public relays)
	publicRelays := []string{
		"wss://nos.lol",
		"wss://relay.damus.io",
		"wss://relay.nostr.band",
		"wss://relay.primal.net",
		"wss://offchain.pub",
		"wss://nostr.mom",
		"wss://nostr.oxtr.dev",
		"wss://nostr.fmt.wiz.biz",
		"wss://nostr.bitcoiner.social",
		"wss://relay.snort.social",
		"wss://soloco.nl",
		// Add more public relays as needed
	}

	// Fetch awarded badges from public relays
	awardedBadges, err := utils.FetchAwardedBadges(publicKey, publicRelays)
	if err != nil {
		http.Error(w, "Failed to fetch awarded badges", http.StatusInternalServerError)
		return
	}

	// Prepare data for the template
	data := utils.PageData{
		AwardedBadges: awardedBadges,
	}

	// Store in cache
	awardedBadgesCache.Lock()
	awardedBadgesCache.data[publicKey] = data
	awardedBadgesCache.Unlock()

	// Render the component
	renderAwardedBadges(w, data)
}

func renderAwardedBadges(w http.ResponseWriter, data utils.PageData) {
	tmpl := template.Must(template.ParseFiles("web/views/components/awarded-badges.html"))
	err := tmpl.ExecuteTemplate(w, "awardedBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
