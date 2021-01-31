package handler

import "net/http"

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Path:  "/",
		Name:  "sheets-session",
		Value: "",
	})
	http.Redirect(w, r, "/", 302)
}
