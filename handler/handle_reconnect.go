package handler

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"
)

func handleReconnect(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		err error
	)
	defer func() {
		if err != nil {
			http.Error(w, err.Error(), 500)
		}
	}()
	tokenID := r.URL.Query().Get("token_id")
	if tokenID == "" {
		err = fmt.Errorf("missing token ID")
		return
	}
	tl, err := truelayer.GetClient(ctx, tokenID)
	if err != nil {
		err = errors.Wrap(err, "getting client")
		return
	}
	reauthURL, err := tl.ReauthenticateURL(ctx)
	if err != nil {
		err = errors.Wrap(err, "getting URL")
		return
	}
	http.Redirect(w, r, reauthURL, http.StatusTemporaryRedirect)
}
