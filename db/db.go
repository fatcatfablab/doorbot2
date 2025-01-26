package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fatcatfablab/doorbot2/types"
	_ "github.com/mattn/go-sqlite3"
)

const (
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
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type dbKey struct{}

type date struct {
	year  int
	month time.Month
	day   int
}

type DB struct {
	db  *sql.DB
	loc *time.Location
}

func newDate(year int, month time.Month, day int) date {
	return date{year: year, month: month, day: day}
}

func New(path, tz string) (*DB, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("error loading tz %q: %w", tz, err)
	}

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

	return &DB{db: db, loc: loc}, nil
}

func initializeDb(db *sql.DB) error {
	_, err := db.Exec(initStmt)
	return err
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) Update(ctx context.Context, r types.Stats) (types.Stats, error) {
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

func (db *DB) bumpWithTimestamp(ctx context.Context, name string, ts time.Time) (types.Stats, bool, error) {
	lastStats, err := db.Get(ctx, name)
	if err != nil {
		return types.Stats{}, false, fmt.Errorf("error retrieving record: %w", err)
	}

	newStats, err := db.Update(ctx, bumpStats(lastStats, ts))
	if err != nil {
		return types.Stats{}, false, fmt.Errorf("error updating record: %w", err)
	}

	return newStats, newStats.Total != lastStats.Total, nil
}

func bumpStats(r types.Stats, ts time.Time) types.Stats {
	if r.Last.IsZero() {
		r.Total = 1
		r.Streak = 1
	} else {
		lastVisit := newDate(r.Last.Date())
		//log.Printf("lastVisit: %+v", lastVisit)
		thisVisit := newDate(ts.Date())
		//log.Printf("thisVisit: %+v", thisVisit)
		if thisVisit != lastVisit {
			// This is a different day from the last visit, so bump the total
			r.Total += 1
		}

		dayBefore := newDate(ts.Add(-24 * time.Hour).Date())
		//log.Printf("dayBefore: %+v", dayBefore)
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

func (db *DB) Get(ctx context.Context, name string) (types.Stats, error) {
	row := db.getDbh(ctx).QueryRowContext(
		ctx,
		"SELECT name, total, streak, last FROM stats WHERE name = ?",
		name,
	)

	var r types.Stats
	var ts int64
	if err := row.Scan(&r.Name, &r.Total, &r.Streak, &ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Name = name
		} else {
			return types.Stats{}, fmt.Errorf("error scanning row: %w", err)
		}
	} else {
		r.Last = time.Unix(ts, 0)
	}

	return r, nil
}

func (db *DB) AddRecord(ctx context.Context, r types.AccessRecord) (s types.Stats, bumped bool, err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return types.Stats{}, false, fmt.Errorf("error starting tx: %w", err)
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

	bumped = false
	if r.AccessGranted {
		s, bumped, err = db.bumpWithTimestamp(ctx, r.Name, r.Timestamp)
	} else {
		s, err = db.Get(ctx, r.Name)
	}

	err = tx.Commit()
	if err != nil {
		err = fmt.Errorf("error commiting tx: %w", err)
	}
	return s, bumped, err
}

func (db *DB) getDbh(ctx context.Context) dbh {
	tx := ctx.Value(dbKey{})
	if tx == nil {
		return db.db
	}
	return tx.(dbh)
}

func (db *DB) Loc() *time.Location {
	return db.loc
}

func (db *DB) DumpHistory(ctx context.Context, name string) ([]types.AccessRecord, error) {
	rows, err := db.getDbh(ctx).QueryContext(
		ctx,
		"SELECT timestamp, name, access_granted FROM history WHERE name=? ORDER BY timestamp ASC",
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("error dumping history: %w", err)
	}

	result := make([]types.AccessRecord, 0)
	for rows.Next() {
		var r types.AccessRecord
		var ts int64
		err = rows.Scan(&ts, &r.Name, &r.AccessGranted)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		r.Timestamp = time.Unix(ts, 0)
		result = append(result, r)
	}

	return result, nil
}

func (db *DB) Recompute(ctx context.Context, name string) (stats types.Stats, err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return types.Stats{}, fmt.Errorf("error starting tx: %w", err)
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
		"DELETE FROM stats WHERE name = ?",
		name,
	)
	if err != nil {
		return types.Stats{}, fmt.Errorf("can't delete stats: %w", err)
	}

	records, err := db.DumpHistory(ctx, name)
	for _, r := range records {
		if !r.AccessGranted {
			continue
		}
		stats, _, err = db.bumpWithTimestamp(ctx, r.Name, r.Timestamp)
		if err != nil {
			return types.Stats{}, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return types.Stats{}, fmt.Errorf("error commiting: %w", err)
	}

	return stats, nil
}
