package files

import "testing"

func TestGetProjectRoot(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "get_root",
			want:    "/Users/tiansheng/workspace/go/tiktok-whisper",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProjectRoot()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetProjectRoot() got = %v, want %v", got, tt.want)
			}
		})
	}
}
