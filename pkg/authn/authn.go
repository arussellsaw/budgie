package authn

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/monzo/slog"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
)

type sessionKey string

func User(ctx context.Context) *domain.User {
	u, ok := ctx.Value(sessionKey("user")).(*domain.User)
	if !ok {
		return nil
	}
	return u
}

func UserSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionCookie, err := r.Cookie("sheets-session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if sessionCookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		token, err := jwt.Parse(sessionCookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			return []byte(os.Getenv("TOKEN_SECRET")), nil
		})

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			userID, ok := claims["user"].(string)
			if !ok {
				slog.Info(ctx, "no user claim")
				next.ServeHTTP(w, r)
				return
			}
			u, err := domain.UserByID(ctx, userID)
			if err != nil {
				slog.Info(ctx, "no user: %s %s", userID, err)
				next.ServeHTTP(w, r)
				return
			}
			slog.Info(ctx, "User session: %s", u.ID)
			ctx := withUser(ctx, u)
			r = r.WithContext(ctx)
		} else {
			slog.Error(ctx, "invalid token %s", token)
		}
		next.ServeHTTP(w, r)
	})
}

func withUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, sessionKey("user"), u)
}

func Session(userID string) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": userID,
	})

	return t.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}
