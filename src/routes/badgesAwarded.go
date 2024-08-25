package routes

import (
	"badger/src/utils"
	"net/http"
)

func AwardedBadges(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Awarded Badges",
		// Populate with actual data as needed
	}
	utils.RenderTemplate(w, data, "components/awarded-badges.html")
}
