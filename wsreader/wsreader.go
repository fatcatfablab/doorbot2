package wsreader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/fatcatfablab/doorbot2/types"

	"github.com/coder/websocket"
)

const (
	path        = "/api/v1/developer/devices/notifications"
	granted     = "ACCESS"
	accessEvent = "access.logs.add"
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
		var err error
		// Can't use `wsjson.Read` here because it'll close the connection
		// on decoding errors, and we expect errors there because the
		// messages received are heterogeneous.
		_, reader, readerErr := w.conn.Reader(ctx)
		if readerErr != nil {
			if !errors.Is(ctx.Err(), context.Canceled) {
				err = fmt.Errorf("error reading from websocket: %s", readerErr)
			}

			// break only if there's no reader, otherwise we need to read
			// from it.
			if reader == nil {
				return err
			}
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, reader); err != nil {
			return fmt.Errorf("error copying ws reader: %s", err)
		}

		var msg wsMsg
		j := json.NewDecoder(&buf)
		if err := j.Decode(&msg); err != nil {
			if readerErr != nil {
				return readerErr
			}
			continue
		}

		if msg.Event != accessEvent {
			if readerErr != nil {
				return readerErr
			}
			continue
		}

		if err := w.processMsg(ctx, &msg); err != nil {
			log.Printf("error dealing with message: %s", err)
		}

		// If we got an error at the start of the loop, break here after all
		// the reading has happened because if there's data left in the reader
		// the connection will hang.
		if readerErr != nil {
			log.Print("returning because there was an error")
			return err
		}
	}

}

func (w *WsReader) processMsg(ctx context.Context, msg *wsMsg) error {
	log.Printf("Processing ws msg: %+v", *msg)
	r := types.AccessRecord{
		Timestamp:     time.Now(),
		Name:          msg.Data.Source.Actor.DisplayName,
		AccessGranted: msg.Data.Source.Event.Result == granted,
	}

	if msg.Data.Source.Actor.DisplayName == "" ||
		msg.Data.Source.Actor.DisplayName == "N/A" ||
		!r.AccessGranted {
		return nil
	}

	s, bumped, err := w.db.AddRecord(ctx, r)
	if err != nil {
		return fmt.Errorf("error bumping %s: %s", r.Name, err)
	}
	if bumped && w.slack != nil {
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

	return nil
}
