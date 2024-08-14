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
	// Login
	mux.HandleFunc("/login", routes.LoginViewHandler) // Login route
	mux.HandleFunc("/do-login", handlers.LoginHandler)
	mux.HandleFunc("/logout", handlers.LogoutHandler) // Logout process
	// Initialize Routes
	mux.HandleFunc("/", routes.IndexHandler)
	mux.HandleFunc("/badgeform", routes.BadgeFormHandler)
	// Serve Static Files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Function Handlers
	// Login Handler TO-DO - handle session with pubkey
	//mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/create-badge", handlers.CreateBadgeHandler)

	fmt.Printf("Server is running on http://localhost:%d\n", cfg.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), mux)
}