package truelayer

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"
	"golang.org/x/oauth2"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/idgen"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/token"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"
)

var (
	OauthConfig *oauth2.Config
)

func Init(ctx context.Context, m *mux.Router) error {
	m.HandleFunc("/api/truelayer/oauth/login", oauthLogin)
	m.HandleFunc("/api/truelayer/oauth/redirect", oauthCallback)

	OauthConfig = &oauth2.Config{
		RedirectURL:  util.BaseURL() + "/api/truelayer/oauth/redirect",
		ClientID:     os.Getenv("TRUELAYER_CLIENT_ID"),
		ClientSecret: os.Getenv("TRUELAYER_CLIENT_SECRET"),
		Scopes: []string{
			"accounts",
			"balance",
			"cards",
			"transactions",
			"offline_access",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://auth.truelayer.com/?providers=uk-ob-all+uk-oauth-all",
			TokenURL:  "https://auth.truelayer.com/connect/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}
	return nil
}

func oauthLogin(w http.ResponseWriter, r *http.Request) {
	u := domain.UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	oauthState := generateStateOauthCookie(w, u.ID)

	url := OauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
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

	err = token.Set(ctx, idgen.New("tok"), oauthState.Value, "truelayer", OauthConfig, t)
	if err != nil {
		slog.Error(ctx, "failed to set token: %s", err)
		return
	}
	slog.Info(ctx, "Set token for user %s", oauthState.Value)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, id string) string {
	var expiration = time.Now().Add(1 * time.Hour)
	cookie := http.Cookie{Name: "oauthstate", Value: id, Expires: expiration}
	http.SetCookie(w, &cookie)
	return id
}
