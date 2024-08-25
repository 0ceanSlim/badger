package components

import (
	"badger/src/utils"
	"html/template"
	"net/http"
)

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
