package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   string
	Commit    string
	Branch    string
	BuildTime string
	BuiltBy   string
)

func PrintVersion(_ *cobra.Command, _ []string) {
	fmt.Printf("Version: %s\n"+
		"Commit: %s\n"+
		"Build Time: %s\n",
		Version,
		Commit,
		BuildTime,
	)
}
