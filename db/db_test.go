package db

import (
	"log"
	"path"
	"testing"
	"time"
)

const (
	username = "Johnny Melavo"
)

func TestBumpWithTimestamp(t *testing.T) {
	db, err := New(path.Join(t.TempDir(), "test-bump.sqlite"))
	if err != nil {
		t.Fatalf("error opening database: %s", err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("error loading timezone: %s", err)
	}

	for _, tt := range []struct {
		name string
		want AccessRecord
	}{
		{
			name: "First bump",
			want: AccessRecord{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 0, 0, 0, 0, loc)},
		},
		{
			name: "Bump in the same day",
			want: AccessRecord{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 1, 0, 0, 0, loc)},
		},
		{
			name: "Bump the next day",
			want: AccessRecord{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 13, 0, 0, 0, loc)},
		},
		{
			name: "Bump the same day again",
			want: AccessRecord{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 14, 0, 0, 0, loc)},
		},
		{
			name: "Bump the next day again",
			want: AccessRecord{Name: username, Total: 3, Streak: 3, Last: time.Date(2025, 1, 18, 13, 0, 0, 0, loc)},
		},
		{
			name: "Break streak",
			want: AccessRecord{Name: username, Total: 4, Streak: 1, Last: time.Date(2025, 1, 26, 12, 0, 0, 0, loc)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.bumpWithTimestamp(username, tt.want.Last)
			if err != nil {
				t.Fatalf("error bumping username: %s", err)
			}

			if got != tt.want {
				log.Printf("want: %+v", tt.want)
				log.Printf("got:  %+v", got)
				t.Error("records do not match")
			}
		})
	}
}
