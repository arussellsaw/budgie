package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arussellsaw/bank-sheets/domain"

	"github.com/monzo/slog"

	"github.com/arussellsaw/bank-sheets/pkg/truelayer"
)

func handleListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := domain.UserFromContext(ctx)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	c, err := truelayer.NewClient(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting client: %s", err)
		return
	}
	accs, err := c.Accounts(ctx)
	if err != nil {
		slog.Error(ctx, "Error getting accounts: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accs)
}
