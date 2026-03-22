package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type captureProvider struct {
	lastRequest *TranscriptionRequest
}

func (p *captureProvider) Transcript(inputFilePath string) (string, error) {
	return "", nil
}

func (p *captureProvider) TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	p.lastRequest = request
	return &TranscriptionResponse{Text: "ok"}, nil
}

func (p *captureProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{Name: "capture", Type: ProviderTypeLocal}
}

func (p *captureProvider) ValidateConfiguration() error { return nil }
func (p *captureProvider) HealthCheck(ctx context.Context) error { return nil }

type healthCheckProbeProvider struct {
	lastCtx context.Context
}

func (p *healthCheckProbeProvider) Transcript(inputFilePath string) (string, error) {
	return "", nil
}

func (p *healthCheckProbeProvider) TranscriptWithOptions(ctx context.Context, request *TranscriptionRequest) (*TranscriptionResponse, error) {
	return &TranscriptionResponse{Text: "ok"}, nil
}

func (p *healthCheckProbeProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{Name: "probe", Type: ProviderTypeLocal}
}

func (p *healthCheckProbeProvider) ValidateConfiguration() error { return nil }

func (p *healthCheckProbeProvider) HealthCheck(ctx context.Context) error {
	p.lastCtx = ctx
	return nil
}

func TestExpandEnvValueSupportsShellDefaultSyntax(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("WHISPER_CPP_BINARY", "/custom/binary")

	if got := expandEnvValue("${WHISPER_CPP_BINARY:-./whisper.cpp/main}"); got != "/custom/binary" {
		t.Fatalf("expected env value to win, got %q", got)
	}

	if got := expandEnvValue("${OPENAI_API_KEY:-fallback-key}"); got != "fallback-key" {
		t.Fatalf("expected default fallback, got %q", got)
	}

	if got := expandEnvValue("prefix-${MISSING_VAR:-value}-suffix"); got != "prefix-value-suffix" {
		t.Fatalf("expected inline fallback expansion, got %q", got)
	}
}

func TestExpandEnvironmentVariablesExpandsProviderSettings(t *testing.T) {
	t.Setenv("WHISPER_CPP_BINARY", "")
	t.Setenv("WHISPER_CPP_MODEL", "")

	cfg := &ProviderConfiguration{
		Providers: map[string]ProviderConfig{
			ProviderNameWhisperCpp: {
				Type:    ProviderNameWhisperCpp,
				Enabled: true,
				Settings: map[string]interface{}{
					"binary_path": "${WHISPER_CPP_BINARY:-./whisper.cpp/main}",
					"model_path":  "${WHISPER_CPP_MODEL:-./whisper.cpp/models/ggml-large-v2.bin}",
				},
			},
		},
	}

	cm := NewConfigManager("unused")
	if err := cm.expandEnvironmentVariables(cfg); err != nil {
		t.Fatalf("expandEnvironmentVariables returned error: %v", err)
	}

	settings := cfg.Providers[ProviderNameWhisperCpp].Settings
	if got := settings["binary_path"]; got != "./whisper.cpp/main" {
		t.Fatalf("expected default binary path, got %#v", got)
	}
	if got := settings["model_path"]; got != "./whisper.cpp/models/ggml-large-v2.bin" {
		t.Fatalf("expected default model path, got %#v", got)
	}
}

func TestResolveProviderConfigPathPrefersHomeConfig(t *testing.T) {
	tmpDir := t.TempDir()
	localConfig := filepath.Join(tmpDir, "providers.yaml")
	homeDir := filepath.Join(tmpDir, "home")
	homeConfig := filepath.Join(homeDir, ".tiktok-whisper", "providers.yaml")

	if err := os.WriteFile(localConfig, []byte("local"), 0644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(homeConfig), 0755); err != nil {
		t.Fatalf("mkdir home config dir: %v", err)
	}
	if err := os.WriteFile(homeConfig, []byte("home"), 0644); err != nil {
		t.Fatalf("write home config: %v", err)
	}

	path, warning, err := resolveProviderConfigPath(localConfig, homeConfig)
	if err != nil {
		t.Fatalf("resolveProviderConfigPath returned error: %v", err)
	}
	if path != homeConfig {
		t.Fatalf("expected home config path, got %q", path)
	}
	if warning == "" {
		t.Fatal("expected warning when both configs exist")
	}
}

func TestSimpleProviderTranscriberUsesSelectedProviderSettings(t *testing.T) {
	provider := &captureProvider{}
	transcriber := &SimpleProviderTranscriber{
		provider:     provider,
		providerName: "alternate",
		config: &ProviderConfiguration{
			DefaultProvider: "default",
			Providers: map[string]ProviderConfig{
				"default": {
					Settings: map[string]interface{}{
						"language": "en",
						"prompt":   "default-prompt",
					},
				},
				"alternate": {
					Settings: map[string]interface{}{
						"language": "zh",
						"prompt":   "alternate-prompt",
					},
				},
			},
		},
	}

	if _, err := transcriber.Transcript("/tmp/audio.wav"); err != nil {
		t.Fatalf("Transcript returned error: %v", err)
	}

	if provider.lastRequest == nil {
		t.Fatal("provider did not receive request")
	}
	if provider.lastRequest.Language != "zh" {
		t.Fatalf("expected language from selected provider, got %q", provider.lastRequest.Language)
	}
	if provider.lastRequest.Prompt != "alternate-prompt" {
		t.Fatalf("expected prompt from selected provider, got %q", provider.lastRequest.Prompt)
	}
}

func TestSmartProviderSelectorUsesNonNilHealthCheckContext(t *testing.T) {
	registry := NewProviderRegistry()
	probe := &healthCheckProbeProvider{}
	if err := registry.RegisterProvider("probe", probe); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	selector := NewSmartProviderSelector(&OrchestratorConfig{
		HealthCheckInterval: time.Minute,
		RouterRules: RouterRules{
			ByLanguage: map[string]string{"zh": "probe"},
		},
	}, registry)

	provider, err := selector.SelectProvider(&TranscriptionRequest{Language: "zh"})
	if err != nil {
		t.Fatalf("SelectProvider returned error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider to be selected")
	}
	if probe.lastCtx == nil {
		t.Fatal("expected health check to receive non-nil context")
	}
}
