package handler

import (
	"net/http"

	"github.com/monzo/slog"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/arussellsaw/bank-sheets/pkg/sheets"
)

func handleCreateSheet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := domain.UserFromContext(ctx)
	if u == nil {
		return
	}
	if u.SheetID != "" {
		w.Write([]byte(u.SheetID))
		http.Redirect(w, r, "/", 302)
	}
	s, err := sheets.NewClient(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting sheets client: %s", err)
		return
	}
	sheetID, err := s.Create(ctx)
	if err != nil {
		slog.Error(ctx, "Error creating sheet: %s", err)
		w.Write([]byte(err.Error()))
		return
	}
	u.SheetID = sheetID
	err = domain.UpdateUser(ctx, u)
	if err != nil {
		slog.Error(ctx, "Error updating user: %s", err)
		w.Write([]byte(err.Error()))
		return
	}
	http.Redirect(w, r, "/", 302)
}
