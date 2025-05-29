package cmd

import (
	"fmt"
	"os"

	"github.com/fatcatfablab/doorbot2/db"
	"github.com/spf13/cobra"
)

var dsn string
var accessDb *db.DB

var rootCmd = &cobra.Command{
	Use:   "doorbot2",
	Short: "Doorbot2 announces arrivals to a slack channel",
	Long: "Doorbot2 acts as a UniFi Access webhook endpoint.\n" +
		"When it receives an access message, stores it, calculates " +
		"some stats, and posts to a configured slack channel",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", os.Getenv("DOORBOT2_DSN"), "DSN for the mysql database")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
