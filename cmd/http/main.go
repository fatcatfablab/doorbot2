package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	tz      = "America/New_York"
	granted = "Access Granted"
)

var (
	addr     = flag.String("addr", ":8443", "Address to listen on")
	secure   = flag.Bool("secure", true, "Whether to use TLS")
	cert     = flag.String("cert", "certs/cert.pem", "Path to the certificate")
	key      = flag.String("key", "certs/key.pem", "Path to the private key")
	dbPath   = flag.String("dbPath", "access.sqlite", "Path to the sqlite3 database")
	accessDb *db.DB
)

type DoordMsg struct {
	Timestamp     string `json:"timestamp"`
	Name          string `json:"name"`
	AccessGranted bool   `json:"access_granted"`
}

type UdmMsg struct {
	Event         string     `json:"event"`
	EventObjectId string     `json:"event_object_id"`
	Data          UdmMsgData `json:"data"`
}

type UdmMsgData struct {
	Location map[string]any `json:"location"`
	Device   map[string]any `json:"device"`
	Actor    *UdmActor      `json:"actor"`
	Object   *UdmObject     `json:"object"`
}

type UdmActor struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type UdmObject struct {
	AuthenticationType  string `json:"authentication_call"`
	AuthenticationValue string `json:"authentication_value"`
	PolicyId            string `json:"policy_id"`
	PolicyName          string `json:"policy_name"`
	ReaderId            string `json:"reader_id"`
	Result              string `json:"result"`
}

func handleUdmRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Udm request received: " + sb.String())
	msg := UdmMsg{}
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
	if _, err := accessDb.AddRecord(req.Context(), r); err != nil {
		log.Printf("error bumping %s: %s", msg.Data.Actor.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleDoordRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Doord request received: " + sb.String())
	msg := DoordMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error decoding override request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := time.ParseInLocation(time.RFC3339, msg.Timestamp, accessDb.Loc())
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
	if _, err := accessDb.AddRecord(req.Context(), r); err != nil {
		log.Printf("error updating db: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func main() {
	flag.Parse()

	var err error
	accessDb, err = db.New(*dbPath, tz)
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer accessDb.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /udm{$}", handleUdmRequest)
	mux.HandleFunc("POST /doord{$}", handleDoordRequest)

	s := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	log.Printf("Server listening on %q", *addr)
	if *secure {
		log.Printf("Listener will use TLS")
		err = s.ListenAndServeTLS(*cert, *key)
	} else {
		err = s.ListenAndServe()
	}
	if err != nil {
		panic(err)
	}
}
