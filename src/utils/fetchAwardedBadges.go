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
	Dtag 		string
}

// FetchAwardedBadges fetches badges awarded to the user by searching public relays for kind 8 events.
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

			// Create the subscription filter to search for kind 8 events
			filter := types.SubscriptionFilter{
				Kinds:   []int{8}, // Badge award events (kind 8)
				Authors: []string{}, // All authors
				P: []string{publicKey},
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

			log.Printf("Sending subscription request to: %s\nRequest: %s", relayURL, string(requestJSON))
			if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
				log.Printf("Failed to send subscription request: %v\n", err)
				errChan <- err
				return
			}

			// Increase the timeout to 5 seconds to give relays more time to respond
			timeout := time.After(5 * time.Second)
			for {
				select {
				case <-timeout:
					log.Printf("Timeout reached for WebSocket: %s\n", relayURL)
					return
				default:
					conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Increase the read timeout
					_, message, err := conn.ReadMessage()
					if err != nil {
						log.Printf("Error reading WebSocket message from %s: %v\n", relayURL, err)
						return
					}

					log.Printf("Received WebSocket message from %s: %s\n", relayURL, message)
					var response []interface{}
					if err := json.Unmarshal(message, &response); err != nil {
						log.Printf("Failed to unmarshal response: %v\n", err)
						continue
					}

					// Check if it's an event
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

						// Process award events (kind 8) with "p" tags matching the user's public key
						for _, tag := range event.Tags {
							if tag[0] == "p" && tag[1] == publicKey {
								// Ignore duplicate events by checking seenEventIDs
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
								
										// Extract the dtag from the badgeType string
										parts := strings.Split(badgeType, ":")
										if len(parts) < 3 {
											log.Printf("Invalid badge type format: %s\n", badgeType)
											continue
										}
										dtag := parts[len(parts)-1]
								
										// Make sure there's a valid third element in the "p" tag for the relay URL
										for _, ptag := range event.Tags {
											if ptag[0] == "p" && len(ptag) > 2 && ptag[1] == publicKey {
												badgeDefinitionRelay := ptag[2] // The relay to fetch the badge definition
								
												// Fetch the badge details using the relay URL and dtag
												badgeDetails, err := fetchAwardedBadgeDetails(badgeDefinitionRelay, dtag)
												if err != nil {
													log.Printf("Failed to fetch badge definition from relay %s: %v\n", badgeDefinitionRelay, err)
													continue
												}
								
												// Create the awarded badge object with the dtag
												badge := AwardedBadge{
													Name:        badgeDetails.Name,
													Description: badgeDetails.Description,
													ImageURL:    badgeDetails.ImageURL,
													ThumbURL:    badgeDetails.ThumbURL,
													AwardedBy:   event.PubKey, // The awarding public key
													EventID:     event.ID,
													CreatedAt:   event.CreatedAt,
													Dtag:        dtag, // Add the dtag to the object
												}
								
												// Send the awarded badge to the channel
												badgeChan <- badge
											}
										}
									}
								}
							}
						}
					} else if response[0] == "EOSE" {
						log.Printf("End of subscription signal received from %s", relayURL)
						return
					}
				}
			}
		}(relayURL)
	}

	// Goroutine to collect awarded badges from the badgeChan
	go func() {
		for badge := range badgeChan {
			awardedBadges = append(awardedBadges, badge)
		}
		close(done)
	}()

	// Wait for all WebSocket connections to finish
	go func() {
		wg.Wait()
		close(badgeChan)
	}()

	// Return either the awarded badges or any error that occurs
	select {
	case <-done:
		return awardedBadges, nil
	case err := <-errChan:
		return nil, err
	}
}

// fetchAwardedBadgeDetails fetches the badge definition details from the relay.
func fetchAwardedBadgeDetails(relayURL, dtag string) (AwardedBadge, error) {
    conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
    if err != nil {
        log.Printf("Failed to connect to WebSocket for badge details: %v\n", err)
        return AwardedBadge{}, err
    }
    defer conn.Close()

    filter := types.SubscriptionFilter{
        Kinds:   []int{30009}, // Badge definition event (kind 30009)
        D: []string{dtag}, // Add the dtag to the filter
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
	