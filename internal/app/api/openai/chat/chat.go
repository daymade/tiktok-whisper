package chat

import (
	"context"
	"github.com/sashabaranov/go-openai"
	openai2 "tiktok-whisper/internal/app/api/openai"
)

func Chat(text string) (openai.ChatCompletionResponse, error) {
	client := openai2.GetClient()
	ctx := context.Background()

	request := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: text,
			},
		},
	}
	resp, err := client.CreateChatCompletion(ctx, request)
	return resp, err
}
