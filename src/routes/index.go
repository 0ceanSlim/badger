package routes

import (
	"net/http"

	"badger/src/handlers"
	"badger/src/types"
	"badger/src/utils"
)


func IndexHandler(w http.ResponseWriter, r *http.Request) {

	session, _ := handlers.User.Get(r, "session-name")

	publicKey, ok := session.Values["publicKey"].(string)
	if !ok || publicKey == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	
	displayName, _ := session.Values["displayName"].(string)
	picture, _ := session.Values["picture"].(string)
	about, _ := session.Values["about"].(string)

	// Prepare the data to be passed to the template
	data := types.PageData{
		Title:       "Dashboard",
		DisplayName: displayName,
		Picture:     picture,
		PublicKey:   publicKey,
		About:       about,
	}

	utils.RenderTemplate(w, data, "index.html")
}
