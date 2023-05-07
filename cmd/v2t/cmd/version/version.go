package version

import (
	"fmt"
	"github.com/spf13/cobra"
)

var version = "v0.0.1"

// Cmd represents the version command
var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of video-to-text",
	Long:  `All software has versions. This is video-to-text's.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printVersion()
		return nil
	},
}

func printVersion() {
	fmt.Println(version)
}
