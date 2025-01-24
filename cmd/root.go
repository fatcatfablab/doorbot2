package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/spf13/cobra"
)

var dbPath string
var accessDb *db.DB

var rootCmd = &cobra.Command{
	Use:   "doorbot2",
	Short: "Doorbot2 announces arrivals to a slack channel",
	Long: `Doorbot2 acts as a UniFi Access webhook endpoint.
		   	When it receives an access message, stores it, calculates
			some stats, and posts to a configured slack channel`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		accessDb, err = db.New(dbPath, tz)
		if err != nil {
			log.Fatalf("error opening database: %s", err)
		}
		return err
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return accessDb.Close()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "dbPath", "access.sqlite", "Path to the sqlite3 database")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
