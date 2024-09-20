package components

import (
	"badger/src/handlers"
	"badger/src/types"
	"badger/src/utils"
	"html/template"
	"net/http"
	"sync"
)

// Cache for storing profile badges
var profileBadgesCache = struct {
	sync.RWMutex
	data map[string]utils.PageData
}{
	data: make(map[string]utils.PageData),
}

func RenderProfileBadgeEvent(w http.ResponseWriter, r *http.Request) {
	// Retrieve session
	session, _ := handlers.User.Get(r, "session-name")

	// Retrieve publicKey from session
	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		http.Error(w, "User not logged in", http.StatusUnauthorized)
		return // Ensure return after http.Error
	}

	// Check if cache should be cleared
	clearCache := r.URL.Query().Get("clear_cache")

	if clearCache == "true" {
		// Clear the cache for this user
		profileBadgesCache.Lock()
		delete(profileBadgesCache.data, publicKey)
		profileBadgesCache.Unlock()
	}

	// Check cache after potential clearing
	profileBadgesCache.RLock()
	cachedData, found := profileBadgesCache.data[publicKey]
	profileBadgesCache.RUnlock()

	if found && clearCache != "true" {
		// Serve from cache
		renderProfileBadge(w, cachedData, cachedData.BadgeDefinitions)
		return // Ensure return after serving from cache
	}

	// Cache miss or cleared: Retrieve relays from session
	relays, ok := session.Values["relays"].(utils.RelayList)
	if !ok {
		http.Error(w, "No relays found in session", http.StatusInternalServerError)
		return // Ensure return after http.Error
	}

	// Combine all relays into a single slice
	allRelays := append(relays.Read, relays.Write...)
	allRelays = append(allRelays, relays.Both...)

	// Fetch the profile badges from the relays
	profileBadgesEvents, err := utils.FetchProfileBadges(publicKey, allRelays)
	if err != nil {
		http.Error(w, "Failed to fetch profile badges", http.StatusInternalServerError)
		return
	}

	// Fetch badge definitions
	badgeDefinitions, err := utils.FetchBadgeDefinitions(profileBadgesEvents, allRelays)
	if err != nil {
		http.Error(w, "Failed to fetch badge definitions", http.StatusInternalServerError)
		return
	}
	// Prepare data for the template
	data := utils.PageData{
		ProfileBadges:    profileBadgesEvents,
		BadgeDefinitions: badgeDefinitions,
	}

	// Store in cache
	profileBadgesCache.Lock()
	profileBadgesCache.data[publicKey] = data
	profileBadgesCache.Unlock()

	// Render the component
	renderProfileBadge(w, data, badgeDefinitions)
}

func renderProfileBadge(w http.ResponseWriter, data utils.PageData, badgeDefinitions map[string]types.BadgeDefinition) {
	tmpl := template.Must(template.ParseFiles("web/views/components/profile-badges.html"))

	// Create a struct to pass to the template
	templateData := struct {
		ProfileBadgesEvents []utils.ProfileBadgesEvent
		BadgeDefinitions    map[string]types.BadgeDefinition
	}{
		ProfileBadgesEvents: data.ProfileBadges,
		BadgeDefinitions:    badgeDefinitions,
	}

	err := tmpl.ExecuteTemplate(w, "profileBadges", templateData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
