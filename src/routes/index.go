package routes

import (
	"badger/src/handlers"
	"badger/src/types"
	"badger/src/utils"
	"net/http"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := handlers.User.Get(r, "session-name")

	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	displayName, _ := session.Values["displayName"].(string)

	data := types.PageData{
		Title:       "Dashboard",
		PublicKey:   publicKey,
		DisplayName: displayName,
	}
	utils.RenderTemplate(w, data, "index.html")
}
