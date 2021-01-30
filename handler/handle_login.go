package handler

import (
	"net/http"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/monzo/slog"
)

func handleLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := r.Form.Get("email")
	pw := r.Form.Get("pass")

	u, err := domain.UserByEmail(ctx, email)
	if err != nil {
		slog.Error(ctx, "Error getting user: %s", err)
		http.Error(w, "couldn't find user", 404)
		return
	}
	if !u.ValidatePassword(pw) {
		slog.Error(ctx, "bad password %s", pw)
		http.Error(w, "couldn't find user", 404)
		return
	}

	sess, err := u.Session()
	if err != nil {
		slog.Error(ctx, "Error creating session: %s", err)
		http.Error(w, "error creating session", 500)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "sheets-session",
		Value: sess,
		Path:  "/",
	})
	http.Redirect(w, r, "/", 302)
}
