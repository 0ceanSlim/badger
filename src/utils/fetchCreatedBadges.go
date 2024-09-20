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

	var wg sync.WaitGroup
	wg.Add(len(relays))

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

			filter := types.SubscriptionFilter{
				Authors: []string{publicKey},
				Kinds:   []int{30009}, // Badge definition event
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

			timeout := time.After(2 * time.Second)

			for {
				select {
				case <-timeout:
					log.Printf("Timeout reached for WebSocket: %s\n", url)
					return
				default:
					conn.SetReadDeadline(time.Now().Add(2 * time.Second))
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
						eventData, err := json.Marshal(response[2])
						if err != nil {
							log.Printf("Failed to marshal event data from %s: %v\n", url, err)
							continue
						}

						var badgeDefEvent types.BadgeDefinition
						if err := json.Unmarshal(eventData, &badgeDefEvent); err != nil {
							log.Printf("Failed to parse event data from %s: %v\n", url, err)
							continue
						}

						mu.Lock()
						if seenEventIDs[badgeDefEvent.ID] {
							mu.Unlock()
							log.Printf("Duplicate event ID found: %s from %s, skipping...", badgeDefEvent.ID, url)
							continue
						}
						seenEventIDs[badgeDefEvent.ID] = true
						mu.Unlock()

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

						badgeChan <- badgeDefEvent
					} else if response[0] == "EOSE" {
						log.Printf("End of subscription signal received from %s\n", url)
						return
					}
				}
			}
		}(url)
	}

	go func() {
		for badge := range badgeChan {
			badges = append(badges, badge)
		}
		close(done)
	}()

	go func() {
		wg.Wait()
		close(badgeChan)
	}()

	select {
	case <-done:
		return badges, nil
	case err := <-errChan:
		return nil, err
	}
}
