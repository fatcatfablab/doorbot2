package httphandlers

import (
	"context"
	"net/http"

	"github.com/fatcatfablab/doorbot2/db"
)

type sender interface {
	Post(ctx context.Context, s db.Stats) error
}

type handlers struct {
	db *db.DB
	s  sender
}

func NewMux(accessDb *db.DB, s sender) *http.ServeMux {
	h := handlers{db: accessDb, s: s}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /doord", h.doordRequest)
	mux.HandleFunc("POST /udm", h.udmRequest)
	return mux
}
