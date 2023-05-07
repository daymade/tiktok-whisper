package whisper

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
)

// RemoteTranscriber implements remote transcription using the OpenAI API.
type RemoteTranscriber struct {
	client *openai.Client
}

// NewRemoteTranscriber creates a new RemoteTranscriber instance.
func NewRemoteTranscriber(client *openai.Client) *RemoteTranscriber {
	return &RemoteTranscriber{client: client}
}

// Transcript uses the OpenAI API for remote transcription.
func (rt *RemoteTranscriber) Transcript(inputFilePath string) (string, error) {
	ctx := context.Background()

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: inputFilePath,
	}
	resp, err := rt.client.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("createTranscription failed: %s", err)
	}

	return resp.Text, nil
}
