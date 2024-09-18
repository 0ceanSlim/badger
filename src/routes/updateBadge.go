package routes

import (
	"badger/src/utils"
	"net/http"
)

func UpdateBadgeForm(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Update Badge",
	}

	// Call RenderTemplate with the specific template for this route
	utils.RenderTemplate(w, data, "update-badge.html", false)
}
