package utils

import (
	"badger/src/types"
	"html/template"
	"net/http"
	"path/filepath"
)

type PageData struct {
	Title            string
	Theme            string
	PublicKey        string
	DisplayName      string
	Picture          string
	About            string
	Relays           RelayList
	AwardedBadges    []AwardedBadge
	ProfileBadges    []ProfileBadgesEvent
	BadgeDefinitions map[string]types.BadgeDefinition
	CreatedBadges    []types.BadgeDefinition
}

// Define the base directories for views and templates
const (
	viewsDir     = "web/views/"
	templatesDir = "web/views/templates/"
)

// Define the common layout templates filenames
var templateFiles = []string{
	"layout.html",
	"header.html",
	"footer.html",
}

// Initialize the common templates with full paths
var layout = PrependDir(templatesDir, templateFiles)

var loginLayout = PrependDir(templatesDir, []string{"login-layout.html", "footer.html"})

func RenderTemplate(w http.ResponseWriter, data PageData, view string, useLoginLayout bool, components ...string) {
	// Define the specific template for the route
	viewTemplate := filepath.Join(viewsDir, view)

	// Define component templates
	componentTemplates := []string{
		filepath.Join(viewsDir, "components", "awarded-badges.html"),
		filepath.Join(viewsDir, "components", "profile-badges.html"),
		filepath.Join(viewsDir, "components", "created-badges.html"),
	}

	var templates []string
	if useLoginLayout {
		templates = append(loginLayout, viewTemplate)
	} else {
		templates = append(layout, viewTemplate)
	}
	templates = append(templates, componentTemplates...)

	// Parse all templates
	tmpl, err := template.ParseFiles(templates...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the appropriate layout template
	layoutName := "layout"
	if useLoginLayout {
		layoutName = "login-layout"
	}
	err = tmpl.ExecuteTemplate(w, layoutName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
