package main

import (
	"context"
	"net"
	"net/http"
	"os"

	"github.com/monzo/slog"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/arussellsaw/bank-sheets/handler"
	"github.com/arussellsaw/bank-sheets/pkg/idgen"
	"github.com/arussellsaw/bank-sheets/pkg/logging"
	"github.com/arussellsaw/bank-sheets/pkg/sheets"
	"github.com/arussellsaw/bank-sheets/pkg/store"
	"github.com/arussellsaw/bank-sheets/pkg/truelayer"
)

func main() {
	logger := logging.ColourLogger{Writer: os.Stdout}
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

	sheets.Routes(r)
	truelayer.Routes(r)
	handler.Routes(r)

	srv := http.Server{
		Addr:    ":8080",
		Handler: domain.UserSessionMiddleware(r),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	slog.Error(ctx, "server exiting: %s", srv.ListenAndServe())
}
