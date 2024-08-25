package routes

import (
	"badger/src/utils"
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	data := utils.PageData{
		Title: "Login",
	}
	utils.RenderTemplate(w, data, "login.html")
}
