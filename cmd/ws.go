package cmd

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"

	"github.com/fatcatfablab/doorbot2/types"
	"github.com/fatcatfablab/doorbot2/wsreader"
	"github.com/spf13/cobra"
)

const path = "/api/v1/developer/devices/notifications"
const event = "access.logs.add"

var (
	wsAddr  string
	wsToken string

	wsCmd = &cobra.Command{
		Use:   "ws",
		Short: "Run the websocket subscriber",
		Run:   ws,
	}
)

func init() {
	pf := wsCmd.PersistentFlags()
	pf.StringVar(&wsAddr, "wsAddr", "localhost:8080", "http service address")
	pf.StringVar(&wsToken, "token", os.Getenv("DOORBOT2_WS_TOKEN"), "auth token")
	rootCmd.AddCommand(wsCmd)
}

type dummySender struct{}

func (d *dummySender) Post(ctx context.Context, s types.Stats) error {
	log.Printf("dummySender called with: %+v", s)
	return nil
}

func ws(_ *cobra.Command, _ []string) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	dummy := dummySender{}
	wr, err := wsreader.New(wsAddr, wsToken, httpClient, accessDb, &dummy, &dummy)
	if err != nil {
		log.Fatalf("error initializing websocket reader: %s", err)
	}

	if err := wr.StartReader(context.Background()); err != nil {
		log.Fatal(err)
	}
}
