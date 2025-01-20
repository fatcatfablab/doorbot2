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

const (
	granted = "Access Granted"
)

type udmMsg struct {
	Event         string     `json:"event"`
	EventObjectId string     `json:"event_object_id"`
	Data          udmMsgData `json:"data"`
}

type udmMsgData struct {
	Location map[string]any `json:"location"`
	Device   map[string]any `json:"device"`
	Actor    *udmActor      `json:"actor"`
	Object   *udmObject     `json:"object"`
}

type udmActor struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type udmObject struct {
	AuthenticationType  string `json:"authentication_call"`
	AuthenticationValue string `json:"authentication_value"`
	PolicyId            string `json:"policy_id"`
	PolicyName          string `json:"policy_name"`
	ReaderId            string `json:"reader_id"`
	Result              string `json:"result"`
}

func (h handlers) udmRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Udm request received: " + sb.String())
	msg := udmMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error parsing message: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	r := db.AccessRecord{
		Timestamp:     time.Now(),
		Name:          msg.Data.Actor.Name,
		AccessGranted: msg.Data.Object.Result == granted,
	}
	if _, err := h.db.AddRecord(req.Context(), r); err != nil {
		log.Printf("error bumping %s: %s", msg.Data.Actor.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
