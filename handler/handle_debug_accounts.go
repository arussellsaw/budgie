package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arussellsaw/budgie/pkg/authn"
	"github.com/arussellsaw/budgie/pkg/truelayer"
)

func handleDebugAccounts(w http.ResponseWriter, r *http.Request) {
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
	buf, _ := json.MarshalIndent(out, "", "  ")
	w.Write(buf)
}
