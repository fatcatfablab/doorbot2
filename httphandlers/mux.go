package httphandlers

import (
	"net/http"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/fatcatfablab/doorbot2/types"
)

type handlers struct {
	db    *db.DB
	slack types.Sender
}

func NewMux(accessDb *db.DB, slack types.Sender) *http.ServeMux {
	h := handlers{db: accessDb, slack: slack}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /udm", h.udmRequest)
	return mux
}
