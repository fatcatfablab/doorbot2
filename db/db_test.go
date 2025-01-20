package db

import (
	"context"
	"log"
	"path"
	"testing"
	"time"
)

const (
	username = "Johnny Melavo"
	tz       = "America/New_York"
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
	ctx := context.Background()
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-get.sqlite"), tz)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ttime := time.Date(2025, 1, 17, 13, 0, 0, 0, db.loc)
	for _, s := range []Stats{
		{Name: "X", Total: 9, Streak: 8, Last: ttime},
		{Name: "Y", Total: 6, Streak: 1, Last: ttime},
		{Name: "Z", Total: 1, Streak: 1, Last: ttime},
	} {
		_, err := db.Update(ctx, s)
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
			got, err := db.Get(ctx, tt.want.Name)
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

func TestBump(t *testing.T) {
	ctx := context.Background()
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-bump.sqlite"), tz)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ttime := time.Now().In(db.loc).Add(-24 * time.Hour)
	r := Stats{Name: username, Total: 9, Streak: 8, Last: ttime}

	_, err = db.Update(ctx, r)
	if err != nil {
		t.Fatalf("error running update: %s", err)
	}

	want := Stats{Name: username, Total: 10, Streak: 9, Last: time.Now().In(ttime.Location())}
	got, err := db.Bump(ctx, username)
	if err != nil {
		t.Fatalf("unexpected error calling bump: %s", err)
	}

	if got.Name != want.Name || got.Total != want.Total || got.Streak != want.Streak {
		log.Printf("want: %+v", want)
		log.Printf("got:  %+v", got)
		t.Error("stats differ")
	}

	if newDate(got.Last.In(db.loc).Date()) != newDate(want.Last.In(db.loc).Date()) {
		log.Printf("want: %+v", want)
		log.Printf("got:  %+v", got)
		t.Error("timestamps differ")
	}
}

func TestAddRecord(t *testing.T) {
	ctx := context.Background()
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-addEntry.sqlite"), tz)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	loc := db.loc
	for _, tt := range []struct {
		name   string
		record AccessRecord
		want   Stats
	}{
		{
			name:   "Add record 1",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 1, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2020, 1, 1, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Next day",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 2, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2020, 1, 2, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Continue streak",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 3, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 3, Streak: 3, Last: time.Date(2020, 1, 3, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Break streak",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 7, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Same day",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 7, 13, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 13, 0, 0, 0, loc)},
		},
		{
			name:   "Same day later",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 7, 14, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 14, 0, 0, 0, loc)},
		},
		{
			name:   "Continue streak again",
			record: AccessRecord{Timestamp: time.Date(2020, 1, 8, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			want:   Stats{Name: username, Total: 5, Streak: 2, Last: time.Date(2020, 1, 8, 12, 0, 0, 0, loc)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := db.AddRecord(ctx, tt.record)
			if err != nil {
				t.Fatalf("error adding record: %s", err)
			}

			got, err := db.Get(ctx, tt.record.Name)
			if err != nil {
				t.Fatalf("error getting stats: %s", err)
			}

			got.Last = got.Last.In(tt.want.Last.Location())
			if tt.want != got {
				log.Printf("want: %+v", tt.want)
				log.Printf("got : %+v", got)
				t.Error("stats differ")
			}
		})
	}
}
