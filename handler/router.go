package handler

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/monzo/slog"

	"github.com/arussellsaw/budgie/pkg/util"

	"github.com/gorilla/mux"
)

func Routes(ctx context.Context, r *mux.Router, build embed.FS) {
	r.HandleFunc("/api/logout", handleLogout)
	r.HandleFunc("/api/create-sheet", handleCreateSheet)
	r.HandleFunc("/api/sync", handleSync)
	r.HandleFunc("/api/enqueue", handleEnqueue)
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/business", handleBusiness)
	r.HandleFunc("/banks", handleSupportedBanks)
	r.HandleFunc("/api/debug/accounts", handleDebugAccounts)
	r.HandleFunc("/api/debug/transactions", handleDebugTransactions)
	r.HandleFunc("/api/debug/cards", handleDebugCards)
	r.HandleFunc("/api/truelayer/reconnect", handleReconnect)
	r.HandleFunc("/delete-token", handleDeleteToken)
	r.Handle("/dashboard", util.ErrHandler(handleDashboard))

	r.Handle("/api/pulse", util.ErrHandler(handlePulse))
	r.PathPrefix("/app/").Handler(frontend(ctx, build))

	fs := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
}

func frontend(ctx context.Context, build embed.FS) http.Handler {
	if util.IsProd() {
		fsys, err := fs.Sub(build, "build")
		if err != nil {
			panic(err)
		}
		return http.FileServer(http.FS(fsys))
	}
	slog.Debug(ctx, "running dev mode frontend")
	u, _ := url.Parse("http://localhost:3000")
	return httputil.NewSingleHostReverseProxy(u)
}
