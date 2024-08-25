package routes

import (
	"badger/src/utils"
	"net/http"
)

func CreatedBadges(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Created Badges",
		// Populate with actual data as needed
	}
	utils.RenderTemplate(w, data, "components/created-badges.html")
}
