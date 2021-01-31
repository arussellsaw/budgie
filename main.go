package main

import (
	"context"
	"net"
	"net/http"
	"os"

	sloggcloud "github.com/arussellsaw/slog-gcloud"

	"github.com/monzo/slog"

	"github.com/gorilla/mux"

	"github.com/arussellsaw/bank-sheets/domain"
	"github.com/arussellsaw/bank-sheets/handler"
	"github.com/arussellsaw/bank-sheets/pkg/idgen"
	"github.com/arussellsaw/bank-sheets/pkg/logging"
	"github.com/arussellsaw/bank-sheets/pkg/sheets"
	"github.com/arussellsaw/bank-sheets/pkg/store"
	"github.com/arussellsaw/bank-sheets/pkg/truelayer"
	"github.com/arussellsaw/bank-sheets/pkg/util"
)

func main() {
	var logger slog.Logger

	sloggcloud.ProjectID = os.Getenv("PROJECT_ID")

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
