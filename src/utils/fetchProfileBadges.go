package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

// ProfileBadgesEvent represents a kind 30008 event
type ProfileBadgesEvent struct {
	types.NostrEvent
	Badges []ProfileBadge
}

// ProfileBadge represents a single badge in a ProfileBadgesEvent
type ProfileBadge struct {
	BadgeAwardATag string // From "a" tag: "kind:pubkey:dtag"
	AwardEventID   string // From "e" tag: Award event ID
	AwardRelayURL  string // From "e" tag: Relay URL for the award event
	BadgeAwardedBy string // From the pubkey of "a" tag: The person who awarded the badge
	BadgeAwardDTag string // From dtag of "a" tag: The dtag associated with the badge
}

// FetchProfileBadges fetches badges from multiple relays concurrently with a timeout
func FetchProfileBadges(publicKey string, relays []string) ([]ProfileBadgesEvent, error) {
	var profileBadges []ProfileBadgesEvent
	uniqueBadgeIDs := make(map[string]struct{}) // Set to track unique badge IDs
	var mu sync.Mutex                           // Mutex to protect shared resources
	var wg sync.WaitGroup                       // WaitGroup to wait for all goroutines to finish

	// Channel for collecting results and errors
	resultCh := make(chan ProfileBadgesEvent)
	errCh := make(chan error)

	// Launch a goroutine for each relay
	for _, relayURL := range relays {
		wg.Add(1)
		go func(relayURL string) {
			defer wg.Done()

			// Set up WebSocket connection with a timeout
			conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
			if err != nil {
				log.Printf("Failed to connect to WebSocket: %v\n", err)
				errCh <- err
				return
			}
			defer conn.Close()

			// Set up a timeout
			conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // Use time package for deadline

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
				errCh <- err
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
				log.Printf("Failed to send subscription request: %v\n", err)
				errCh <- err
				return
			}

			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("Error reading WebSocket message: %v\n", err)
					return
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

					if !containsTag(profileBadgesEvent.Tags, "d", "profile_badges") {
						continue
					}

					mu.Lock()
					for i := 0; i < len(profileBadgesEvent.Tags); i++ {
						tag := profileBadgesEvent.Tags[i]
						if tag[0] == "a" && i+1 < len(profileBadgesEvent.Tags) && profileBadgesEvent.Tags[i+1][0] == "e" {
							badgeAwardATag := tag[1]
							if _, exists := uniqueBadgeIDs[badgeAwardATag]; exists {
								i++
								continue
							}
							uniqueBadgeIDs[badgeAwardATag] = struct{}{}

							awardEventID := profileBadgesEvent.Tags[i+1][1]
							awardRelayURL := ""
							if len(profileBadgesEvent.Tags[i+1]) > 2 {
								awardRelayURL = profileBadgesEvent.Tags[i+1][2]
							}

							parts := strings.Split(badgeAwardATag, ":")
							if len(parts) == 3 {
								badgeAwardedBy := parts[1]
								badgeAwardDTag := parts[2]

								profileBadgesEvent.Badges = append(profileBadgesEvent.Badges, ProfileBadge{
									BadgeAwardATag: badgeAwardATag,
									AwardEventID:   awardEventID,
									AwardRelayURL:  awardRelayURL,
									BadgeAwardedBy: badgeAwardedBy,
									BadgeAwardDTag: badgeAwardDTag,
								})
							}
							i++
						}
					}
					mu.Unlock()

					// Send result to the resultCh channel
					resultCh <- profileBadgesEvent
				} else if response[0] == "EOSE" {
					log.Println("End of subscription signal received")
					break
				}
			}
		}(relayURL)
	}

	// Close the result and error channels after all goroutines have finished
	go func() {
		wg.Wait()
		close(resultCh)
		close(errCh)
	}()

	// Collect results from the channels
	for {
		select {
		case profileBadge := <-resultCh:
			if profileBadge.Tags != nil {
				profileBadges = append(profileBadges, profileBadge)
			}
		case err := <-errCh:
			if err != nil {
				log.Printf("Error fetching profile badges: %v\n", err)
			}
		}

		// Exit once the channels are closed and all results have been processed
		if len(profileBadges) > 0 && wg == (sync.WaitGroup{}) {
			return profileBadges, nil
		}
	}
}
// FetchBadgeDefinitions fetches the badge definitions for all profile badges
func FetchBadgeDefinitions(profileBadgesEvents []ProfileBadgesEvent, relays []string) (map[string]types.BadgeDefinition, error) {
    badgeDefinitions := make(map[string]types.BadgeDefinition)
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, event := range profileBadgesEvents {
        for _, badge := range event.Badges {
            awarderPubKey := badge.BadgeAwardedBy
            badgeDTag := badge.BadgeAwardDTag

            wg.Add(1)
            go func(awarderPubKey, badgeDTag string) {
                defer wg.Done()
                for _, relayURL := range relays {
                    // Fetch the badge definition using awarderPubKey and badgeDTag
                    badgeDef, err := fetchBadgeDefinition(relayURL, awarderPubKey, badgeDTag)
                    if err != nil {
                        log.Printf("Failed to fetch badge definition from %s: %v\n", relayURL, err)
                        continue
                    }
                    // Create a unique key based on pubkey and dtag combination
                    combinedKey := fmt.Sprintf("%s:%s", awarderPubKey, badgeDTag)
                    mu.Lock()
                    badgeDefinitions[combinedKey] = badgeDef
                    mu.Unlock()
                    break // Stop trying other relays once a badge definition is found
                }
            }(awarderPubKey, badgeDTag)
        }
    }

    wg.Wait() // Wait for all goroutines to complete
    return badgeDefinitions, nil
}




// fetchBadgeDefinition fetches the badge definition details from the relay using the pubkey and dtag
func fetchBadgeDefinition(relayURL, authorPubKey, dTag string) (types.BadgeDefinition, error) {
    log.Printf("Fetching badge definition for pubkey: %s and dtag: %s from relay: %s\n", authorPubKey, dTag, relayURL)

    conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
    if err != nil {
        log.Printf("Failed to connect to WebSocket for badge definition: %v\n", err)
        return types.BadgeDefinition{}, err
    }
    defer conn.Close()

    // Subscription filter for badge definition (kind 30009)
    filter := types.SubscriptionFilter{
        Kinds:   []int{30009},              // We're interested in badge definitions
        Authors: []string{authorPubKey},    // The badge creator's pubkey
        Tags:    map[string][]string{"d": {dTag}},  // The dtag of the badge
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

        // Handle NOTICE messages indicating bad request or unsupported tag filter
        if response[0] == "NOTICE" {
            log.Printf("NOTICE from WebSocket: %v\n", response)
            return types.BadgeDefinition{}, errors.New("error fetching badge definition: " + response[1].(string))
        }

        if response[0] == "EVENT" {
            // Extract event data for the badge definition
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

            // Ensure the badge definition dTag matches exactly
            if badgeDefEvent.DTag != dTag {
                log.Printf("Received wrong badge definition for dtag: %s. Expected: %s\n", badgeDefEvent.DTag, dTag)
                continue
            }

            return badgeDefEvent, nil
        } else if response[0] == "EOSE" {
            log.Println("End of subscription signal received")
            break
        }
    }

    return types.BadgeDefinition{}, errors.New("badge definition not found")
}



func containsTag(tags [][]string, key, value string) bool {
	for _, tag := range tags {
		if tag[0] == key && tag[1] == value {
			return true
		}
	}
	return false
}
