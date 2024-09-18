package main

import (
	"badger/src/components"
	"badger/src/handlers"
	"badger/src/routes"
	"badger/src/utils"
	"embed"

	"fmt"
	"net/http"
)

//go:embed web/*
var staticFiles embed.FS

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

	// Render component htmls
	mux.HandleFunc("/collected-badges", components.RenderCollectedBadges)
	mux.HandleFunc("/awarded-badges", components.RenderAwardedBadges)
	mux.HandleFunc("/created-badges", components.RenderCreatedBadges)

	// Function Handlers
	mux.HandleFunc("/create-badge", handlers.CreateBadgeHandler)
	mux.HandleFunc("/delete-badge", handlers.DeleteBadgeHandler)
	mux.HandleFunc("/delete-signed-badge", handlers.DeleteSignedBadgeHandler)

	// Serve Static Files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/static/img/favicon.ico")
	})
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.FS(staticFiles))))


	fmt.Printf("Server is running on http://localhost:%d\n", cfg.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), mux)
}
