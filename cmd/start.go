package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fatcatfablab/doorbot2/httphandlers"
	"github.com/fatcatfablab/doorbot2/sender"
	"github.com/fatcatfablab/doorbot2/types"
	"github.com/fatcatfablab/doorbot2/wsreader"
	"github.com/spf13/cobra"
)

const (
	wsRetry = 5 * time.Second
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
	wsAddr       string
	wsToken      string

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
	pf.StringVar(&doordUrl, "doordUrl", os.Getenv("DOORBOT2_DOORD_URL"), "Doord integration url")
	pf.StringVar(&tz, "timezone", "America/New_York", "Time zone")
	pf.StringVar(&wsAddr, "wsAddr", os.Getenv("DOORBOT2_WS_URL"), "http service address")
	pf.StringVar(&wsToken, "token", os.Getenv("DOORBOT2_WS_TOKEN"), "auth token")

	rootCmd.AddCommand(startCmd)
}

func start(cmd *cobra.Command, args []string) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	slack, doord := initSenders()

	httpServer := initHttpServer(slack, doord)
	go startHttpServer(&wg, httpServer)
	wg.Add(1)

	go initWsReader(ctx, &wg, slack, doord)
	wg.Add(1)

	s := <-done
	log.Print("Received signal ", s)

	if err := httpServer.Close(); err != nil {
		log.Printf("error closing http server: %s", err)
	}

	cancel()
	wg.Wait()
}

func initSenders() (types.Sender, types.Sender) {
	dUrl, err := url.Parse(doordUrl)
	if err != nil {
		log.Fatalf("failed to parse %s: %s", doordUrl, err)
	}
	doord := sender.NewDoord(dUrl)
	slack := sender.NewSlack(slackChannel, slackToken)

	return slack, doord
}

func initHttpServer(slack, doord types.Sender) *http.Server {
	return &http.Server{
		Addr:    httpAddr,
		Handler: httphandlers.NewMux(accessDb, slack, doord),
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

// This function doesn't return until the context is cancelled. If it fails to
// connect to the websocket, or if the connection dies, it retries after 10s.
func initWsReader(ctx context.Context, wg *sync.WaitGroup, slack, doord types.Sender) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for {
		if errors.Is(ctx.Err(), context.Canceled) {
			break
		}

		wr, err := wsreader.New(
			wsAddr,
			wsToken,
			httpClient,
			accessDb,
			slack,
			doord,
		)
		if err != nil {
			log.Printf("error initializing websocket reader: %s", err)
			time.Sleep(wsRetry)
			continue
		}

		if err := wr.StartReader(ctx); err != nil {
			log.Printf("websocket error: %s", err)
			time.Sleep(wsRetry)
			continue
		}

		log.Print("websocket closed gracefully")
	}

	wg.Done()
}
