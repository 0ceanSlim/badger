package components

import (
	"badger/src/utils"
	"html/template"
	"net/http"
)

func RenderAwardedBadges(w http.ResponseWriter, r *http.Request) {
	// Prepare data
	data := utils.PageData{
		AwardedBadges: []utils.Badge{
			//{Name: "Badge A", Description: "Description A", DateAwarded: "2024-08-25"},
			//{Name: "Badge B", Description: "Description B", DateAwarded: "2024-08-24"},
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
