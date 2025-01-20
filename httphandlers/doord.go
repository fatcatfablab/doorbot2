package httphandlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

type doordMsg struct {
	Timestamp     string `json:"timestamp"`
	Name          string `json:"name"`
	AccessGranted bool   `json:"access_granted"`
}

func (h handlers) doordRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Doord request received: " + sb.String())
	msg := doordMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error decoding override request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := time.ParseInLocation(time.RFC3339, msg.Timestamp, h.db.Loc())
	if err != nil {
		log.Printf("couldn't parse timestamp %q: %s", msg.Timestamp, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	r := db.AccessRecord{
		Timestamp:     t,
		Name:          msg.Name,
		AccessGranted: msg.AccessGranted,
	}
	if _, err := h.db.AddRecord(req.Context(), r); err != nil {
		log.Printf("error updating db: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
