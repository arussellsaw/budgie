package handler

import (
	"net/http"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/stripe"

	"cloud.google.com/go/pubsub"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"

	"github.com/monzo/slog"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
)

func handleEnqueue(w http.ResponseWriter, r *http.Request) {
	var (
		ctx   = r.Context()
		users []domain.User
		u     = authn.User(ctx)
		err   error
	)
	if u != nil {
		users = []domain.User{*u}
	} else {
		users, err = domain.ListUsers(ctx)
		if err != nil {
			slog.Error(ctx, "error listing users: %s", err)
			return
		}
	}
	ps, err := pubsub.NewClient(ctx, util.Project())
	if err != nil {
		slog.Error(ctx, "error getting pubsub client: %s", err)
		return
	}
	t := ps.Topic("sync-users")
	for _, user := range users {
		user := user
		ok, err := stripe.HasSubscription(ctx, &user)
		if err != nil {
			slog.Error(ctx, "error checking subscription: %s", err)
			continue
		}
		if !ok {
			slog.Warn(ctx, "not enqueueing lapsed user: %s", user.ID)
			continue
		}
		result := t.Publish(ctx, &pubsub.Message{
			Data: []byte(user.ID),
		})
		_, err = result.Get(ctx)
		if err != nil {
			slog.Error(ctx, "error publishing: %s", err)
		}
	}
}
