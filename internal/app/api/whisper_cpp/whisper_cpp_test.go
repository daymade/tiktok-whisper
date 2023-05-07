package whisper_cpp

import (
	"strings"
	"testing"
)

func TestLocalTranscriber_Transcript(t *testing.T) {
	type fields struct {
		binaryPath string
		modelPath  string
	}
	type args struct {
		inputFilePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "large",
			fields: fields{
				binaryPath: "/Users/tiansheng/workspace/cpp/whisper.cpp/main",
				modelPath:  "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin",
			},
			args: args{
				inputFilePath: "/Users/tiansheng/workspace/go/tiktok-whisper/test/data/jfk.wav",
			},
			want:    "And so my fellow Americans, ask not what your country can do for you, ask what you can do for your country!",
			wantErr: false,
		},
		{
			name: "large-mp3",
			fields: fields{
				binaryPath: "/Users/tiansheng/workspace/cpp/whisper.cpp/main",
				modelPath:  "/Users/tiansheng/workspace/cpp/whisper.cpp/models/ggml-large-v2.bin",
			},
			args: args{
				inputFilePath: "/Users/tiansheng/workspace/go/tiktok-whisper/test/data/test.mp3",
			},
			want:    "星巴克",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lt := &LocalTranscriber{
				binaryPath: tt.fields.binaryPath,
				modelPath:  tt.fields.modelPath,
			}
			got, err := lt.Transcript(tt.args.inputFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transcript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("Transcript() got = %v, want %v", got, tt.want)
			}
		})
	}
}
