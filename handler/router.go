package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

func Routes(r *mux.Router) {
	r.HandleFunc("/api/transactions", handleListTransactions)
	r.HandleFunc("/api/login", handleLogin)
	r.HandleFunc("/api/signup", handleSignUp)
	r.HandleFunc("/api/create-sheet", handleCreateSheet)
	r.HandleFunc("/api/sync", handleSync)

	r.HandleFunc("/", handleIndex)

	fs := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
}
