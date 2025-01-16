package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	addr   = flag.String("addr", ":8443", "Address to listen on")
	secure = flag.Bool("secure", true, "Whether to use TLS")
	cert   = flag.String("cert", "certs/cert.pem", "Path to the certificate")
	key    = flag.String("key", "certs/key.pem", "Path to the private key")
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

	var err error
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
