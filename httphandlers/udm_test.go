package httphandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	udmUrl = "/udm"
)

func udmReqBuilder(payload string) func(*testing.T) *http.Request {
	return func(t *testing.T) *http.Request {
		var buffer bytes.Buffer
		b := &buffer
		b.WriteString(payload)
		req, err := http.NewRequest(http.MethodPost, udmUrl, b)
		if err != nil {
			t.Fatalf("error creating request: %s", err)
		}
		return req
	}
}

func udmReqBuilderFromMsg(msg udmMsg) func(*testing.T) *http.Request {
	var sb strings.Builder
	j := json.NewEncoder(&sb)
	j.Encode(msg)
	return udmReqBuilder(sb.String())
}

func TestUdmRequest(t *testing.T) {
	accessDb, err := db.New(path.Join(t.TempDir(), "test-udm-request.sqlite"), tz)
	if err != nil {
		t.Fatalf("error creating db: %s", err)
	}

	mux := NewMux(accessDb)
	origTs := time.Date(2025, 1, 20, 0, 20, 9, 0, accessDb.Loc())
	origNext := origTs.Add(24 * time.Hour)

	for _, tt := range []struct {
		name       string
		reqBuilder func(*testing.T) *http.Request
		wantCode   int
		wantStats  db.Stats
	}{
		{
			name:       "Invalid json",
			reqBuilder: udmReqBuilder("invalid json"),
			wantCode:   http.StatusBadRequest,
			wantStats:  db.Stats{},
		},
		{
			name: "Valid request",
			reqBuilder: udmReqBuilderFromMsg(udmMsg{
				Data: udmMsgData{
					Actor:  &udmActor{Name: username},
					Object: &udmObject{Result: granted},
				},
				TimeForTesting: &origTs,
			}),
			wantCode: http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  1,
				Streak: 1,
				Last:   origTs,
			},
		},
		{
			name: "Continue streak",
			reqBuilder: udmReqBuilderFromMsg(udmMsg{
				Data: udmMsgData{
					Actor:  &udmActor{Name: username},
					Object: &udmObject{Result: granted},
				},
				TimeForTesting: &origNext,
			}),
			wantCode: http.StatusOK,
			wantStats: db.Stats{
				Name:   username,
				Total:  2,
				Streak: 2,
				Last:   origNext,
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
