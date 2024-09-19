package utils

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

type AwardedBadge struct {
	Name        string
	Description string
	ImageURL    string
	ThumbURL    string
	AwardedBy   string // Public key of the person who awarded the badge
	EventID     string
	CreatedAt   int64
}

// FetchAwardedBadges fetches badges awarded to the user by searching public relays.
func FetchAwardedBadges(publicKey string, publicRelays []string) ([]AwardedBadge, error) {
	var awardedBadges []AwardedBadge
	seenEventIDs := make(map[string]bool)
	var mu sync.Mutex

	badgeChan := make(chan AwardedBadge)
	errChan := make(chan error)
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(len(publicRelays))

	for _, relayURL := range publicRelays {
		go func(relayURL string) {
			defer wg.Done()

			log.Printf("Connecting to WebSocket: %s\n", relayURL)
			conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
			if err != nil {
				log.Printf("Failed to connect to WebSocket: %v\n", err)
				errChan <- err
				return
			}
			defer conn.Close()

			filter := types.SubscriptionFilter{
				Kinds:   []int{8}, // Badge award events
				Authors: []string{}, // All authors
			}

			subRequest := []interface{}{
				"REQ",
				"sub2",
				filter,
			}

			requestJSON, err := json.Marshal(subRequest)
			if err != nil {
				log.Printf("Failed to marshal subscription request: %v\n", err)
				errChan <- err
				return
			}

			if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
				log.Printf("Failed to send subscription request: %v\n", err)
				errChan <- err
				return
			}

			timeout := time.After(2 * time.Second)
			for {
				select {
				case <-timeout:
					log.Printf("Timeout reached for WebSocket: %s\n", relayURL)
					return
				default:
					conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set 2-second timeout
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

						var event types.NostrEvent
						if err := json.Unmarshal(eventData, &event); err != nil {
							log.Printf("Failed to parse event data: %v\n", err)
							continue
						}

						// Process award events with "p" tags matching the user's public key
						if !containsTag(event.Tags, "p", publicKey) {
							continue
						}

						mu.Lock()
						if seenEventIDs[event.ID] {
							mu.Unlock()
							continue
						}
						seenEventIDs[event.ID] = true
						mu.Unlock()

						// Extract badge details from "a" tag and get the definition
						for _, tag := range event.Tags {
							if tag[0] == "a" {
								badgeType := tag[1]

								// Make sure there's a third element in the "p" tag for the relay URL
								if len(event.Tags) > 2 && len(event.Tags[2]) > 2 {
									badgeDefinitionRelay := event.Tags[2][2] // The relay to fetch the definition

									// Fetch the badge details using the relay URL
									badgeDetails, err := fetchAwardedBadgeDetails(badgeDefinitionRelay, badgeType)
									if err != nil {
										log.Printf("Failed to fetch badge definition: %v\n", err)
										continue
									}

									badge := AwardedBadge{
										Name:        badgeDetails.Name,
										Description: badgeDetails.Description,
										ImageURL:    badgeDetails.ImageURL,
										ThumbURL:    badgeDetails.ThumbURL,
										AwardedBy:   event.PubKey, // The awarding public key
										EventID:     event.ID,
										CreatedAt:   event.CreatedAt,
									}

									badgeChan <- badge
								} else {
									log.Printf("Unexpected tag format: %v\n", event.Tags)
								}
							}
						}
					} else if response[0] == "EOSE" {
						log.Println("End of subscription signal received")
						return
					}
				}
			}
		}(relayURL)
	}

	go func() {
		for badge := range badgeChan {
			awardedBadges = append(awardedBadges, badge)
		}
		close(done)
	}()

	go func() {
		wg.Wait()
		close(badgeChan)
	}()

	select {
	case <-done:
		return awardedBadges, nil
	case err := <-errChan:
		return nil, err
	}
}
// fetchBadgeDetails fetches the badge definition details from the relay.
func fetchAwardedBadgeDetails(relayURL, badgeType string) (AwardedBadge, error) {
	conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		log.Printf("Failed to connect to WebSocket for badge details: %v\n", err)
		return AwardedBadge{}, err
	}
	defer conn.Close()

	parts := strings.Split(badgeType, ":")
	if len(parts) < 2 {
		return AwardedBadge{}, errors.New("invalid badge type format")
	}
	badgeEventID := parts[1]

	filter := types.SubscriptionFilter{
		Kinds:   []int{30009}, // Badge definition event
		Authors: []string{badgeEventID},
	}

	subRequest := []interface{}{
		"REQ",
		"sub3",
		filter,
	}

	requestJSON, err := json.Marshal(subRequest)
	if err != nil {
		return AwardedBadge{}, err
	}

	if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
		return AwardedBadge{}, err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return AwardedBadge{}, err
		}

		var response []interface{}
		if err := json.Unmarshal(message, &response); err != nil {
			continue
		}

		if response[0] == "EVENT" {
			eventData, err := json.Marshal(response[2])
			if err != nil {
				continue
			}

			var event types.NostrEvent
			if err := json.Unmarshal(eventData, &event); err != nil {
				continue
			}

			var badge AwardedBadge
			for _, tag := range event.Tags {
				switch tag[0] {
				case "name":
					badge.Name = tag[1]
				case "description":
					badge.Description = tag[1]
				case "image":
					badge.ImageURL = tag[1]
				case "thumb":
					badge.ThumbURL = tag[1]
				}
			}

			return badge, nil
		} else if response[0] == "EOSE" {
			break
		}
	}

	return AwardedBadge{}, errors.New("badge details not found")
}
