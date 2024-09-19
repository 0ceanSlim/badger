package utils

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"badger/src/types"

	"github.com/gorilla/websocket"
)

// FetchCreatedBadges fetches all badges created by a user from their relays concurrently, with timeout
func FetchCreatedBadges(publicKey string, relays []string) ([]types.BadgeDefinition, error) {
	var badges []types.BadgeDefinition
	seenEventIDs := make(map[string]bool)
	var mu sync.Mutex

	badgeChan := make(chan types.BadgeDefinition)
	errChan := make(chan error)
	done := make(chan struct{})

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(len(relays))

	// Spawn a goroutine for each relay
	for _, url := range relays {
		go func(url string) {
			defer wg.Done()
			log.Printf("Connecting to WebSocket: %s\n", url)
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				log.Printf("Failed to connect to WebSocket: %v\n", err)
				errChan <- err
				return
			}
			defer conn.Close()

			// Subscription filter to request badge events
			filter := types.SubscriptionFilter{
				Authors: []string{publicKey},
				Kinds:   []int{30009}, // Badge creation event
			}
			subRequest := []interface{}{
				"REQ",
				"sub1",
				filter,
			}

			requestJSON, err := json.Marshal(subRequest)
			if err != nil {
				log.Printf("Failed to marshal subscription request: %v\n", err)
				errChan <- err
				return
			}

			log.Printf("Sending subscription request to: %s\n", url)
			if err := conn.WriteMessage(websocket.TextMessage, requestJSON); err != nil {
				log.Printf("Failed to send subscription request: %v\n", err)
				errChan <- err
				return
			}

			// Timeout mechanism to exit if no response in 2 seconds
			timeout := time.After(2 * time.Second)

			for {
				select {
				case <-timeout:
					log.Printf("Timeout reached for WebSocket: %s\n", url)
					return
				default:
					// Read message with timeout
					conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set 2-second read deadline
					_, message, err := conn.ReadMessage()
					if err != nil {
						log.Printf("Error reading WebSocket message from %s: %v\n", url, err)
						return
					}

					var response []interface{}
					if err := json.Unmarshal(message, &response); err != nil {
						log.Printf("Failed to unmarshal response from %s: %v\n", url, err)
						continue
					}

					if response[0] == "EVENT" {
						// Process the event
						eventData, err := json.Marshal(response[2])
						if err != nil {
							log.Printf("Failed to marshal event data from %s: %v\n", url, err)
							continue
						}

						var event types.NostrEvent
						if err := json.Unmarshal(eventData, &event); err != nil {
							log.Printf("Failed to parse event data from %s: %v\n", url, err)
							continue
						}

						mu.Lock()
						// Check for duplicate event
						if seenEventIDs[event.ID] {
							mu.Unlock()
							log.Printf("Duplicate event ID found: %s from %s, skipping...", event.ID, url)
							continue
						}
						seenEventIDs[event.ID] = true
						mu.Unlock()

						// Parse badge data from the event
						badge := types.BadgeDefinition{EventID: event.ID}
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

						// Send the badge to the channel
						badgeChan <- badge
					} else if response[0] == "EOSE" {
						log.Printf("End of subscription signal received from %s\n", url)
						return // Move to the next relay
					}
				}
			}
		}(url)
	}

	// Goroutine to collect badges from all workers
	go func() {
		for badge := range badgeChan {
			badges = append(badges, badge)
		}
		close(done)
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(badgeChan)
	}()

	// Block until all badges are collected or error occurs
	select {
	case <-done:
		return badges, nil
	case err := <-errChan:
		return nil, err
	}
}
