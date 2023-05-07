package convert

import (
	"github.com/spf13/cobra"
	"tiktok-whisper/internal/app"
)

var userNickname string
var videoDir string

func init() {
	Cmd.Flags().StringVarP(&userNickname, "userNickname", "n", "",
		"Which user owns the videos, this parameter affects the 'user' field when they are saved to the database")
	Cmd.Flags().StringVarP(&videoDir, "videoDir", "v", "",
		"videoDir specifies the mp4 file directory, example: . /test/data/mp4")

	Cmd.MarkFlagRequired("userNickname")
	Cmd.MarkFlagRequired("videoDir")
}

// Cmd represents the convert command
var Cmd = &cobra.Command{
	Use:   "convert",
	Short: "Start converting the video files in the specified directory to text",
	Long: `Start converting the video files in the specified directory to text

- Iterate through the mp4 files in the specified directory
- Convert to mp3 or wav and convert to text
- Support openai whisper or native whisper.cpp as conversion engine`,
	Run: func(cmd *cobra.Command, args []string) {
		converter := app.InitializeConverter()
		defer converter.Close()

		converter.Do(
			userNickname,
			videoDir,
			500,
		)
	},
}
