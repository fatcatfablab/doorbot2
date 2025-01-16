package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	tz       = "America/New_York"
	driver   = "sqlite3"
	initStmt = `
CREATE TABLE IF NOT EXISTS records (
       name TEXT NOT NULL,
       total INTEGER NOT NULL,
	   streak INTEGER NOT NULL,
	   timestamp INTEGER NOT NULL,
       PRIMARY KEY (name)
) STRICT;`
)

type AccessRecord struct {
	Name      string    `json:"name"`
	Total     uint      `json:"total"`
	Streak    uint      `json:"streak"`
	Timestamp time.Time `json:"timestamp"`
}

type DB struct {
	db *sql.DB
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

func (db *DB) Update(r AccessRecord) (AccessRecord, error) {
	_, err := db.db.Exec(
		`INSERT INTO records(name, total, streak, timestamp) VALUES (?, ?, ?, ?)`+
			`ON CONFLICT(name) DO UPDATE SET total=?, streak=?, timestamp=?`,
		r.Name,
		r.Total,
		r.Streak,
		r.Timestamp.Unix(),
		r.Total,
		r.Streak,
		r.Timestamp.Unix(),
	)
	return r, err
}

func (db *DB) Bump(name string) (AccessRecord, error) {
	return db.bumpWithTimestamp(name, time.Now())
}

func (db *DB) bumpWithTimestamp(name string, ts time.Time) (AccessRecord, error) {
	r, err := db.Get(name)
	if err != nil {
		return AccessRecord{}, fmt.Errorf("error retrieving record: %w", err)
	}

	if r.Timestamp.IsZero() {
		r.Timestamp = ts
		r.Total = 1
		r.Streak = 1
	}

	rYear, rMonth, rDay := r.Timestamp.Date()
	tsYear, tsMonth, tsDay := ts.Date()
	if rYear != tsYear || rMonth != tsMonth || rDay != tsDay {
		// This is a different day from the last visit, so bump the total
		r.Total += 1
	}

	nYear, nMonth, nDay := ts.Add(-24 * time.Hour).Date()
	if rYear == nYear && rMonth == nMonth && rDay == nDay {
		// There's only a difference of 24h from the last visit, so bump the
		// streak
		r.Streak += 1
	} else {
		r.Streak = 1
	}

	r.Timestamp = ts

	r, err = db.Update(r)
	if err != nil {
		return AccessRecord{}, fmt.Errorf("error updating record: %w", err)
	}

	return r, nil
}

func (db *DB) Get(name string) (AccessRecord, error) {
	rows, err := db.db.Query(
		"SELECT name, total, streak, timestamp FROM records WHERE name = ?",
		name,
	)
	if err != nil {
		return AccessRecord{}, fmt.Errorf("error querying record: %w", err)
	}

	r := AccessRecord{
		Name: name,
	}
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&r.Name, &r.Total, &r.Streak, &ts); err != nil {
			return AccessRecord{}, fmt.Errorf("error scanning row: %w", err)
		}
		r.Timestamp = time.Unix(ts, 0)
		break
	}

	return r, nil
}
