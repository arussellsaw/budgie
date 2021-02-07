package main

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/stripe"

	sloggcloud "github.com/arussellsaw/slog-gcloud"

	"github.com/monzo/slog"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/youneedaspreadsheet/handler"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/idgen"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/logging"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/sheets"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/store"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"
)

func main() {
	var logger slog.Logger

	sloggcloud.ProjectID = util.Project()

	logger = logging.ContextParamLogger{Logger: &sloggcloud.StackDriverLogger{}}

	if !util.IsProd() {
		logger = logging.ColourLogger{Writer: os.Stdout}
	}

	slog.SetDefaultLogger(logger)

	ctx := context.Background()

	idgen.Init(ctx)

	fs, err := store.Init(ctx)
	if err != nil {
		slog.Error(ctx, "Error intialising FireStore: %s", err)
		os.Exit(1)
	}
	ctx = store.WithStore(ctx, fs)

	r := mux.NewRouter()

	err = sheets.Init(ctx, r)
	if err != nil {
		slog.Error(ctx, "Error intialising Google Sheets: %s", err)
		os.Exit(1)
	}
	err = truelayer.Init(ctx, r)
	if err != nil {
		slog.Error(ctx, "Error intialising Truelayer: %s", err)
		os.Exit(1)
	}
	err = stripe.Init(ctx, r)
	if err != nil {
		slog.Error(ctx, "Error intialising Stripe: %s", err)
		os.Exit(1)
	}

	handler.Routes(r)

	srv := http.Server{
		Addr:    ":8080",
		Handler: sloggcloud.CloudContextMiddleware(authn.UserSessionMiddleware(r)),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	slog.Error(ctx, "server exiting: %s", srv.ListenAndServe())
}
