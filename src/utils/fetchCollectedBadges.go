package utils

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type CollectedBadge struct {
	BadgeType   string // Badge type from the "a" tag
	AwardedBy   string // Awarding pubkey from the "a" tag
	EventID     string // Event ID from the "e" tag (badge award event)
	Name        string // Badge name from the 30009 event
	Description string // Badge description from the 30009 event
	ImageURL    string // Full-size badge image URL fetched from the 30009 event
	ThumbURL    string // Badge thumbnail URL fetched from the 30009 event
	CreatedAt   int64  // Event creation time
}

// ProfileBadgesEvent represents a kind 30008 event
type ProfileBadgesEvent struct {
	types.NostrEvent
	Badges []ProfileBadge
}

// ProfileBadge represents a single badge in a ProfileBadgesEvent
type ProfileBadge struct {
	BadgeDefinitionID string // From "a" tag: "kind:pubkey:dtag"
	AwardEventID      string // From "e" tag
	AwardRelayURL     string // From "e" tag
}

// FetchCollectedBadges fetches badges from multiple relays.
func FetchCollectedBadges(publicKey string, relays []string) ([]ProfileBadgesEvent, error) {
	var collectedBadges []ProfileBadgesEvent
	uniqueBadgeIDs := make(map[string]struct{}) // Set to track unique badge IDs

	for _, relayURL := range relays {
		log.Printf("Connecting to WebSocket: %s\n", relayURL)
		conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
		if err != nil {
			log.Printf("Failed to connect to WebSocket: %v\n", err)
			continue
		}
		defer conn.Close()

		filter := types.SubscriptionFilter{
			Authors: []string{publicKey},
			Kinds:   []int{30008}, // Profile Badges events
		}

		subRequest := []interface{}{
			"REQ",
			"sub2",
			filter,
		}

		requestJSON, err := json.Marshal(subRequest)
		if err != nil {
			log.Printf("Failed to marshal subscription request: %v\n", err)
			return nil, err
		}

		if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
			log.Printf("Failed to send subscription request: %v\n", err)
			return nil, err
		}

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading WebSocket message: %v\n", err)
				break
			}

			var response []interface{}
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("Failed to unmarshal response: %v\n", err)
				continue
			}

			if response[0] == "EVENT" {
				eventData, err := json.Marshal(response[2])
				if err != nil {
					log.Printf("Failed to marshal event data: %v\n", err)
					continue
				}

				var profileBadgesEvent ProfileBadgesEvent
				if err := json.Unmarshal(eventData, &profileBadgesEvent); err != nil {
					log.Printf("Failed to parse event data: %v\n", err)
					continue
				}

				// Process badge events with "profile_badges" tag
				if !containsTag(profileBadgesEvent.Tags, "d", "profile_badges") {
					continue
				}

				// Process pairs of "a" and "e" tags
				for i := 0; i < len(profileBadgesEvent.Tags); i++ {
					tag := profileBadgesEvent.Tags[i]
					if tag[0] == "a" && i+1 < len(profileBadgesEvent.Tags) && profileBadgesEvent.Tags[i+1][0] == "e" {
						badgeDefinitionID := tag[1]
						if _, exists := uniqueBadgeIDs[badgeDefinitionID]; exists {
							i++ // Skip the next tag as we've already processed this badge
							continue
						}
						uniqueBadgeIDs[badgeDefinitionID] = struct{}{} // Mark this badge as seen

						awardEventID := profileBadgesEvent.Tags[i+1][1]
						awardRelayURL := ""
						if len(profileBadgesEvent.Tags[i+1]) > 2 {
							awardRelayURL = profileBadgesEvent.Tags[i+1][2]
						}

						profileBadgesEvent.Badges = append(profileBadgesEvent.Badges, ProfileBadge{
							BadgeDefinitionID: badgeDefinitionID,
							AwardEventID:      awardEventID,
							AwardRelayURL:     awardRelayURL,
						})
						i++ // Skip the next tag as we've processed it
					}
				}

				collectedBadges = append(collectedBadges, profileBadgesEvent)
			} else if response[0] == "EOSE" {
				log.Println("End of subscription signal received")
				break
			}
		}
	}

	return collectedBadges, nil
}

func FetchBadgeDefinitions(profileBadgesEvents []ProfileBadgesEvent, relays []string) (map[string]types.BadgeDefinition, error) {
	badgeDefinitions := make(map[string]types.BadgeDefinition)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, event := range profileBadgesEvents {
		for _, badge := range event.Badges {
			parts := strings.Split(badge.BadgeDefinitionID, ":")
			if len(parts) != 3 {
				log.Printf("Invalid badge definition ID format: %s\n", badge.BadgeDefinitionID)
				continue
			}
			authorPubKey := parts[1]
			dTag := parts[2]

			wg.Add(1)
			go func(badge ProfileBadge) {
				defer wg.Done()
				for _, relayURL := range relays {
					badgeDef, err := fetchBadgeDefinition(relayURL, authorPubKey, dTag)
					if err != nil {
						log.Printf("Failed to fetch badge definition from %s: %v\n", relayURL, err)
						continue
					}
					mu.Lock()
					badgeDefinitions[badge.BadgeDefinitionID] = badgeDef
					mu.Unlock()
					break // Successfully fetched the badge definition, no need to try other relays
				}
			}(badge)
		}
	}

	wg.Wait() // Wait for all goroutines to finish
	return badgeDefinitions, nil
}

// fetchBadgeDetails fetches the badge definition details from the relay.
func fetchBadgeDefinition(relayURL, authorPubKey, dTag string) (types.BadgeDefinition, error) {
	conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		log.Printf("Failed to connect to WebSocket for badge definition: %v\n", err)
		return types.BadgeDefinition{}, err
	}
	defer conn.Close()

	// Subscription filter for badge definition (kind 30009)
	filter := types.SubscriptionFilter{
		Kinds:   []int{30009},
		Authors: []string{authorPubKey},
		Tags:    map[string][]string{"d": {dTag}},
	}

	subRequest := []interface{}{
		"REQ",
		"sub3",
		filter,
	}

	requestJSON, err := json.Marshal(subRequest)
	if err != nil {
		log.Printf("Failed to marshal subscription request: %v\n", err)
		return types.BadgeDefinition{}, err
	}

	if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
		log.Printf("Failed to send subscription request for badge definition: %v\n", err)
		return types.BadgeDefinition{}, err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v\n", err)
			return types.BadgeDefinition{}, err
		}

		log.Printf("Received WebSocket message for badge definition: %s\n", message)

		var response []interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("Failed to unmarshal badge definition response: %v\n", err)
			continue
		}

		// Handle NOTICE messages indicating bad request
		if response[0] == "NOTICE" {
			log.Printf("NOTICE from WebSocket: %v\n", response)
			return types.BadgeDefinition{}, errors.New("error fetching badge definition: " + response[1].(string))
		}

		if response[0] == "EVENT" {
			eventData, err := json.Marshal(response[2])
			if err != nil {
				log.Printf("Failed to marshal badge definition event data: %v\n", err)
				continue
			}

			var badgeDefEvent types.BadgeDefinition
			if err := json.Unmarshal(eventData, &badgeDefEvent); err != nil {
				log.Printf("Failed to parse badge definition event data: %v\n", err)
				continue
			}

			// Parse badge details from the event tags
			for _, tag := range badgeDefEvent.Tags {
				switch tag[0] {
				case "name":
					badgeDefEvent.Name = tag[1]
				case "description":
					badgeDefEvent.Description = tag[1]
				case "image":
					badgeDefEvent.ImageURL = tag[1]
				case "thumb":
					badgeDefEvent.ThumbURL = tag[1]
				case "d":
					badgeDefEvent.DTag = tag[1]
				}
			}

			return badgeDefEvent, nil
		} else if response[0] == "EOSE" {
			break
		}
	}

	return types.BadgeDefinition{}, errors.New("badge definition not found")
}

// Helper function to check if a tag exists with a specific key and value
func containsTag(tags [][]string, key, value string) bool {
	for _, tag := range tags {
		if tag[0] == key && tag[1] == value {
			return true
		}
	}
	return false
}
