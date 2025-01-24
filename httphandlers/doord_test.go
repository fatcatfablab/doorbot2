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
	doordUrl = "/doord"
	username = "xx"
)

func doordReqBuilder(payload string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		var buffer bytes.Buffer
		b := &buffer
		b.WriteString(payload)
		req, err := http.NewRequest(http.MethodPost, doordUrl, b)
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

	mux := NewMux(accessDb, nil, nil)

	for _, tt := range []struct {
		name       string
		reqBuilder func(*testing.T) *http.Request
		wantCode   int
		wantStats  db.Stats
		postSlack  bool
	}{
		{
			name:       "Invalid json",
			reqBuilder: doordReqBuilder("invalid json"),
			wantCode:   http.StatusBadRequest,
			wantStats:  db.Stats{},
		},
		{
			name:       "Valid request",
			reqBuilder: doordReqBuilder(`{"timestamp":"2025-01-20T00:20:09.760614","name":"xx","access_granted":true}`),
			wantCode:   http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  1,
				Streak: 1,
				Last:   time.Date(2025, 1, 20, 0, 20, 9, 0, accessDb.Loc()),
			},
			postSlack: true,
		},
		{
			name:       "Continue streak",
			reqBuilder: doordReqBuilder(`{"timestamp":"2025-01-21T00:20:09.760614","name":"xx","access_granted":true}`),
			wantCode:   http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  2,
				Streak: 2,
				Last:   time.Date(2025, 1, 21, 0, 20, 9, 0, accessDb.Loc()),
			},
			postSlack: true,
		},
		{
			name:       "Access denied doesn't post",
			reqBuilder: doordReqBuilder(`{"timestamp":"2025-01-21T00:20:09.760614","name":"xx","access_granted":false}`),
			wantCode:   http.StatusNoContent,
			wantStats:  db.Stats{},
			postSlack:  false,
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
