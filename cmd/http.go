package cmd

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/fatcatfablab/doorbot2/httphandlers"
	"github.com/fatcatfablab/doorbot2/sender"
	"github.com/spf13/cobra"
)

var (
	// flags
	httpAddr     string
	secure       bool
	cert         string
	key          string
	slackToken   string
	slackChannel string
	doordUrl     string
	tz           string

	// command
	httpCmd = &cobra.Command{
		Use:   "http",
		Short: "Run the http webhook endpoint",
		Run:   httpServe,
	}
)

func init() {
	pf := httpCmd.PersistentFlags()
	pf.StringVar(&httpAddr, "addr", ":8443", "Address to listen on")
	pf.BoolVar(&secure, "secure", true, "Listen using TLS")
	pf.StringVar(&cert, "cert", "certs/cert.pem", "Path to the certificate")
	pf.StringVar(&key, "key", "certs/key.pem", "Path to the private key")
	pf.StringVar(&slackToken, "slackToken", os.Getenv("DOORBOT2_SLACK_TOKEN"), "Slack token")
	pf.StringVar(&slackChannel, "slackChannel", os.Getenv("DOORBOT2_SLACK_CHANNEL"), "Slack channel")
	pf.StringVar(&doordUrl, "doordUrl", os.Getenv("DOORBOT2_DOORD_URL"), "Doord integration url")
	pf.StringVar(&tz, "timezone", "America/New_York", "Time zone")

	rootCmd.AddCommand(httpCmd)
}

func httpServe(_ *cobra.Command, _ []string) {
	var err error

	dUrl, err := url.Parse(doordUrl)
	if err != nil {
		log.Fatalf("failed to parse %s: %s", doordUrl, err)
	}
	doordSender := sender.NewDoord(dUrl)
	slackSender := sender.NewSlack(slackChannel, slackToken)
	s := &http.Server{
		Addr:    httpAddr,
		Handler: httphandlers.NewMux(accessDb, slackSender, doordSender),
	}

	log.Printf("Server listening on %q", httpAddr)
	if secure {
		log.Printf("Listener will use TLS")
		err = s.ListenAndServeTLS(cert, key)
	} else {
		err = s.ListenAndServe()
	}
	if err != nil {
		panic(err)
	}

}
