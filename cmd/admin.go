package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/spf13/cobra"
)

var (
	name string

	adminCmd = &cobra.Command{
		Use:   "admin",
		Short: "Admin actions on a doorbot2 database",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			accessDb, err = db.New(dsn, tz)
			if err != nil {
				log.Fatalf("error opening database: %s", err)
			}
			return err
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return accessDb.Close()
		},
	}

	dumpCmd = &cobra.Command{
		Use: "dump",
		Run: func(cmd *cobra.Command, args []string) {
			dump(accessDb, name)
		},
	}

	recomputeCmd = &cobra.Command{
		Use: "recompute",
		Run: func(cmd *cobra.Command, args []string) {
			recompute(accessDb, name)
		},
	}
)

func init() {
	adminCmd.PersistentFlags().StringVar(&name, "name", "", "Member name to act on")
	adminCmd.MarkFlagRequired("name")

	adminCmd.AddCommand(dumpCmd)
	adminCmd.AddCommand(recomputeCmd)

	rootCmd.AddCommand(adminCmd)
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
