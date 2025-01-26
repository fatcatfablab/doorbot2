package sender

import (
	"fmt"
	"log"
	"testing"

	"github.com/fatcatfablab/doorbot2/types"
)

const (
	name = "Johnny Melavo"
)

func TestStatsToString(t *testing.T) {
	for _, tt := range []struct {
		name  string
		stats types.Stats
		want  string
	}{
		{
			name:  "First visit",
			stats: types.Stats{Name: name, Total: 1, Streak: 1},
			want: fmt.Sprintf(
				"%s %s %d %s %d",
				name, ":fatcat:", 1, ":cat2:", 1,
			),
		},
		{
			name:  "5 streak",
			stats: types.Stats{Name: name, Total: 5, Streak: 5},
			want: fmt.Sprintf(
				"%s %s %d %s %d\nOne dedicated cat!",
				name, ":fatcat:", 5, ":black_cat:", 5,
			),
		},
		{
			name:  "UNO medal",
			stats: types.Stats{Name: name, Total: 7, Streak: 2},
			want: fmt.Sprintf(
				"%s %s %d %s %d\n:tada: Achievement unlocked! You get the UNO medal: :fatcat-yellow:",
				name, ":fatcat-yellow:", 7, ":cat2:", 2,
			),
		},
		{
			name:  "medal and streak achievements",
			stats: types.Stats{Name: name, Total: 31, Streak: 14},
			want: fmt.Sprintf(
				"%s %s %d %s %d"+
					"\n:tada: Achievement unlocked! You get the TEENSY medal: :fatcat-green:"+
					"\nLab cat to lab rat!",
				name, ":fatcat-green:", 31, ":rat:", 14,
			),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := statsToString(tt.stats)
			if got != tt.want {
				log.Printf("want: %s", tt.want)
				log.Printf("got : %s", got)
				t.Error("strings differ")
			}
		})
	}

}
