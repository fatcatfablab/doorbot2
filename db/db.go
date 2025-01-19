package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tz       = "America/New_York"
	driver   = "sqlite3"
	initStmt = `
CREATE TABLE IF NOT EXISTS stats (
       name TEXT NOT NULL,
       total INTEGER NOT NULL,
	   streak INTEGER NOT NULL,
	   last INTEGER NOT NULL,
       PRIMARY KEY (name)
) STRICT;`
)

type Stats struct {
	Name   string    `json:"name"`
	Total  uint      `json:"total"`
	Streak uint      `json:"streak"`
	Last   time.Time `json:"last"`
}

type date struct {
	year  int
	month time.Month
	day   int
}

type DB struct {
	db *sql.DB
}

func newDate(year int, month time.Month, day int) date {
	return date{year: year, month: month, day: day}
}

func New(path string) (*DB, error) {
	db, err := sql.Open(driver, path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open database: %w", err)
	}

	finfo, err := os.Stat(path)
	if err != nil || finfo.Size() == 0 {
		if err := initializeDb(db); err != nil {
			return nil, fmt.Errorf("error initializing database: %s", err)
		}
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("couldn't enable WAL mode: %w", err)
	}

	return &DB{db: db}, nil
}

func initializeDb(db *sql.DB) error {
	_, err := db.Exec(initStmt)
	return err
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) Update(r Stats) (Stats, error) {
	_, err := db.db.Exec(
		`INSERT INTO stats(name, total, streak, last) VALUES (?, ?, ?, ?)`+
			`ON CONFLICT(name) DO UPDATE SET total=?, streak=?, last=?`,
		r.Name,
		r.Total,
		r.Streak,
		r.Last.Unix(),
		r.Total,
		r.Streak,
		r.Last.Unix(),
	)
	return r, err
}

func (db *DB) Bump(name string) (Stats, error) {
	r, err := db.Get(name)
	if err != nil {
		return Stats{}, fmt.Errorf("error retrieving record: %w", err)
	}

	r, err = db.Update(bumpStats(r, time.Now()))
	if err != nil {
		return Stats{}, fmt.Errorf("error updating record: %w", err)
	}
	return r, nil
}

func bumpStats(r Stats, ts time.Time) Stats {
	if r.Last.IsZero() {
		r.Total = 1
		r.Streak = 1
	} else {
		lastVisit := newDate(r.Last.Date())
		log.Printf("lastVisit: %+v", lastVisit)
		thisVisit := newDate(ts.Date())
		log.Printf("thisVisit: %+v", thisVisit)
		if thisVisit != lastVisit {
			// This is a different day from the last visit, so bump the total
			r.Total += 1
		}

		dayBefore := newDate(ts.Add(-24 * time.Hour).Date())
		log.Printf("dayBefore: %+v", dayBefore)
		if lastVisit == dayBefore {
			// Last visit was the day before, so bump the streak
			r.Streak += 1
		} else if lastVisit != thisVisit {
			// Reset the streak
			r.Streak = 1
		}
	}

	r.Last = ts
	return r
}

func (db *DB) Get(name string) (Stats, error) {
	row := db.db.QueryRow(
		"SELECT name, total, streak, last FROM stats WHERE name = ?",
		name,
	)

	var r Stats
	var ts int64
	if err := row.Scan(&r.Name, &r.Total, &r.Streak, &ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Name = name
		} else {
			return Stats{}, fmt.Errorf("error scanning row: %w", err)
		}
	} else {
		r.Last = time.Unix(ts, 0)
	}

	return r, nil
}
