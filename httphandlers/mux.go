package httphandlers

import (
	"net/http"

	"github.com/fatcatfablab/doorbot2/db"
)

type handlers struct {
	db *db.DB
}

func NewMux(accessDb *db.DB) *http.ServeMux {
	h := handlers{db: accessDb}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /doord", h.doordRequest)
	mux.HandleFunc("POST /udm", h.udmRequest)
	return mux
}
