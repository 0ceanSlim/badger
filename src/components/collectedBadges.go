package components

import (
	"badger/src/handlers"
	"badger/src/utils"
	"html/template"
	"net/http"
	"sync"
)

// Cache for storing collected badges
var collectedBadgesCache = struct {
	sync.RWMutex
	data map[string]utils.PageData
}{
	data: make(map[string]utils.PageData),
}

func RenderCollectedBadges(w http.ResponseWriter, r *http.Request) {
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
        collectedBadgesCache.Lock()
        delete(collectedBadgesCache.data, publicKey)
        collectedBadgesCache.Unlock()
    }

    // Check cache after potential clearing
    collectedBadgesCache.RLock()
    cachedData, found := collectedBadgesCache.data[publicKey]
    collectedBadgesCache.RUnlock()

    if found && clearCache != "true" {
        // Serve from cache
        renderCollectedBadges(w, cachedData)
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

    // Fetch the collected badges from the relays
    badges, err := utils.FetchCollectedBadges(publicKey, allRelays)
    if err != nil {
        http.Error(w, "Failed to fetch collected badges", http.StatusInternalServerError)
        return // Ensure return after http.Error
    }

    // Prepare data for the template
    data := utils.PageData{
        CollectedBadges: badges,
    }

    // Store in cache
    collectedBadgesCache.Lock()
    collectedBadgesCache.data[publicKey] = data
    collectedBadgesCache.Unlock()

    // Render the component
    renderCollectedBadges(w, data)
}

func renderCollectedBadges(w http.ResponseWriter, data utils.PageData) {
    tmpl := template.Must(template.ParseFiles("web/views/components/collected-badges.html"))
    err := tmpl.ExecuteTemplate(w, "collectedBadges", data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return // Ensure return after http.Error
    }
}

