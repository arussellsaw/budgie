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

	fs := http.FileServer(http.Dir("./static/"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
}
