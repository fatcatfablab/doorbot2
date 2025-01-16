package db

import (
	"path"
	"testing"
	"time"
)

const (
	username = "Johnny Melavo"
)

func (r AccessRecord) Equal(x AccessRecord) bool {
	return r.Name == x.Name && r.Total == x.Total && r.Streak == x.Streak && r.Timestamp == x.Timestamp
}

func TestBump(t *testing.T) {
	db, err := New(path.Join(t.TempDir(), "test-bump.sqlite"))
	if err != nil {
		t.Fatalf("error opening database: %s", err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("error loading timezone: %s", err)
	}
	originTs := time.Date(2025, 1, 16, 0, 0, 0, 0, loc)

	for _, tt := range []struct {
		name string
		want AccessRecord
	}{
		{
			name: "First bump",
			want: AccessRecord{Name: username, Total: 1, Streak: 1, Timestamp: originTs},
		},
		{
			name: "Bump in the same day",
			want: AccessRecord{Name: username, Total: 1, Streak: 1, Timestamp: originTs.Add(1 * time.Hour)},
		},
		{
			name: "Bump the next day",
			want: AccessRecord{Name: username, Total: 2, Streak: 2, Timestamp: originTs.Add(24 * time.Hour)},
		},
		{
			name: "Bump next week",
			want: AccessRecord{Name: username, Total: 3, Streak: 1, Timestamp: originTs.Add(7 * 24 * time.Hour)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.bumpWithTimestamp(username, tt.want.Timestamp)
			if err != nil {
				t.Fatalf("error bumping username: %s", err)
			}

			if !got.Equal(tt.want) {
				t.Errorf("records do not match. Got: %v Want: %v", got, tt.want)
			}
		})
	}
}
