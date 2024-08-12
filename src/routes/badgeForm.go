package routes

import (
	"badger/src/utils"
	"net/http"
)

func BadgeFormHandler(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Badge Form Page",
	}

	// Call RenderTemplate with the specific template for this route
	utils.RenderTemplate(w, data, "badgeForm.html")
}
