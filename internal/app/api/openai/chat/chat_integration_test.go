//go:build integration
// +build integration

package chat

import (
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestChat_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration tests")
	}

	type args struct {
		text string
	}
	tests := []struct {
		name    string
		args    args
		want    openai.ChatCompletionResponse
		wantErr bool
	}{
		{
			name: "hello",
			args: args{
				text: "hello, who are you?",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Chat(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}