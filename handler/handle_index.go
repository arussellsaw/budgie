package handler

import (
	"context"
	"html/template"
	"net/http"

	"github.com/arussellsaw/bank-sheets/pkg/sheets"

	"github.com/arussellsaw/bank-sheets/pkg/truelayer"

	"github.com/arussellsaw/bank-sheets/domain"

	"github.com/monzo/slog"
)

type indexData struct {
	User         *domain.User
	HasTruelayer bool
	HasSheets    bool
	Accounts     []truelayer.Account
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t := template.New("index.html")
	t, err := t.ParseFiles("tmpl/index.html")
	if err != nil {
		slog.Error(ctx, "Error parsing template: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
	u := domain.UserFromContext(ctx)
	hasTL, accs := hasTruelayer(ctx, u)
	slog.Debug(ctx, "%+v", accs)
	t.Execute(w, indexData{
		User:         u,
		HasTruelayer: hasTL,
		HasSheets:    hasSheets(ctx, u),
		Accounts:     accs,
	})
}

func hasTruelayer(ctx context.Context, user *domain.User) (bool, []truelayer.Account) {
	if user == nil {
		return false, nil
	}
	tl, err := truelayer.NewClient(ctx, user.ID)
	if err != nil {
		slog.Error(ctx, "error getting truelayer client: %s", err)
		return false, nil
	}
	accs, err := tl.Accounts(ctx)
	if err != nil {
		slog.Error(ctx, "error getting authorised accounts: %s", err)
		return false, nil
	}
	return len(accs) != 0, accs
}

func hasSheets(ctx context.Context, user *domain.User) bool {
	if user == nil {
		return false
	}
	s, err := sheets.NewClient(ctx, user.ID)
	if err != nil {
		slog.Error(ctx, "error getting sheets client: %s", err)
		return false
	}
	if s == nil {
		return false
	}
	return true
}
