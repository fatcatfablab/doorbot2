package types

import (
	"context"
	"time"
)

type Sender interface {
	Post(ctx context.Context, s Stats) error
}

type Stats struct {
	Name   string    `json:"name"`
	Total  uint      `json:"total"`
	Streak uint      `json:"streak"`
	Last   time.Time `json:"last"`
}

type AccessRecord struct {
	Timestamp     time.Time `json:"timestamp"`
	Name          string    `json:"name"`
	AccessGranted bool      `json:"access_granted"`
}
