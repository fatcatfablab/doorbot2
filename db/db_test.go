package db

import (
	"context"
	"log"
	"path"
	"testing"
	"time"

	"github.com/fatcatfablab/doorbot2/types"
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

	prevStats := types.Stats{
		Name: username,
	}
	for _, tt := range []struct {
		name string
		want types.Stats
	}{
		{
			name: "First bump",
			want: types.Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 0, 0, 0, 0, loc)},
		},
		{
			name: "Visit in the same day does not bump stats",
			want: types.Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2025, 1, 16, 1, 0, 0, 0, loc)},
		},
		{
			name: "Visit the next day bumps stats",
			want: types.Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 13, 0, 0, 0, loc)},
		},
		{
			name: "Visit the same day again does not bump stats",
			want: types.Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2025, 1, 17, 14, 0, 0, 0, loc)},
		},
		{
			name: "Visit the next day bumps stats again",
			want: types.Stats{Name: username, Total: 3, Streak: 3, Last: time.Date(2025, 1, 18, 13, 0, 0, 0, loc)},
		},
		{
			name: "Visit on a later date breaks streak, but bumps total",
			want: types.Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2025, 1, 26, 12, 0, 0, 0, loc)},
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
	for _, s := range []types.Stats{
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
		want types.Stats
	}{
		{
			name: "Get unexistent member",
			want: types.Stats{Name: "A"},
		},
		{
			name: "Get existing member",
			want: types.Stats{Name: "X", Total: 9, Streak: 8, Last: ttime},
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
		record types.AccessRecord
		bumped bool
		want   types.Stats
	}{
		{
			name:   "Add record 1",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 1, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: true,
			want:   types.Stats{Name: username, Total: 1, Streak: 1, Last: time.Date(2020, 1, 1, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Next day",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 2, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: true,
			want:   types.Stats{Name: username, Total: 2, Streak: 2, Last: time.Date(2020, 1, 2, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Continue streak",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 3, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: true,
			want:   types.Stats{Name: username, Total: 3, Streak: 3, Last: time.Date(2020, 1, 3, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Break streak",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 7, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: true,
			want:   types.Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Same day",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 7, 13, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: false,
			want:   types.Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 13, 0, 0, 0, loc)},
		},
		{
			name:   "Same day later",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 7, 14, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: false,
			want:   types.Stats{Name: username, Total: 4, Streak: 1, Last: time.Date(2020, 1, 7, 14, 0, 0, 0, loc)},
		},
		{
			name:   "Continue streak again",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 8, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
			bumped: true,
			want:   types.Stats{Name: username, Total: 5, Streak: 2, Last: time.Date(2020, 1, 8, 12, 0, 0, 0, loc)},
		},
		{
			name:   "Access not granted doesn't bump stats",
			record: types.AccessRecord{Timestamp: time.Date(2020, 1, 9, 12, 0, 0, 0, loc), Name: username, AccessGranted: false},
			bumped: false,
			want:   types.Stats{Name: username, Total: 5, Streak: 2, Last: time.Date(2020, 1, 8, 12, 0, 0, 0, loc)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, bumped, err := db.AddRecord(ctx, tt.record)
			if err != nil {
				t.Fatalf("error adding record: %s", err)
			}

			if bumped != tt.bumped {
				t.Error("wrong bump detection")
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

func TestDumpHistory(t *testing.T) {
	ctx := context.Background()
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-dumpHistory.sqlite"), tz)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	loc := db.loc
	want := []types.AccessRecord{
		{Timestamp: time.Date(2020, 1, 1, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 2, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 3, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 13, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 14, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 8, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 9, 12, 0, 0, 0, loc), Name: username, AccessGranted: false},
	}
	for _, r := range want {
		_, _, err := db.AddRecord(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error adding record: %s", err)
		}
	}

	got, err := db.DumpHistory(ctx, "some unknown username")
	if len(got) != 0 {
		t.Errorf("unexpected history for unknown username")
	}

	got, err = db.DumpHistory(ctx, username)
	if len(got) != len(want) {
		t.Errorf("different history lengths")
	}

	for i := range len(got) {
		// Need to make sure the location objects are the same one in order to
		// compare.
		got[i].Timestamp = got[i].Timestamp.In(loc)
		if got[i] != want[i] {
			log.Printf("want: %+v", want[i])
			log.Printf("got : %+v", got[i])
			t.Errorf("element in history differ")
		}
	}
}

func TestRecompute(t *testing.T) {
	ctx := context.Background()
	db, err := New(path.Join(t.TempDir(), "doorbot2-test-recompute.sqlite"), tz)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	loc := db.loc
	records := []types.AccessRecord{
		{Timestamp: time.Date(2020, 1, 1, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 2, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 3, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 13, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 7, 14, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 8, 12, 0, 0, 0, loc), Name: username, AccessGranted: true},
		{Timestamp: time.Date(2020, 1, 9, 12, 0, 0, 0, loc), Name: username, AccessGranted: false},
	}
	for _, r := range records {
		_, _, err := db.AddRecord(ctx, r)
		if err != nil {
			t.Fatalf("unexpected error adding record: %s", err)
		}
	}

	want := types.Stats{
		Name:   username,
		Total:  5,
		Streak: 2,
		Last:   time.Date(2020, 1, 8, 12, 0, 0, 0, loc),
	}

	got, err := db.Recompute(ctx, username)
	if err != nil {
		t.Errorf("unexpected error calling recompute: %s", err)
	}

	got.Last = got.Last.In(loc)
	if got != want {
		log.Printf("want: %+v", want)
		log.Printf("got : %+v", got)
		t.Errorf("stats differ")
	}
}
