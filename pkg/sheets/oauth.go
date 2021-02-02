package sheets

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/idgen"

	"github.com/coreos/go-oidc"

	"github.com/arussellsaw/youneedaspreadsheet/domain"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/token"
)

var OauthConfig *oauth2.Config

func Init(ctx context.Context, m *mux.Router) error {
	m.HandleFunc("/api/sheets/oauth/login", oauthGoogleLogin)
	m.HandleFunc("/api/sheets/oauth/redirect", oauthGoogleCallback)

	OauthConfig = &oauth2.Config{
		RedirectURL:  util.BaseURL() + "/api/sheets/oauth/redirect",
		ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		Scopes: []string{
			"openid",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/spreadsheets",
		},
		Endpoint: google.Endpoint,
	}
	return nil
}

func oauthGoogleLogin(w http.ResponseWriter, r *http.Request) {
	userID := idgen.New("usr")
	u := domain.UserFromContext(r.Context())
	if u != nil {
		userID = u.ID
	}

	oauthState := generateStateOauthCookie(w, userID)

	url := OauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func oauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, _ := r.Cookie("oauthstate")
	ctx := r.Context()
	if oauthState == nil {
		return
	}

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

	if t.RefreshToken == "" {
		slog.Warn(ctx, "no refresh token in response for %s", oauthState.Value)
	}

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		slog.Error(ctx, "error getting oidc provider: %s", err)
		return
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: OauthConfig.ClientID})

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := t.Extra("id_token").(string)
	if !ok {
		slog.Error(ctx, "missing ID token")
		return
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		slog.Error(ctx, "error getting oidc provider: %s", err)
		return
	}

	// Extract custom claims
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		slog.Error(ctx, "error getting oidc claims: %s", err)
		return
	}
	u, err := domain.UserByEmail(ctx, claims.Email)
	if err != nil {
		slog.Error(ctx, "error getting user: %s", err)
		return
	}
	if u == nil {
		// create a new user
		u, err = domain.NewUserWithID(ctx, oauthState.Value, claims.Email)
		if err != nil {
			slog.Error(ctx, "unable to create user: %s", err)
		}
	}

	err = token.Set(ctx, token.LegacyTokenID(oauthState.Value, OauthConfig), oauthState.Value, "sheets", OauthConfig, t)
	if err != nil {
		slog.Error(ctx, "failed to set token: %s", err)
		return
	}

	sess, err := u.Session()
	if err != nil {
		slog.Error(ctx, "Error creating session: %s", err)
		http.Error(w, "error creating session", 500)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "sheets-session",
		Value: sess,
		Path:  "/",
	})
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateStateOauthCookie(w http.ResponseWriter, userID string) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)

	cookie := http.Cookie{Name: "oauthstate", Value: userID, Expires: expiration}
	http.SetCookie(w, &cookie)

	return userID
}
