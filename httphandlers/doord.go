package httphandlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	layout = "2006-01-02T15:04:05"
)

type doordMsg struct {
	Timestamp     string `json:"timestamp"`
	Name          string `json:"name"`
	AccessGranted bool   `json:"access_granted"`
}

func (h handlers) doordRequest(w http.ResponseWriter, req *http.Request) {
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, req.Body)
	if err != nil {
		log.Printf("error copying body to buffer")
	}
	log.Printf("Doord request received: %s", buffer.String())

	j := json.NewDecoder(&buffer)
	msg := doordMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error decoding doord request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := time.ParseInLocation(layout, msg.Timestamp, h.db.Loc())
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
