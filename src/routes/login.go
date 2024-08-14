package routes

import (
	types "badger/src/types"
	"badger/src/utils"
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	data := types.PageData{
		Title: "Login",
	}
	utils.RenderTemplate(w, data, "login.html")
}