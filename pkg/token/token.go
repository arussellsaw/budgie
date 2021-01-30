package token

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arussellsaw/bank-sheets/pkg/secret"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/monzo/slog"
	"google.golang.org/grpc"

	"github.com/arussellsaw/bank-sheets/pkg/store"
)

const collection = "banksheets#tokens"

func Set(ctx context.Context, id string, config *oauth2.Config, token *oauth2.Token) error {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return err
	}
	buf, err := json.Marshal(token)
	if err != nil {
		return err
	}
	ciphertext, keyName, err := secret.Encrypt(ctx, buf)
	if err != nil {
		return err
	}

	t := &StoredToken{
		ID:             tokenID(id, config),
		KeyName:        keyName,
		EncryptedToken: ciphertext,
	}

	_, err = fs.Collection(collection).Doc(t.ID).Set(ctx, t)
	if err != nil {
		return err
	}

	return nil
}

func Get(ctx context.Context, config *oauth2.Config, id string) (*oauth2.Token, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	st := StoredToken{}

	doc, err := fs.Collection(collection).Doc(tokenID(id, config)).Get(ctx)
	if err != nil {
		slog.Error(ctx, "code: %s", grpc.Code(err))
		return nil, err
	}
	err = doc.DataTo(&st)
	if err != nil {
		return nil, err
	}

	t := oauth2.Token{}
	buf, err := secret.Decrypt(ctx, st.EncryptedToken, st.KeyName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &t)

	src := config.TokenSource(ctx, &t)

	token, err := src.Token()
	if err != nil {
		return nil, errors.Wrap(err, "getting token")
	}

	// token was refreshed, let's store the new access token
	if token.AccessToken != t.AccessToken {
		slog.Info(ctx, "Access token was refreshed, setting new token")
		err = Set(ctx, id, config, token)
		if err != nil {
			return nil, err
		}
	}
	return token, nil
}

func GetSource(ctx context.Context, config *oauth2.Config, id string) (oauth2.TokenSource, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	st := StoredToken{}

	doc, err := fs.Collection(collection).Doc(tokenID(id, config)).Get(ctx)
	if err != nil {
		slog.Error(ctx, "code: %s", grpc.Code(err))
		return nil, err
	}
	err = doc.DataTo(&st)
	if err != nil {
		return nil, err
	}

	t := oauth2.Token{}
	buf, err := secret.Decrypt(ctx, st.EncryptedToken, st.KeyName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &t)
	if err != nil {
		return nil, err
	}

	return config.TokenSource(ctx, &t), nil
}

type StoredToken struct {
	ID             string
	KeyName        string
	EncryptedToken string
}

func tokenID(id string, config *oauth2.Config) string {
	return fmt.Sprintf("%s#%s", id, config.ClientID)
}
