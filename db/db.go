package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fatcatfablab/doorbot2/types"
	"github.com/go-sql-driver/mysql"
)

const (
	driver      = "mysql"
	createStats = `
CREATE TABLE IF NOT EXISTS stats (
	name VARCHAR(255) NOT NULL,
	total INTEGER NOT NULL,
	streak INTEGER NOT NULL,
	last TIMESTAMP NOT NULL,
	PRIMARY KEY (name)
);`
	createHistory = `
CREATE TABLE IF NOT EXISTS history (
	ts TIMESTAMP NOT NULL,
	name VARCHAR(255) NOT NULL,
	access_granted BOOL NOT NULL,
	PRIMARY KEY (ts, name)
);`
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

func New(dsn, tz string) (*DB, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("error loading tz %q: %w", tz, err)
	}

	conf, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing dsn: %s", err)
	}

	// We rely on these being set, so set them here instead of expecting they've
	// been properly configured
	conf.Loc = loc
	conf.ParseTime = true

	db, err := sql.Open(driver, conf.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to db: %w", err)
	}

	d := &DB{db: db, loc: loc}
	if err := d.initialize(); err != nil {
		return nil, fmt.Errorf("error initializing db: %w", err)
	}

	return d, nil
}

func (db *DB) initialize() error {
	_, err1 := db.db.Exec(createStats)
	_, err2 := db.db.Exec(createHistory)
	return errors.Join(err1, err2)
}

func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) Update(ctx context.Context, r types.Stats) (types.Stats, error) {
	_, err := db.getDbh(ctx).ExecContext(
		ctx,
		`INSERT INTO stats(name, total, streak, last) VALUES (?, ?, ?, ?)`+
			`ON DUPLICATE KEY UPDATE total=?, streak=?, last=?`,
		r.Name,
		r.Total,
		r.Streak,
		r.Last,
		r.Total,
		r.Streak,
		r.Last,
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
	if err := row.Scan(&r.Name, &r.Total, &r.Streak, &r.Last); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Name = name
		} else {
			return types.Stats{}, fmt.Errorf("error scanning row: %w", err)
		}
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

	bumped = false
	ctx = context.WithValue(ctx, dbKey{}, tx)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO history(ts, name, access_granted) VALUES (?, ?, ?)`+
			`ON DUPLICATE KEY UPDATE access_granted=?`,
		r.Timestamp,
		r.Name,
		r.AccessGranted,
		r.AccessGranted,
	)
	if err != nil {
		return s, bumped, fmt.Errorf("error running insert: %w", err)
	}

	if r.AccessGranted {
		s, bumped, err = db.bumpWithTimestamp(ctx, r.Name, r.Timestamp)
		if err != nil {
			return s, bumped, fmt.Errorf("error calling bumpWithTimestamp: %w", err)
		}
	} else {
		s, err = db.Get(ctx, r.Name)
		if err != nil {
			return s, bumped, fmt.Errorf("error calling db.Get: %w", err)
		}
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
		"SELECT ts, name, access_granted FROM history WHERE name=? ORDER BY ts ASC",
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("error dumping history: %w", err)
	}

	result := make([]types.AccessRecord, 0)
	for rows.Next() {
		var r types.AccessRecord
		err = rows.Scan(&r.Timestamp, &r.Name, &r.AccessGranted)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
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
