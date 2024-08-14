package routes

import (
	"badger/src/utils"
	"net/http"

	types "badger/src/types"
)

func BadgeForm(w http.ResponseWriter, r *http.Request) {
	data := types.PageData{
		Title: "Badge Form Page",
	}

	// Call RenderTemplate with the specific template for this route
	utils.RenderTemplate(w, data, "badgeForm.html")
}
