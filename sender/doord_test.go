package sender

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	username = "Johnny Melavo"
)

func TestPost(t *testing.T) {
	nyLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("can't load NY time zone: %s", err)
	}

	for _, tt := range []struct {
		name          string
		stats         db.Stats
		server        *httptest.Server
		errorExpected string
	}{
		{
			name:  "UTC",
			stats: db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 0, 0, 0, time.UTC)},
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method")
				}
				var got strings.Builder
				_, err := io.Copy(&got, r.Body)
				if err != nil {
					t.Fatalf("error copying req body: %s", err)
				}

				want := "01/21/2025,12:00:00,Johnny Melavo,1"
				if got.String() != want {
					log.Printf("want: %+v", want)
					log.Printf("got : %+v", got.String())
					t.Fatalf("strings differ")
				}

				w.WriteHeader(http.StatusOK)
			})),
		},
		{
			name:  "America/New_York",
			stats: db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 4, 9, 0, nyLoc)},
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method")
				}
				var got strings.Builder
				_, err := io.Copy(&got, r.Body)
				if err != nil {
					t.Fatalf("error copying req body: %s", err)
				}

				want := "01/21/2025,12:04:09,Johnny Melavo,1"
				if got.String() != want {
					log.Printf("want: %+v", want)
					log.Printf("got : %+v", got.String())
					t.Fatalf("strings differ")
				}

				w.WriteHeader(http.StatusOK)
			})),
		},
		{
			name:  "Invisible nsec",
			stats: db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 4, 9, 1235858, nyLoc)},
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method")
				}
				var got strings.Builder
				_, err := io.Copy(&got, r.Body)
				if err != nil {
					t.Fatalf("error copying req body: %s", err)
				}

				want := "01/21/2025,12:04:09,Johnny Melavo,1"
				if got.String() != want {
					log.Printf("want: %+v", want)
					log.Printf("got : %+v", got.String())
					t.Fatalf("strings differ")
				}

				w.WriteHeader(http.StatusOK)
			})),
		},
		{
			name:          "Error",
			stats:         db.Stats{},
			errorExpected: "doord request returned: 400 Bad Request",
			server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			})),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer tt.server.Close()
			defer tt.server.CloseClientConnections()

			u, err := url.Parse(tt.server.URL)
			if err != nil {
				t.Fatalf("error parsing httptest server url: %s", err)
			}

			s := NewDoord(u)
			err = s.Post(context.Background(), tt.stats)
			if err != nil {
				if tt.errorExpected == "" {
					t.Errorf("unexpected error: %s", err)
				} else if !strings.Contains(err.Error(), tt.errorExpected) {
					log.Printf("want: %+v", tt.errorExpected)
					log.Printf("got : %+v", err.Error())
					t.Errorf("errors differ")
				}
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-ctx.Done()
	}))
	defer server.Close()
	defer cancel()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("error parsing httptest server url: %s", err)
	}
	d := NewDoord(u)
	err = d.Post(ctx, db.Stats{})
	if err == nil {
		t.Error("client didn't error")
	}
	if !strings.Contains(err.Error(), "Client.Timeout exceeded") {
		t.Errorf("client didn't timeout: %s", err)
	}
}
