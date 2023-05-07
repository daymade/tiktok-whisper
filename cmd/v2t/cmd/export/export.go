package export

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"path/filepath"
	"tiktok-whisper/internal/app/converter/export"
	"tiktok-whisper/internal/app/repository/sqlite"
	"tiktok-whisper/internal/app/util/files"
)

var userNickname string
var outputFilePath string

func init() {
	Cmd.Flags().StringVarP(&userNickname, "userNickname", "n", "", "set userNickname")
	Cmd.Flags().StringVarP(&outputFilePath, "outputFilePath", "o", "", "set outputFilePath")

	Cmd.MarkFlagRequired("userNickname")
	Cmd.MarkFlagRequired("outputFilePath")
}

// Cmd represents the export command
var Cmd = &cobra.Command{
	Use:   "export",
	Short: "Export the specified user's text to excel",
	Long: `Export the specified user's text to excel

- Export all the user's text to excel, currently does not support a limited number`,
	Run: func(cmd *cobra.Command, args []string) {
		projectRoot, err := files.GetProjectRoot()
		if err != nil {
			log.Fatalf("Failed to get project root: %v\n", err)
		}

		dbPath := filepath.Join(projectRoot, "data/transcription.db")
		db := sqlite.NewSQLiteDB(dbPath)

		transcriptions, err := db.GetAllByUser(userNickname)
		if err != nil {
			log.Fatal(err)
		}

		export.ToExcel(transcriptions, outputFilePath)
		fmt.Printf("export finished, exported file path: %v\n", outputFilePath)
	},
}
