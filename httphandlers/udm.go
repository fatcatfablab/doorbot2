package httphandlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fatcatfablab/doorbot2/types"
)

const (
	granted = "Access Granted"
)

type udmMsg struct {
	Event          string     `json:"event"`
	EventObjectId  string     `json:"event_object_id"`
	Data           udmMsgData `json:"data"`
	TimeForTesting *time.Time `json:"time_for_testing,omitempty"`
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
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, req.Body)
	if err != nil {
		log.Printf("error copying body to buffer")
	}
	log.Printf("UDM request received: %s", buffer.String())

	j := json.NewDecoder(&buffer)
	msg := udmMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error parsing message: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var ts time.Time
	if msg.TimeForTesting != nil {
		ts = *msg.TimeForTesting
	} else {
		ts = time.Now()
	}

	r := types.AccessRecord{
		Timestamp:     ts,
		Name:          msg.Data.Actor.Name,
		AccessGranted: msg.Data.Object.Result == granted,
	}

	if r.Name == "" || r.Name == "N/A" || !r.AccessGranted {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s, bumped, err := h.db.AddRecord(req.Context(), r)
	if err != nil {
		log.Printf("error bumping %s: %s", msg.Data.Actor.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if bumped && h.slack != nil {
		err = h.slack.Post(req.Context(), s)
		if err != nil {
			log.Printf("error posting message to slack: %s", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
