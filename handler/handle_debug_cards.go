package handler

import (
	"encoding/json"
	"net/http"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"
)

func handleDebugCards(w http.ResponseWriter, r *http.Request) {
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
	out := []truelayer.Card{}
	for _, tl := range tls {
		cards, err := tl.Cards(ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		out = append(out, cards...)
	}
	buf, _ := json.MarshalIndent(out, "", "  ")
	w.Write(buf)
}
