package handler

import (
	"net/http"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/monzo/slog"
)

func handleSignUp(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := r.Form.Get("email")
	pass := r.Form.Get("pass")

	u, err := domain.UserByEmail(ctx, email)
	if err != nil {
		slog.Error(ctx, "Err: %s", err)
	}
	if u != nil {
		slog.Info(ctx, "User already exists: %s", email)
		http.Error(w, "user already exists", http.StatusBadRequest)
		return
	}

	u, err = domain.NewUser(ctx, email, pass)
	if err != nil {
		slog.Error(ctx, "Error creating user: %s", err)
		http.Error(w, "couldn't create user", 404)
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

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
