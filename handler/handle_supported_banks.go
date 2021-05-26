package handler

import (
	"html/template"
	"net/http"

	"github.com/arussellsaw/budgie/pkg/truelayer"

	"github.com/monzo/slog"
)

type banksData struct {
	Providers []truelayer.Provider
}

func handleSupportedBanks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t := template.New("banks.html")
	t, err := t.ParseFiles("tmpl/banks.html")
	if err != nil {
		slog.Error(ctx, "Error parsing template: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
	ps, err := truelayer.Providers(ctx)
	if err != nil {
		slog.Error(ctx, "Error getting providers: %s", err)
		http.Error(w, err.Error(), 500)
	}
	err = t.Execute(w, banksData{
		Providers: ps,
	})
	if err != nil {
		slog.Error(ctx, "Index: %s", err)
	}
}
