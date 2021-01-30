package truelayer

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"
	"golang.org/x/oauth2"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/arussellsaw/bank-sheets/pkg/token"
)

func Routes(m *mux.Router) {
	m.HandleFunc("/api/truelayer/oauth/login", oauthLogin)
	m.HandleFunc("/api/truelayer/oauth/redirect", oauthCallback)
}

var OauthConfig = &oauth2.Config{
	RedirectURL:  "http://banksheets.russellsaw.io/api/truelayer/oauth/redirect",
	ClientID:     os.Getenv("TRUELAYER_CLIENT_ID"),
	ClientSecret: os.Getenv("TRUELAYER_CLIENT_SECRET"),
	Scopes: []string{
		"info",
		"accounts",
		"balance",
		"cards",
		"transactions",
		"direct_debits",
		"standing_orders",
		"offline_access",
	},
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://auth.truelayer.com/?providers=uk-ob-all+uk-oauth-all",
		TokenURL:  "https://auth.truelayer.com/connect/token",
		AuthStyle: oauth2.AuthStyleAutoDetect,
	},
}

func oauthLogin(w http.ResponseWriter, r *http.Request) {
	u := domain.UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	oauthState := generateStateOauthCookie(w, u.ID)

	url := OauthConfig.AuthCodeURL(oauthState)
	slog.Info(r.Context(), "generating auth URL for user: %s", u.ID)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func oauthCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")
	ctx := r.Context()

	if r.FormValue("state") != oauthState.Value {
		slog.Error(ctx, "invalid oauth state: %s", oauthState.Value)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	t, err := OauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		slog.Error(ctx, "code exchange wrong: %s", err.Error())
		return
	}

	err = token.Set(ctx, oauthState.Value, OauthConfig, t)
	if err != nil {
		slog.Error(ctx, "failed to set token: %s", err)
		return
	}
	slog.Info(ctx, "Set token for user %s", oauthState.Value)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, id string) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{Name: "oauthstate", Value: id, Expires: expiration}
	http.SetCookie(w, &cookie)
	return id
}
