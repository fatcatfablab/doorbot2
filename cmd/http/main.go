package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/fatcatfablab/doorbot2/httphandlers"
	"github.com/fatcatfablab/doorbot2/sender"
)

const (
	tz = "America/New_York"
)

var (
	addr         = flag.String("addr", ":8443", "Address to listen on")
	secure       = flag.Bool("secure", true, "Whether to use TLS")
	cert         = flag.String("cert", "certs/cert.pem", "Path to the certificate")
	key          = flag.String("key", "certs/key.pem", "Path to the private key")
	dbPath       = flag.String("dbPath", "access.sqlite", "Path to the sqlite3 database")
	slackToken   = flag.String("slackToken", os.Getenv("DOORBOT2_SLACK_TOKEN"), "Slack token")
	slackChannel = flag.String("slackChannel", os.Getenv("DOORBOT2_SLACK_CHANNEL"), "Slack channel")
	doordUrl     = flag.String("doordUrl", os.Getenv("DOORBOT2_DOORD_URL"), "Doord integration url")
)

func main() {
	flag.Parse()

	var err error
	accessDb, err := db.New(*dbPath, tz)
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer accessDb.Close()

	dUrl, err := url.Parse(*doordUrl)
	if err != nil {
		log.Fatalf("failed to parse %s: %s", *doordUrl, err)
	}
	doordSender := sender.NewDoord(dUrl)
	slackSender := sender.NewSlack(*slackChannel, *slackToken)
	s := &http.Server{
		Addr:    *addr,
		Handler: httphandlers.NewMux(accessDb, slackSender, doordSender),
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
