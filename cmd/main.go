package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	addr = flag.String("addr", ":8080", "Address to listen on")
)

type Msg struct {
	Event         string  `json:"event"`
	EventObjectId string  `json:"event_object_id"`
	Data          MsgData `json:"data"`
}

type MsgData struct {
	Location map[string]any `json:"location"`
	Device   map[string]any `json:"device"`
	Actor    *Actor         `json:"actor"`
	Object   map[string]any `json:"object"`
}

type Actor struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func handler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("req received!")

	j := json.NewDecoder(req.Body)
	msg := Msg{}
	if err := j.Decode(&msg); err != nil {
		log.Printf("error parsing message: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Printf("%+v\n", msg)
	if msg.Data.Actor != nil {
		fmt.Printf("%+v\n", msg.Data.Actor)
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /{$}", handler)

	s := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	log.Printf("Server listening on %q", *addr)
	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
