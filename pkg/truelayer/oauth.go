package truelayer

import (
	"context"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"
	"golang.org/x/oauth2"

	"github.com/arussellsaw/budgie/pkg/authn"
	"github.com/arussellsaw/budgie/pkg/idgen"
	"github.com/arussellsaw/budgie/pkg/token"
	"github.com/arussellsaw/budgie/pkg/util"
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
			AuthURL:   "https://auth.truelayer.com/?providers=uk-ob-all+uk-oauth-all+de-xs2a-all",
			TokenURL:  "https://auth.truelayer.com/connect/token",
			AuthStyle: oauth2.AuthStyleAutoDetect,
		},
	}
	return nil
}

func oauthLogin(w http.ResponseWriter, r *http.Request) {
	u := authn.User(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	id := idgen.New("tok")
	if tokenID := r.URL.Query().Get("token_id"); tokenID != "" {
		id = tokenID
	}

	oauthState := generateStateOauthCookie(w, id)

	url := OauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
	slog.Info(r.Context(), "generating auth URL for user: %s", u.ID)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func oauthCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")
	ctx := r.Context()

	u := authn.User(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	if r.FormValue("state") != oauthState.Value {
		slog.Error(ctx, "invalid oauth state: %s", oauthState.Value)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	tokenID := oauthState.Value

	t, err := OauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		slog.Error(ctx, "code exchange wrong: %s", err.Error())
		return
	}

	err = token.Set(ctx, tokenID, u.ID, "truelayer", OauthConfig, t)
	if err != nil {
		slog.Error(ctx, "failed to set token: %s", err)
		return
	}
	slog.Info(ctx, "Set token for user %s", oauthState.Value)
	ps, err := pubsub.NewClient(ctx, util.Project())
	if err != nil {
		slog.Error(ctx, "error getting pubsub client: %s", err)
		return
	}
	topic := ps.Topic("sync-users")
	result := topic.Publish(ctx, &pubsub.Message{
		Data: []byte(u.ID + "|true"),
	})
	_, err = result.Get(ctx)
	if err != nil {
		slog.Error(ctx, "error publishing: %s", err)
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, id string) string {
	var expiration = time.Now().Add(1 * time.Hour)
	cookie := http.Cookie{Name: "oauthstate", Value: id, Expires: expiration}
	http.SetCookie(w, &cookie)
	return id
}
