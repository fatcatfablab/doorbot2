package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	pb "github.com/schollz/progressbar/v3"
)

const (
	sqlite3Driver = "sqlite3"
	mysqlDriver   = "mysql"
)

var (
	srcDsn string
	dstDsn string

	tz = "America/New_York"
)

type rowMigrator func(*sql.Rows) error

func init() {
	flag.StringVar(&srcDsn, "src-dsn", "", "Source DSN")
	flag.StringVar(&dstDsn, "dst-dsn", "", "Dest DSN")
	flag.StringVar(&tz, "tz", tz, "Time Zone")
}

func main() {
	flag.Parse()
	if srcDsn == "" || dstDsn == "" {
		fmt.Fprintf(os.Stderr, "src-dsn and dst-dsn are required")
		os.Exit(1)
	}

	src, err := sql.Open(sqlite3Driver, srcDsn)
	if err != nil {
		panic(err)
	}
	defer src.Close()

	dstDsn, err = mysqlDsn(dstDsn)
	if err != nil {
		panic(err)
	}

	dst, err := sql.Open(mysqlDriver, dstDsn)
	if err != nil {
		panic(err)
	}
	defer dst.Close()

	if err := runMigration(src, dst, "history", migrateHistoryRow); err != nil {
		panic(err)
	}

	if err := runMigration(src, dst, "stats", migrateStatsRow); err != nil {
		panic(err)
	}
}

func mysqlDsn(dsn string) (string, error) {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", err
	}

	c.Loc = loc
	c.ParseTime = true

	return c.FormatDSN(), nil
}

func runMigration(src, dst *sql.DB, name string, migrateRow func(*sql.Tx) rowMigrator) error {
	tx, err := dst.Begin()
	if err != nil {
		return err
	}

	if err := migrateTable(src, name, migrateRow(tx)); err != nil {
		rberr := tx.Rollback()
		if rberr != nil {
			return rberr
		}
		return err
	}

	log.Printf("Running commit for %q", name)
	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Printf("Done %q", name)
	return nil
}

func migrateTable(s *sql.DB, table string, f rowMigrator) error {
	log.Printf("Migrating table %q", table)

	var num int64
	count := s.QueryRow("SELECT COUNT(*) FROM " + table)
	err := count.Scan(&num)
	if err != nil {
		return err
	}

	bar := pb.Default(num)

	rows, err := s.Query("SELECT * FROM " + table)
	if err != nil {
		return err
	}

	for rows.Next() {
		bar.Add(1)
		if err := f(rows); err != nil {
			return err
		}
	}

	return rows.Err()
}

func migrateHistoryRow(tx *sql.Tx) rowMigrator {
	insert, err := tx.Prepare(
		"INSERT INTO history (ts, name, access_granted) VALUES (?, ?, ?)",
	)
	if err != nil {
		panic(err)
	}

	return func(rows *sql.Rows) error {
		var t int64
		var name string
		var granted bool

		if err := rows.Scan(&t, &name, &granted); err != nil {
			return err
		}

		if _, err := insert.Exec(time.Unix(t, 0), name, granted); err != nil {
			return err
		}

		return nil
	}
}

func migrateStatsRow(tx *sql.Tx) rowMigrator {
	insert, err := tx.Prepare(
		"INSERT INTO stats (name, total, streak, last) VALUES (?, ?, ?, ?)",
	)
	if err != nil {
		panic(err)
	}

	return func(rows *sql.Rows) error {
		var name string
		var total int
		var streak int
		var last int64

		if err := rows.Scan(&name, &total, &streak, &last); err != nil {
			return err
		}

		t := time.Unix(last, 0)
		if _, err := insert.Exec(
			name, total, streak, t,
		); err != nil {
			return fmt.Errorf("error inserting stat (%s, %d, %d, %s): %s", name, total, streak, t, err)
		}

		return nil
	}
}
