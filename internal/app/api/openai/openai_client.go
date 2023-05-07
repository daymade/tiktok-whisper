package openai

import (
	"github.com/sashabaranov/go-openai"
	"os"
	"sync"
)

var (
	once      sync.Once
	singleton *openai.Client
)

func GetClient() *openai.Client {
	once.Do(func() {
		token, ok := os.LookupEnv("OPENAI_API_KEY")
		if !ok {
			panic("OPENAI_API_KEY environment variable not set")
		}
		singleton = openai.NewClient(token)
	})

	return singleton
}
