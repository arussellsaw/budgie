package store

import (
	"context"
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
)

var ErrStoreNotFound = errors.New("not_found.store: couldn't find store in context")

func Init(ctx context.Context) (*firestore.Client, error) {
	projectID := "russellsaw"

	var err error
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return fs, err
	}

	return fs, nil
}

type fsKey string

func FromContext(ctx context.Context) (*firestore.Client, error) {
	c, ok := ctx.Value(fsKey("firestore")).(*firestore.Client)
	if !ok {
		return nil, ErrStoreNotFound
	}
	return c, nil
}

func WithStore(ctx context.Context, c *firestore.Client) context.Context {
	return context.WithValue(ctx, fsKey("firestore"), c)
}

func StoreMiddleware(h http.Handler, fs *firestore.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(
			w,
			r.WithContext(
				WithStore(
					r.Context(),
					fs,
				),
			),
		)
	})
}
