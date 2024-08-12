package routes

import (
	"badger/src/utils"
	"net/http"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Dashboard",
	}
	utils.RenderTemplate(w, data, "index.html")
}
