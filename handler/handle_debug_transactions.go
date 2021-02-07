package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"
)

func handleDebugTransactions(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	ctx := r.Context()
	u := authn.User(ctx)
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
	for _, acc := range out {
		if acc.AccountID == accountID {
			res, err := acc.Transactions(ctx, false)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			buf, _ := json.MarshalIndent(res, "", "  ")
			w.Write(buf)
			return
		}
	}
}
