package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

var (
	dbPath = flag.String("dbPath", "access.sqlite", "Path to the sqlite3 database")
	name   = flag.String("name", "", "Name of the member to act on")
	tz     = flag.String("tz", "America/New_York", "Time zone")
)

func main() {
	flag.Parse()

	if *name == "" {
		fmt.Println("--name required to be non empty")
		return
	}

	var err error
	accessDb, err := db.New(*dbPath, *tz)
	if err != nil {
		log.Fatalf("error opening database: %s", err)
	}
	defer accessDb.Close()

	switch flag.Arg(0) {
	case "dump":
		dump(accessDb, *name)
	case "recompute":
		recompute(accessDb, *name)
	}
}

func dump(accessDb *db.DB, name string) {
	records, err := accessDb.DumpHistory(context.Background(), name)
	if err != nil {
		log.Printf("error dumping history: %s", err)
	}

	for _, r := range records {
		granted := 1
		if !r.AccessGranted {
			granted = 0
		}
		fmt.Printf(
			"%s,%s,%s,%d\n",
			r.Timestamp.Format("01/02/2006"),
			r.Timestamp.Format(time.TimeOnly),
			r.Name,
			granted,
		)
	}
}

func recompute(accessDb *db.DB, name string) {
	s, err := accessDb.Recompute(context.Background(), name)
	if err != nil {
		log.Printf("error recomputing: %s", err)
	}

	fmt.Printf("%+v\n", s)
}
