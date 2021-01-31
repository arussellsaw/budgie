package domain

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/monzo/slog"

	"github.com/dgrijalva/jwt-go"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/store"
)

type User struct {
	ID       string     `json:"id"`
	Email    string     `json:"email"`
	Created  time.Time  `json:"created"`
	SheetID  string     `json:"sheet_id"`
	LastSync time.Time  `json:"last_sync"`
	Stripe   StripeData `json:"stripe"`
}

type StripeData struct {
	FreeForMyBuds bool
	SessionID     string
	CustomerID    string
	PaidUntil     time.Time
	Error         string
}

func NewUserWithID(ctx context.Context, id, email string) (*User, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	user := User{
		ID:      id,
		Email:   email,
		Created: time.Now(),
	}
	_, err = fs.Collection("banksheets#users").Doc(id).Set(ctx, user)
	return &user, err
}

func UserByID(ctx context.Context, userID string) (*User, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := fs.Collection("banksheets#users").Doc(userID).Get(ctx)
	if err != nil {
		return nil, err
	}
	usr := User{}
	err = doc.DataTo(&usr)
	return &usr, err
}

func UserByEmail(ctx context.Context, email string) (*User, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	iter := fs.Collection("banksheets#users").Where("Email", "==", email).Limit(1).Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	usr := User{}
	err = docs[0].DataTo(&usr)
	return &usr, err
}

func UpdateUser(ctx context.Context, u *User) error {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return err
	}
	_, err = fs.Collection("banksheets#users").Doc(u.ID).Set(ctx, u)
	return err
}

func UserFromContext(ctx context.Context) *User {
	u, ok := ctx.Value("user").(*User)
	if !ok {
		return nil
	}
	return u
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, "user", u)
}

func (u *User) Session() (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": u.ID,
	})

	return t.SignedString([]byte(os.Getenv("TOKEN_SECRET")))
}

func UserSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionCookie, err := r.Cookie("sheets-session")
		if err != nil {
			slog.Info(ctx, "no cookie: %s", err)
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
			u, err := UserByID(ctx, userID)
			if err != nil {
				slog.Info(ctx, "no user: %s %s", userID, err)
				next.ServeHTTP(w, r)
				return
			}
			slog.Info(ctx, "User session: %s", u.ID)
			ctx := WithUser(ctx, u)
			r = r.WithContext(ctx)
		} else {
			slog.Error(ctx, "invalid token %s", token)
		}
		next.ServeHTTP(w, r)
	})
}

func (u *User) SyncTime() string {
	return u.LastSync.Format("2006-01-02 15:04")
}
