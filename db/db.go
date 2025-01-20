package db

import (
	"context"
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
) STRICT;
CREATE TABLE IF NOT EXISTS history (
               timestamp INTEGER NOT NULL,
               name TEXT NOT NULL,
               access_granted INTEGER NOT NULL,
               PRIMARY KEY (timestamp, name)
) STRICT;`
)

// This is the common interface between a *sql.DB and a *sql.Tx used here,
// so methods can seamlessly work with either
type dbh interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type dbKey struct{}

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

func (db *DB) Update(ctx context.Context, r Stats) (Stats, error) {
	_, err := db.getDbh(ctx).ExecContext(
		ctx,
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

func (db *DB) Bump(ctx context.Context, name string) (Stats, error) {
	return db.bumpWithTimestamp(ctx, name, time.Now())
}

func (db *DB) bumpWithTimestamp(ctx context.Context, name string, ts time.Time) (Stats, error) {
	r, err := db.Get(ctx, name)
	if err != nil {
		return Stats{}, fmt.Errorf("error retrieving record: %w", err)
	}

	r, err = db.Update(ctx, bumpStats(r, ts))
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

func (db *DB) Get(ctx context.Context, name string) (Stats, error) {
	row := db.getDbh(ctx).QueryRowContext(
		ctx,
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

func (db *DB) AddRecord(ctx context.Context, r AccessRecord) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting tx: %w", err)
	}
	defer func() {
		if err != nil {
			rerr := tx.Rollback()
			if rerr != nil {
				err = errors.Join(err, rerr)
			}
		}
	}()

	ctx = context.WithValue(ctx, dbKey{}, tx)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO history(timestamp, name, access_granted) VALUES (?, ?, ?)`+
			`ON CONFLICT(timestamp, name) DO UPDATE SET access_granted=?`,
		r.Timestamp.Unix(),
		r.Name,
		r.AccessGranted,
		r.AccessGranted,
	)

	_, err = db.bumpWithTimestamp(ctx, r.Name, r.Timestamp)
	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return err
}

func (db *DB) getDbh(ctx context.Context) dbh {
	tx := ctx.Value(dbKey{})
	if tx == nil {
		return db.db
	}
	return tx.(dbh)
}
