package convert

import (
	"math"
	"strings"
	"tiktok-whisper/internal/app"
	"tiktok-whisper/internal/app/converter"
	"tiktok-whisper/internal/app/api/provider"

	"github.com/spf13/cobra"
)

var userNickname string
var directory string
var outputDirectory string
var fileExtension string
var video bool
var audio bool
var convertCount int
var parallel int
var noProgress bool
var providerName string

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

	Cmd.Flags().BoolVar(&noProgress, "no-progress", false,
		"Disable progress bar display")
	
	Cmd.Flags().StringVar(&providerName, "provider", "",
		"Transcription provider to use (whisper_cpp, openai, elevenlabs, whisper_server, etc.)")
	
}

// Cmd represents the convert command
var Cmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert video or audio files to text",
	Long: `Convert video or audio files in a directory to text transcriptions

- Process mp4/mp3 files in the specified directory
- Convert to appropriate format and transcribe to text
- Support OpenAI Whisper API or local whisper.cpp engine`,
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


		// Set runtime configuration for provider selection
		if providerName != "" {
			provider.InitializeRuntimeConfig()
			provider.SetRuntimeConfig(&provider.RuntimeConfig{
				ProviderName: providerName,
			})
		}
		
		progressConfig := converter.ProgressConfig{
			Enabled: !noProgress && converter.ShouldShowProgress(false),
			Writer:  nil, // Use default (stderr)
		}
		progressConverter := app.InitializeProgressAwareConverter(progressConfig)
		defer progressConverter.Close()

		if video {
			if directory != "" && userNickname == "" {
				cmd.PrintErrf("UserNickName must be set when converting video in directory\n")
				cmd.Help()
				return
			}

			if fileExtension == "" {
				fileExtension = "mp4"
			}

			if directory != "" {
				err := progressConverter.ConvertVideoDirWithProgress(
					userNickname,
					directory,
					fileExtension,
					convertCount,
					parallel,
				)
				if err != nil {
					cmd.PrintErrf("ConvertVideoDirWithProgress error: %v\n", err)
					return
				}
			} else if inputFile != "" {
				if userNickname == "" {
					userNickname = "default"
				}

				// set convert count to int max
				err := progressConverter.ConvertVideosWithProgress(strings.Split(inputFile, ","), userNickname, math.MaxInt, parallel)
				if err != nil {
					cmd.PrintErrf("ConvertVideosWithProgress error: %v\n", err)
					return
				}
			}

			return
		}

		if audio {
			if fileExtension == "" {
				fileExtension = "mp3"
			}

			if directory != "" {
				// Use userNickname if provided, otherwise use empty string
				if userNickname == "" {
					userNickname = "default"
				}
				err := progressConverter.ConvertAudioDirWithProgress(
					userNickname,
					directory,
					fileExtension,
					outputDirectory,
					convertCount,
					parallel,
				)
				if err != nil {
					cmd.PrintErrf("ConvertAudioDirWithProgress error: %v\n", err)
					return
				}
			} else if inputFile != "" {
				if userNickname == "" {
					userNickname = "default"
				}
				err := progressConverter.ConvertAudiosWithProgress(strings.Split(inputFile, ","), outputDirectory, userNickname, parallel)
				if err != nil {
					cmd.PrintErrf("ConvertAudiosWithProgress error: %v\n", err)
					return
				}
			}
			return
		}

		cmd.Help()
	},
}

