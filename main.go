package main

import (
	"badger/src/handlers"
	"badger/src/routes"
	"badger/src/utils"

	//"badger/src/handlers"
	"fmt"
	"net/http"
)

func main() {
	// Load Configurations
	cfg, err := utils.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	mux := http.NewServeMux()
	// Login / Logout
	mux.HandleFunc("/login", routes.Login) // Login route
	mux.HandleFunc("/do-login", handlers.LoginHandler)
	mux.HandleFunc("/logout", handlers.LogoutHandler) // Logout process

	// Initialize Routes
	mux.HandleFunc("/", routes.Index)
	mux.HandleFunc("/badgeform", routes.BadgeForm)
	mux.HandleFunc("/relay-list", routes.RelayList)

	// Function Handlers
	mux.HandleFunc("/create-badge", handlers.CreateBadgeHandler)

	// Serve Static Files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Serve HTML files from the components directory
	mux.Handle("/component/", http.StripPrefix("/component/", http.FileServer(http.Dir("web/views/components"))))

	fmt.Printf("Server is running on http://localhost:%d\n", cfg.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), mux)
}
