package cmd

import (
	"github.com/fatcatfablab/doorbot2/version"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run:   version.PrintVersion,
	})
}
