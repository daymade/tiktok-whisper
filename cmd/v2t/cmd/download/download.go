package download

import (
	"github.com/spf13/cobra"
	"tiktok-whisper/cmd/v2t/cmd/download/xiaoyuzhou"
)

func init() {
	Cmd.AddCommand(xiaoyuzhou.Cmd)
}

// Cmd represents the export command
var Cmd = &cobra.Command{
	Use:   "download",
	Short: "Download podcasts from Small Universe or tiktok(unsupported now)",
	Long:  `Download podcasts from Small Universe or tiktok(unsupported now), support downloading all shows from the home page and single downloads`,
}
