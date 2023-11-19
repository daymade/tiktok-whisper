package convert

import (
	"github.com/spf13/cobra"
	"strings"
	"tiktok-whisper/internal/app"
)

var userNickname string
var directory string
var outputDirectory string
var fileExtension string
var video bool
var audio bool
var convertCount int
var parallel int

var inputFile string

func init() {
	Cmd.Flags().StringVarP(&userNickname, "userNickname", "u", "",
		"Which user owns the videos, this parameter affects the 'user' field when they are saved to the database")
	Cmd.Flags().StringVarP(&directory, "directory", "d", "",
		"Specifies the mp4 file directory, example: ./test/data/mp4")
	Cmd.Flags().StringVarP(&outputDirectory, "outputDirectory", "o", "./data/transcription",
		"Specifies the transcriptions directory, example: ./test/data/transcription")
	Cmd.Flags().IntVarP(&convertCount, "convertCount", "n", 1,
		"How many files to convert from the directory this time")
	Cmd.Flags().IntVarP(&parallel, "parallel", "p", 1,
		"How many files to convert at the same time")

	Cmd.Flags().StringVarP(&inputFile, "input", "i", "",
		"Specifies the audio file to convert, example: . /test/data/test.mp3")

	Cmd.Flags().StringVarP(&fileExtension, "type", "t", "",
		"When converting the specified directory, you can use this option to filter the files with the specified extension, example: mp3")

	Cmd.Flags().BoolVarP(&video, "video", "v", false,
		"Convert video to text")

	Cmd.Flags().BoolVarP(&audio, "audio", "a", false,
		"Convert audio to text")
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
		if !video && !audio {
			cmd.PrintErrf("Please specify the conversion type, -v or -a\n")
			cmd.Help()
			return
		}

		if video && audio {
			cmd.PrintErrf("Please specify the conversion type, -v or -a\n")
			cmd.Help()
			return
		}

		if directory == "" && inputFile == "" {
			cmd.PrintErrf("Please specify the directory or file to convert\n")
			cmd.Help()
			return
		}

		if directory != "" && inputFile != "" {
			cmd.PrintErrf("Please specify the directory or file to convert\n")
			cmd.Help()
			return
		}

		if video {
			if directory != "" && userNickname == "" {
				cmd.PrintErrf("UserNickName must be set when converting video in directory\n")
				cmd.Help()
				return
			}

			if fileExtension == "" {
				fileExtension = "mp4"
			}
		}

		if audio {
			if fileExtension == "" {
				fileExtension = "mp3"
			}
		}

		converter := app.InitializeConverter()
		defer converter.Close()

		if video {
			converter.ConvertVideoDir(
				userNickname,
				directory,
				fileExtension,
				convertCount,
				parallel,
			)
		} else if audio {
			if directory != "" {
				err := converter.ConvertAudioDir(
					directory, 
					fileExtension, 
					outputDirectory, 
					convertCount, 
					parallel,
				)
				if err != nil {
					cmd.PrintErrf("ConvertAudioDir error: %v\n", err)
					return
				}
			} else if inputFile != "" {
				err := converter.ConvertAudios(strings.Split(inputFile, ","), outputDirectory, parallel)
				if err != nil {
					cmd.PrintErrf("ConvertAudios error: %v\n", err)
					return
				}
			}
		}
	},
}
