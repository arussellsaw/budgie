package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Routes(r *mux.Router) {
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

	fs := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
}
