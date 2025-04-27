package cmd

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fatcatfablab/doorbot2/httphandlers"
	"github.com/fatcatfablab/doorbot2/sender"
	"github.com/fatcatfablab/doorbot2/types"
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
	tz           string
	silent       bool

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start duties",
		Run:   start,
	}
)

func init() {
	pf := startCmd.PersistentFlags()
	pf.StringVar(&httpAddr, "httpAddr", ":8443", "Address to listen on")
	pf.BoolVar(&secure, "secure", true, "Listen using TLS")
	pf.StringVar(&cert, "cert", "certs/cert.pem", "Path to the certificate")
	pf.StringVar(&key, "key", "certs/key.pem", "Path to the private key")
	pf.StringVar(&slackToken, "slackToken", os.Getenv("DOORBOT2_SLACK_TOKEN"), "Slack token")
	pf.StringVar(&slackChannel, "slackChannel", os.Getenv("DOORBOT2_SLACK_CHANNEL"), "Slack channel")
	pf.StringVar(&tz, "timezone", "America/New_York", "Time zone")
	pf.BoolVar(&silent, "silent", false, "Whether it should post to slack or not")

	rootCmd.AddCommand(startCmd)
}

func start(cmd *cobra.Command, args []string) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT)
	wg := sync.WaitGroup{}

	slack := sender.NewSlack(slackChannel, slackToken, silent)
	httpServer := initHttpServer(slack)
	go startHttpServer(&wg, httpServer)
	wg.Add(1)

	s := <-done
	log.Print("Received signal ", s)

	if err := httpServer.Close(); err != nil {
		log.Printf("error closing http server: %s", err)
	}

	wg.Wait()
}

func initHttpServer(slack types.Sender) *http.Server {
	return &http.Server{
		Addr:    httpAddr,
		Handler: httphandlers.NewMux(accessDb, slack),
	}
}

// This function doesn't return until s is closed, or on error calling
// ListenAndServe
func startHttpServer(wg *sync.WaitGroup, s *http.Server) {
	var err error

	log.Printf("Server listening on %q", httpAddr)
	if secure {
		log.Printf("Listener will use TLS")
		err = s.ListenAndServeTLS(cert, key)
	} else {
		err = s.ListenAndServe()
	}
	wg.Done()

	if !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("error starting http server", err)
	} else {
		log.Print("http server closed gracefully")
	}
}
