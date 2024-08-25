package routes

import (
	"badger/src/utils"
	"net/http"
)

func CollectedBadges(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Collected Badges",
		// Populate with actual data as needed
	}
	utils.RenderTemplate(w, data, "components/collected-badges.html")
}
