package token

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/monzo/slog"
	"google.golang.org/grpc"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/secret"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/store"
)

const collection = "banksheets#tokens"

func Set(ctx context.Context, id, owner, kind string, config *oauth2.Config, token *oauth2.Token) error {
	var refreshToken string

	if existing, st, err := doGet(ctx, config, id); err == nil {
		refreshToken = existing.RefreshToken
		if st.OwnerID != owner {
			return fmt.Errorf("existing token has different ownerID")
		}
	}

	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
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
		ID:             id,
		OwnerID:        owner,
		Kind:           kind,
		KeyName:        keyName,
		EncryptedToken: ciphertext,
	}

	_, err = fs.Collection(collection).Doc(t.ID).Set(ctx, t)
	if err != nil {
		return err
	}

	return nil
}

func doGet(ctx context.Context, config *oauth2.Config, id string) (*oauth2.Token, *StoredToken, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	st := StoredToken{}

	doc, err := fs.Collection(collection).Doc(id).Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	err = doc.DataTo(&st)
	if err != nil {
		return nil, nil, err
	}

	t := oauth2.Token{}
	buf, err := secret.Decrypt(ctx, st.EncryptedToken, st.KeyName)
	if err != nil {
		return nil, nil, err
	}
	err = json.Unmarshal(buf, &t)

	if t.RefreshToken == "" {
		slog.Warn(ctx, "Stored token has no referesh token! %s", id)
	}
	return &t, &st, nil
}

func Get(ctx context.Context, config *oauth2.Config, id string) (*oauth2.Token, error) {
	t, st, err := doGet(ctx, config, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting old token")
	}
	src := config.TokenSource(ctx, t)

	token, err := src.Token()
	if err != nil {
		return nil, errors.Wrap(err, "getting token")
	}

	// token was refreshed, let's store the new access token
	if token.AccessToken != t.AccessToken {
		slog.Info(ctx, "Access token was refreshed, setting new token %s", st.ID)
		err = Set(ctx, id, st.OwnerID, st.Kind, config, token)
		if err != nil {
			return nil, err
		}
	}
	return token, nil
}

func ListByUser(ctx context.Context, userID, kind string, config *oauth2.Config) ([]*oauth2.Token, error) {
	fs, err := store.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var tokens []*oauth2.Token
	// maybe get old style token
	t, err := Get(ctx, config, LegacyTokenID(userID, config))
	if err == nil {
		tokens = append(tokens, t)
	}

	docs, err := fs.Collection(collection).
		Where("OwnerID", "==", userID).
		Where("Kind", "==", kind).Documents(ctx).GetAll()
	if err != nil {
		slog.Error(ctx, "code: %s", grpc.Code(err))
		return nil, err
	}
	var errs []error
	for _, doc := range docs {
		st := StoredToken{}
		err = doc.DataTo(&st)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "unmarshaling token"))
			continue
		}

		t := oauth2.Token{}
		buf, err := secret.Decrypt(ctx, st.EncryptedToken, st.KeyName)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "decrypting token"))
			continue
		}
		err = json.Unmarshal(buf, &t)

		if t.RefreshToken == "" {
			slog.Warn(ctx, "Stored token has no referesh token! %s", st.ID)
		}
		src := config.TokenSource(ctx, &t)

		token, err := src.Token()
		if err != nil {
			errs = append(errs, errors.Wrap(err, "getting/refreshing token"))
			continue
		}
		// token was refreshed, let's store the new access token
		if token.AccessToken != t.AccessToken {
			slog.Info(ctx, "Access token was refreshed, setting new token %s", st.ID)
			err = Set(ctx, st.ID, st.OwnerID, st.Kind, config, token)
			if err != nil {
				errs = append(errs, errors.Wrap(err, "storing updated token"))
				continue
			}
		}
		tokens = append(tokens, token)
	}
	return tokens, joinErrors(errs...)
}

func joinErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	var out string
	for _, err := range errs {
		out += err.Error() + ", "
	}
	return errors.New(out)
}

type StoredToken struct {
	ID             string
	OwnerID        string
	Kind           string
	KeyName        string
	EncryptedToken string
}

func LegacyTokenID(id string, config *oauth2.Config) string {
	return fmt.Sprintf("%s#%s", id, config.ClientID)
}
