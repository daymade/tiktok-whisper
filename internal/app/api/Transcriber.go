package api

// Transcriber defines a transcription interface for converting audio files to text.
type Transcriber interface {
	Transcript(inputFilePath string) (string, error)
}
