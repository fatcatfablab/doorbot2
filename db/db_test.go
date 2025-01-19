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

func TestBumpStats(t *testing.T) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("error loading timezone: %s", err)
	}

	prevStats := Stats{
		Name: username,
	}
	for _, tt := range []struct {
		name string
		want Stats
	}{
		{
			name: "First bump",
			want: Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 0, 0, 0, 0, loc)},
		},
		{
			name: "Visit in the same day does not bump stats",
			want: Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 1, 0, 0, 0, loc)},
		},
		{
			name: "Visit the next day bumps stats",
			want: Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 13, 0, 0, 0, loc)},
		},
		{
			name: "Visit the same day again does not bump stats",
			want: Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 14, 0, 0, 0, loc)},
		},
		{
			name: "Visit the next day bumps stats again",
			want: Stats{Name: username, Total: 3, Streak: 3, Last: time.Date(2025, 1, 18, 13, 0, 0, 0, loc)},
		},
		{
			name: "Visit on a later date breaks streak, but bumps total",
			want: Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2025, 1, 26, 12, 0, 0, 0, loc)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := bumpStats(prevStats, tt.want.Last)
			if got != tt.want {
				log.Printf("want: %+v", tt.want)
				log.Printf("got:  %+v", got)
				t.Error("records do not match")
			}
			prevStats = got
		})
	}
}

func TestUpdateAndGet(t *testing.T) {
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-get.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("error loading timezone: %s", err)
	}

	ttime := time.Date(2025, 1, 17, 13, 0, 0, 0, loc)
	for _, s := range []Stats{
		{Name: "X", Total: 9, Streak: 8, Last: ttime},
		{Name: "Y", Total: 6, Streak: 1, Last: ttime},
		{Name: "Z", Total: 1, Streak: 1, Last: ttime},
	} {
		_, err := db.Update(s)
		if err != nil {
			t.Fatalf("error updating db: %s", err)
		}
	}

	for _, tt := range []struct {
		name string
		want Stats
	}{
		{
			name: "Get unexistent member",
			want: Stats{Name: "A"},
		},
		{
			name: "Get existing member",
			want: Stats{Name: "X", Total: 9, Streak: 8, Last: ttime},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.Get(tt.want.Name)
			if err != nil {
				t.Fatalf("error getting stats: %s", err)
			}
			// time.Time objects can't be compared without ensuring they
			// share location (same exact location object)
			got.Last = got.Last.In(tt.want.Last.Location())
			if got != tt.want {
				t.Fatal("stats differ")
			}
		})
	}
}
