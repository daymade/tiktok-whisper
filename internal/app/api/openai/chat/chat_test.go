package chat

import (
	"github.com/sashabaranov/go-openai"
	"testing"
)

func TestChat(t *testing.T) {
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
