package sheets

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/arussellsaw/bank-sheets/domain"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/arussellsaw/bank-sheets/pkg/token"
)

func Routes(m *mux.Router) {
	m.HandleFunc("/api/sheets/oauth/login", oauthGoogleLogin)
	m.HandleFunc("/api/sheets/oauth/redirect", oauthGoogleCallback)
}

var OauthConfig = &oauth2.Config{
	RedirectURL:  "http://banksheets.russellsaw.io/api/sheets/oauth/redirect",
	ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
	Scopes: []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/spreadsheets",
	},
	Endpoint: google.Endpoint,
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	u := domain.UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorised", http.StatusForbidden)
		return
	}

	oauthState := generateStateOauthCookie(w, u.ID)

	url := OauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")
	ctx := r.Context()

	if r.FormValue("state") != oauthState.Value {
		slog.Error(ctx, "invalid oauth google state: %s", oauthState.Value)
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
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, userID string) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)

	cookie := http.Cookie{Name: "oauthstate", Value: userID, Expires: expiration}
	http.SetCookie(w, &cookie)

	return userID
}
