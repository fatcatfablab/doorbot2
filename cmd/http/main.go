package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	tz = "America/New_York"
)

var (
	addr     = flag.String("addr", ":8443", "Address to listen on")
	secure   = flag.Bool("secure", true, "Whether to use TLS")
	cert     = flag.String("cert", "certs/cert.pem", "Path to the certificate")
	key      = flag.String("key", "certs/key.pem", "Path to the private key")
	dbPath   = flag.String("dbPath", "access.sqlite", "Path to the sqlite3 database")
	accessDb *db.DB
)

type UdmMsg struct {
	Event         string     `json:"event"`
	EventObjectId string     `json:"event_object_id"`
	Data          UdmMsgData `json:"data"`
}

type UdmMsgData struct {
	Location map[string]any `json:"location"`
	Device   map[string]any `json:"device"`
	Actor    *UdmActor      `json:"actor"`
	Object   map[string]any `json:"object"`
}

type UdmActor struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func handleUdmRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Udm request received: " + sb.String())
	msg := UdmMsg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error parsing message: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := accessDb.Bump(req.Context(), msg.Data.Actor.Name); err != nil {
		log.Printf("error bumping %s: %s", msg.Data.Actor.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleUpdateRequest(w http.ResponseWriter, req *http.Request) {
	sb := &strings.Builder{}
	j := json.NewDecoder(io.TeeReader(req.Body, sb))
	log.Print("Update received: " + sb.String())
	r := db.Stats{}
	if err := j.Decode(&r); err != nil {
		log.Printf("error decoding override request: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := accessDb.Update(req.Context(), r); err != nil {
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
	mux.HandleFunc("POST /{$}", handleUdmRequest)
	mux.HandleFunc("POST /update{$}", handleUpdateRequest)

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
