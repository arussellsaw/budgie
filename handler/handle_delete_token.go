package handler

import (
	"html/template"
	"net/http"

	"github.com/monzo/slog"

	"github.com/arussellsaw/budgie/pkg/token"
)

func handleDeleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method == http.MethodPost {
		r.ParseForm()
		if r.Form.Get("confirm") != "true" {
			return
		}
		err := token.Delete(ctx, r.Form.Get("token_id"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	tokenID := r.URL.Query().Get("token_id")
	t := template.New("delete_token.html")
	t, err := t.ParseFiles("tmpl/delete_token.html")
	if err != nil {
		slog.Error(ctx, "Error parsing template: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
	err = t.Execute(w, struct {
		TokenID string
		Message string
	}{tokenID, r.URL.Query().Get("message")})
	if err != nil {
		slog.Error(ctx, "Error parsing template: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
}
