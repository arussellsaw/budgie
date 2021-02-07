package domain

import (
	"context"
	"time"

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

func ListUsers(ctx context.Context) ([]User, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	iter := fs.Collection("banksheets#users").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}
	out := []User{}
	for _, doc := range docs {
		usr := User{}
		err = doc.DataTo(&usr)
		if err != nil {
			return nil, err
		}
		out = append(out, usr)
	}
	return out, nil
}

func UpdateUser(ctx context.Context, u *User) error {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return err
	}
	_, err = fs.Collection("banksheets#users").Doc(u.ID).Set(ctx, u)
	return err
}

func (u *User) SyncTime() string {
	if u.LastSync.IsZero() {
		return ""
	}
	return u.LastSync.Format("2006-01-02 15:04")
}
