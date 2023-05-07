package export

import (
	"fmt"
	"github.com/tealeg/xlsx"
	"log"
	"tiktok-whisper/internal/app/model"
	"time"
)

func ToExcel(transcriptions []model.Transcription, outputFilePath string) {
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Transcriptions")
	if err != nil {
		log.Fatal(err)
	}

	headerRow := sheet.AddRow()
	headerRow.AddCell().Value = "ID"
	headerRow.AddCell().Value = "User"
	headerRow.AddCell().Value = "Last Conversion Time"
	headerRow.AddCell().Value = "MP3 File Name"
	headerRow.AddCell().Value = "Audio Duration"
	headerRow.AddCell().Value = "Transcription"
	headerRow.AddCell().Value = "Error Message"

	for _, t := range transcriptions {
		row := sheet.AddRow()
		row.AddCell().Value = fmt.Sprint(t.ID)
		row.AddCell().Value = t.User
		row.AddCell().Value = t.LastConversionTime.Format(time.RFC3339)
		row.AddCell().Value = t.Mp3FileName
		row.AddCell().Value = fmt.Sprintf("%.2f", t.AudioDuration)
		row.AddCell().Value = t.Transcription
		row.AddCell().Value = t.ErrorMessage
	}

	err = file.Save(outputFilePath)
	if err != nil {
		log.Fatal(err)
	}
}
