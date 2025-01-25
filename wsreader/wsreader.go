package wsreader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/fatcatfablab/doorbot2/types"

	"github.com/coder/websocket"
)

const (
	path    = "/api/v1/developer/devices/notifications"
	granted = "GRANTED"
)

type WsReader struct {
	db    *db.DB
	conn  *websocket.Conn
	slack types.Sender
	doord types.Sender
}

func New(host, token string, hc *http.Client, db *db.DB, slack types.Sender, doord types.Sender) (*WsReader, error) {
	conn, err := connect(host, token, hc)
	if err != nil {
		return nil, fmt.Errorf("error connecting websocket: %w", err)
	}

	return &WsReader{
		db:    db,
		conn:  conn,
		slack: slack,
		doord: doord,
	}, nil
}

func connect(host, token string, hc *http.Client) (*websocket.Conn, error) {
	u := url.URL{Scheme: "wss", Host: host, Path: path}
	log.Printf("connecting to %s", u.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + token},
			"Upgrade":       {"websocket"},
			"Connection":    {"Upgrade"},
		},
		HTTPClient: hc,
	})
	if err != nil {
		return nil, fmt.Errorf("error dialing websocket: %w", err)
	}

	return c, nil
}

// StartReader only returns when ctx is Done
func (w *WsReader) StartReader(ctx context.Context) error {
	for {
		// Can't use `wsjson.Read` here because it'll close the connection
		// on decoding errors, and we expect errors here because the
		// messages received are heterogeneous.
		_, reader, err := w.conn.Reader(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("error reading from websocket: %s", err)
		}

		var msg wsMsg
		j := json.NewDecoder(reader)
		if err := j.Decode(&msg); err != nil {
			continue
		}

		if err := w.processMsg(ctx, &msg); err != nil {
			log.Printf("error dealing with message: %s", err)
		}

	}
}

func (w *WsReader) processMsg(ctx context.Context, msg *wsMsg) error {
	r := types.AccessRecord{
		Timestamp:     time.Now(),
		Name:          msg.Data.Source.Actor.DisplayName,
		AccessGranted: msg.Data.Source.Event.Result == granted,
	}

	if msg.Data.Source.Actor.DisplayName == "" || !r.AccessGranted {
		return nil
	}

	if s, bumped, err := w.db.AddRecord(ctx, r); err != nil {
		return fmt.Errorf("error bumping %s: %s", r.Name, err)
	} else if bumped {
		if w.slack != nil {
			err = w.slack.Post(ctx, s)
			if err != nil {
				log.Printf("error posting message to slack: %s", err)
			}
		}
		if w.doord != nil {
			err = w.doord.Post(ctx, s)
			if err != nil {
				log.Printf("error posting message to doord: %s", err)
			}
		}
	}

	return nil
}
