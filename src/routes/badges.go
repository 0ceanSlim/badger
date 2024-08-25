package routes

import (
	"badger/src/utils"
	"html/template"
	"net/http"
)

func RenderCollectedBadges(w http.ResponseWriter, r *http.Request) {
	// Prepare data
	data := utils.PageData{
		CollectedBadges: []utils.Badge{
			{Name: "Badge C", Description: "Description C"},
			// Add more badges as needed
		},
	}

	// Render the component
	tmpl := template.Must(template.ParseFiles("web/views/components/collected-badges.html"))
	err := tmpl.ExecuteTemplate(w, "collectedBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RenderAwardedBadges(w http.ResponseWriter, r *http.Request) {
	// Prepare data
	data := utils.PageData{
		AwardedBadges: []utils.Badge{
			{Name: "Badge A", Description: "Description A", DateAwarded: "2024-08-25"},
			{Name: "Badge B", Description: "Description B", DateAwarded: "2024-08-24"},
			// Add more badges as needed
		},
	}

	// Render the component
	tmpl := template.Must(template.ParseFiles("web/views/components/awarded-badges.html"))
	err := tmpl.ExecuteTemplate(w, "awardedBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RenderCreatedBadges(w http.ResponseWriter, r *http.Request) {
	// Prepare data
	data := utils.PageData{
		CreatedBadges: []utils.Badge{
			{Name: "Badge D", Description: "Description D"},
			// Add more badges as needed
		},
	}

	// Render the component
	tmpl := template.Must(template.ParseFiles("web/views/components/created-badges.html"))
	err := tmpl.ExecuteTemplate(w, "createdBadges", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
