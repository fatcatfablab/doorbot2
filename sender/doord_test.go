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
		name      string
		stats     db.Stats
		want      string
		wantError string
	}{
		{
			name:      "UTC",
			stats:     db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 0, 0, 0, time.UTC)},
			want:      "01/21/2025,12:00:00,Johnny Melavo,1",
			wantError: "",
		},
		{
			name:      "America/New_York",
			stats:     db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 4, 9, 0, nyLoc)},
			want:      "01/21/2025,12:04:09,Johnny Melavo,1",
			wantError: "",
		},
		{
			name:      "Invisible nsec",
			stats:     db.Stats{Name: username, Last: time.Date(2025, 1, 21, 12, 4, 9, 1235858, nyLoc)},
			want:      "01/21/2025,12:04:09,Johnny Melavo,1",
			wantError: "",
		},
		{
			name:      "Error",
			stats:     db.Stats{},
			want:      "",
			wantError: "doord request returned: 400 Bad Request",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method")
				}
				var got strings.Builder
				_, err := io.Copy(&got, r.Body)
				if err != nil {
					t.Fatalf("error copying req body: %s", err)
				}

				if tt.want != "" && got.String() != tt.want {
					log.Printf("want: %+v", tt.want)
					log.Printf("got : %+v", got.String())
					t.Fatalf("strings differ")
				}

				if tt.wantError != "" {
					w.WriteHeader(http.StatusBadRequest)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()

			u, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("error parsing httptest server url: %s", err)
			}

			s := NewDoord(u)
			err = s.Post(context.Background(), tt.stats)
			if err != nil {
				if tt.wantError == "" {
					t.Errorf("unexpected error: %s", err)
				} else if !strings.Contains(err.Error(), tt.wantError) {
					log.Printf("want: %+v", tt.wantError)
					log.Printf("got : %+v", err.Error())
					t.Errorf("errors differ")
				}
			}
		})
	}
}
