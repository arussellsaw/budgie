package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
)

func handleDebugAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := domain.UserFromContext(ctx)
	if u == nil {
		http.Error(w, "unauthorised", http.StatusForbidden)
		return
	}

	tls, err := truelayer.GetClients(ctx, u.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out := []truelayer.Account{}
	for _, tl := range tls {
		accs, err := tl.Accounts(ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		out = append(out, accs...)
	}
	buf, _ := json.MarshalIndent(out, "", "  ")
	w.Write(buf)
}
