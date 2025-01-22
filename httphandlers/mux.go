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
	db    *db.DB
	slack sender
	doord sender
}

func NewMux(accessDb *db.DB, slack sender, doord sender) *http.ServeMux {
	h := handlers{db: accessDb, slack: slack, doord: doord}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /doord", h.doordRequest)
	mux.HandleFunc("POST /udm", h.udmRequest)
	return mux
}
