package httphandlers

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	tz       = "America/New_York"
	method   = http.MethodPost
	url      = "/doord"
	username = "xx"
)

func reqBuilder(payload string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		var buffer bytes.Buffer
		b := &buffer
		b.WriteString(payload)
		req, err := http.NewRequest(http.MethodPost, url, b)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}
		return req
	}
}

func TestDoordRequest(t *testing.T) {
	accessDb, err := db.New(path.Join(t.TempDir(), "test-doord-request.sqlite"), tz)
	if err != nil {
		t.Fatalf("error creating db: %s", err)
	}

	mux := NewMux(accessDb)

	for _, tt := range []struct {
		name       string
		reqBuilder func(*testing.T) *http.Request
		wantCode   int
		wantStats  db.Stats
	}{
		{
			name:       "Invalid json",
			reqBuilder: reqBuilder("invalid json"),
			wantCode:   http.StatusBadRequest,
			wantStats:  db.Stats{},
		},
		{
			name:       "Valid request",
			reqBuilder: reqBuilder(`{"timestamp":"2025-01-20T00:20:09.760614","name":"xx","access_granted":true}`),
			wantCode:   http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  1,
				Streak: 1,
				Last:   time.Date(2025, 1, 20, 0, 20, 9, 0, accessDb.Loc()),
			},
		},
		{
			name:       "Continue streak",
			reqBuilder: reqBuilder(`{"timestamp":"2025-01-21T00:20:09.760614","name":"xx","access_granted":true}`),
			wantCode:   http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  2,
				Streak: 2,
				Last:   time.Date(2025, 1, 21, 0, 20, 9, 0, accessDb.Loc()),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, tt.reqBuilder(t))

			got := resp.Result().StatusCode
			if got != tt.wantCode {
				t.Errorf("unexpected status code: %d. Wanted %d", got, tt.wantCode)
			}

			if tt.wantCode == http.StatusOK {
				got, err := accessDb.Get(context.Background(), username)
				if err != nil {
					t.Fatalf("error getting stats: %s", err)
				}
				if got.Last.In(accessDb.Loc()) != tt.wantStats.Last.In(accessDb.Loc()) {
					log.Printf("want: %+v", tt.wantStats)
					log.Printf("got : %+v", got)
					t.Errorf("stats differ")
				}
			}
		})
	}
}
