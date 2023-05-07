package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"tiktok-whisper/cmd/v2t/cmd/config"
	"tiktok-whisper/cmd/v2t/cmd/convert"
	"tiktok-whisper/cmd/v2t/cmd/export"
	"tiktok-whisper/cmd/v2t/cmd/version"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cmd",
	Short: "An application for batch converting video to text, supports tiktok and other video sites",
	Long: `An application for batch converting video to text, supports tiktok and other video sites or local video.
- First download all videos to local machine
- Call v2t to batch process the videos with local folder path
- The processed records will be saved to sqlite.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(export.Cmd)
	rootCmd.AddCommand(config.Cmd)
	rootCmd.AddCommand(convert.Cmd)
	rootCmd.AddCommand(version.Cmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cmd.yaml)")
}
