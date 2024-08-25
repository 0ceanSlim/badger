package utils

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type PageData struct {
	Title           string
	Theme           string
	PublicKey       string
	DisplayName     string
	Picture         string
	About           string
	Relays          RelayList
	AwardedBadges   []Badge
	CollectedBadges []Badge
	CreatedBadges   []Badge
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

func RenderTemplate(w http.ResponseWriter, data PageData, view string, components ...string) {

	// Define the specific template for the route
	viewTemplate := filepath.Join(viewsDir, view)

	// Define component templates
	componentTemplates := []string{
		filepath.Join(viewsDir, "components", "awarded-badges.html"),
		filepath.Join(viewsDir, "components", "collected-badges.html"),
		filepath.Join(viewsDir, "components", "created-badges.html"),
	}

	// Combine layout, view, and component templates
	templates := append(layout, viewTemplate)
	templates = append(templates, componentTemplates...)

	// Parse all templates
	tmpl, err := template.ParseFiles(templates...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute the "layout" template
	err = tmpl.ExecuteTemplate(w, "layout", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
