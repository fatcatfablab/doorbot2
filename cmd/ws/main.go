package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const path = "/api/v1/developer/devices/notifications"

var addr = flag.String("addr", "localhost:8080", "http service address")

func main() {
	flag.Parse()

	u := url.URL{Scheme: "wss", Host: *addr, Path: path}
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
		mType, r, err := c.Reader(context.Background())
		if err != nil {
			log.Printf("error reading from websocket: %s", err)
			continue
		}

		buffer := &bytes.Buffer{}
		if _, err := io.Copy(buffer, r); err != nil {
			log.Printf("error copying msg to buffer: %s", err)
			continue
		}

		msgStr := buffer.String()
		if strings.HasPrefix(msgStr, "\"Hello\"") {
			continue
		}

		log.Printf("MessageType: %s", mType)
		log.Printf("Message: %s", msgStr)
	}

	// c.Close(websocket.StatusNormalClosure, "")
}
