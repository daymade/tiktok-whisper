package embedding

import (
	"context"
	"github.com/sashabaranov/go-openai"
	openai2 "tiktok-whisper/internal/app/api/openai"
)

func Embedding(text string) (openai.EmbeddingResponse, error) {
	client := openai2.GetClient()
	ctx := context.Background()

	request := openai.EmbeddingRequest{
		Model: openai.DavinciSimilarity,
		Input: []string{
			"text",
		},
	}
	resp, err := client.CreateEmbeddings(ctx, request)
	return resp, err
}
