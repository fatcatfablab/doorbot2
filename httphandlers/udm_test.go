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
	"github.com/fatcatfablab/doorbot2/types"
)

const (
	udmUrl   = "/udm"
	username = "dummy username"
	tz       = "America/New_York"
)

type MockSender struct {
	posted bool
}

func (s *MockSender) Post(_ context.Context, stats types.Stats) error {
	s.posted = true
	return nil
}

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

	origTs := time.Date(2025, 1, 20, 0, 20, 9, 0, accessDb.Loc())
	origNext := origTs.Add(24 * time.Hour)
	origSame := origNext.Add(1 * time.Hour)

	for _, tt := range []struct {
		name       string
		reqBuilder func(*testing.T) *http.Request
		wantCode   int
		wantStats  types.Stats
		postSlack  bool
	}{
		{
			name:       "Invalid json",
			reqBuilder: udmReqBuilder("invalid json"),
			wantCode:   http.StatusBadRequest,
			wantStats:  types.Stats{},
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
			wantStats: types.Stats{
				Name:   username,
				Total:  1,
				Streak: 1,
				Last:   origTs,
			},
			postSlack: true,
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
			wantStats: types.Stats{
				Name:   username,
				Total:  2,
				Streak: 2,
				Last:   origNext,
			},
			postSlack: true,
		},
		{
			name: "Same day",
			reqBuilder: udmReqBuilderFromMsg(udmMsg{
				Data: udmMsgData{
					Actor:  &udmActor{Name: username},
					Object: &udmObject{Result: granted},
				},
				TimeForTesting: &origSame,
			}),
			wantCode: http.StatusOK,
			wantStats: types.Stats{
				Name:   username,
				Total:  2,
				Streak: 2,
				Last:   origSame,
			},
			postSlack: false,
		},
		{
			name: "N/A doesn't post",
			reqBuilder: udmReqBuilderFromMsg(udmMsg{
				Data: udmMsgData{
					Actor:  &udmActor{Name: "N/A"},
					Object: &udmObject{Result: granted},
				},
				TimeForTesting: &origNext,
			}),
			wantCode:  http.StatusNoContent,
			wantStats: types.Stats{},
			postSlack: false,
		},
		{
			name: "Access denied doesn't post",
			reqBuilder: udmReqBuilderFromMsg(udmMsg{
				Data: udmMsgData{
					Actor:  &udmActor{},
					Object: &udmObject{Result: "Access Denied"},
				},
				TimeForTesting: &origNext,
			}),
			wantCode:  http.StatusNoContent,
			wantStats: types.Stats{},
			postSlack: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			slackSender := MockSender{}
			mux := NewMux(accessDb, &slackSender)
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
			if tt.postSlack != slackSender.posted {
				log.Printf("want slack: %t", tt.postSlack)
				log.Printf("got  slack: %t", slackSender.posted)
				t.Errorf("unexpected slack call/no call")
			}
		})
	}
}
