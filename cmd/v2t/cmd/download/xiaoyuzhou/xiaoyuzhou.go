package xiaoyuzhou

import (
	"errors"
	"github.com/spf13/cobra"
	"log"
	"strings"
	"tiktok-whisper/internal/app/util/files"
	"tiktok-whisper/internal/downloader"
)

var downloadDir string
var podcast string
var episode string

func init() {
	Cmd.Flags().StringVarP(&downloadDir, "downloadDir", "d", "data/xiaoyuzhou", "set directory to save downloaded files ")
	Cmd.Flags().StringVarP(&podcast, "podcast", "p", "", "set podcast url, e.g. https://www.xiaoyuzhoufm.com/podcast/61a9f093ca6141933d1a1c63")
	Cmd.Flags().StringVarP(&episode, "episode", "e", "", "set episode, If it is more than one episode can be separated by a comma, e.g. https://www.xiaoyuzhoufm.com/episode/64411602a79cc81470055c96")
}

// Cmd represents the "download xiaoyuzhou" command
var Cmd = &cobra.Command{
	Use:   "xiaoyuzhou",
	Short: "Download podcasts from Small Universe",
	Long:  `Download podcasts from Small Universe, support downloading all shows from the home page and single downloads`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if podcast == "" && episode == "" {
			return errors.New("please input a podcast or an episode")
		}

		dir, err := files.GetAbsolutePath(downloadDir)
		if err != nil {
			log.Fatal(err)
		}

		// If podcast is not empty, download the podcast
		if podcast != "" {
			return downloader.DownloadPodcast(podcast, dir)
		}

		// otherwise download the episodes

		// The Split function returns a slice of the string split around each instance of the separator.
		// If the separator is not present in the string, the resulting slice has a single element,
		// which is the original string. So no need to check for a single episode and wrap it into a slice.
		episodeList := strings.Split(episode, ",")

		return downloader.BatchDownloadEpisodes(episodeList, dir)
	},
}
