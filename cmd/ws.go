package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/spf13/cobra"
)

const path = "/api/v1/developer/devices/notifications"
const event = "access.logs.add"

type wsMsg struct {
	Event string `json:"event"`
	Data  wsData `json:"data"`
}

type wsData struct {
	Source wsSource `json:"_source"`
}

type wsSource struct {
	Actor wsActor `json:"actor"`
	Event wsEvent `json:"event"`
}

type wsActor struct {
	Id          string `json:"id"`
	DisplayName string `json:"display_name"`
	AlternateId string `json:"alternate_id"`
}

type wsEvent struct {
	Type   string `json:"type"`
	Result string `json:"result"`
}

var (
	wsAddr string

	wsCmd = &cobra.Command{
		Use:   "ws",
		Short: "Run the websocket subscriber",
		Run:   ws,
	}
)

func init() {
	wsCmd.PersistentFlags().StringVar(&wsAddr, "addr", "localhost:8080", "http service address")
	rootCmd.AddCommand(wsCmd)
}

func ws(cmd *cobra.Command, args []string) {
	u := url.URL{Scheme: "wss", Host: wsAddr, Path: path}
	log.Printf("connecting to %s", u.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	opts := websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + os.Getenv("UA_TOKEN")},
			"Upgrade":       {"websocket"},
			"Connection":    {"Upgrade"},
		},
		HTTPClient: httpClient,
	}
	c, _, err := websocket.Dial(ctx, u.String(), &opts)
	if err != nil {
		log.Fatal(err)
	}
	defer c.CloseNow()

	for {
		_, r, err := c.Reader(context.Background())
		if err != nil {
			log.Printf("error reading from websocket: %s", err)
			continue
		}

		var msg wsMsg
		j := json.NewDecoder(r)
		if err := j.Decode(&msg); err != nil {
			continue
		}

		processMsg(msg)
	}
}

func processMsg(msg wsMsg) {
	if msg.Event != event {
		log.Printf("%s received. Ignoring", msg.Event)
		return
	}
	log.Printf("%+v", msg)
}
