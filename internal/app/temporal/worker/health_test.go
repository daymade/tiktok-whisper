package main

import "testing"

func TestIsReadyRequiresTemporalProviderAndMinIO(t *testing.T) {
	base := &HealthStatus{
		Temporal:  ConnectionStatus{Connected: true, Endpoint: "temporal:7233"},
		MinIO:     ConnectionStatus{Connected: true, Endpoint: "minio:9000"},
		Providers: []ProviderStatus{{Name: "whisper_cpp", Available: true}},
	}
	if !isReady(base) {
		t.Fatalf("expected healthy status to be ready")
	}

	noProvider := &HealthStatus{
		Temporal:  ConnectionStatus{Connected: true, Endpoint: "temporal:7233"},
		MinIO:     ConnectionStatus{Connected: true, Endpoint: "minio:9000"},
		Providers: []ProviderStatus{{Name: "whisper_cpp", Available: false}},
	}
	if isReady(noProvider) {
		t.Fatalf("expected no-provider status to be not ready")
	}

	noMinio := &HealthStatus{
		Temporal:  ConnectionStatus{Connected: true, Endpoint: "minio:9000"},
		MinIO:     ConnectionStatus{Connected: false, Endpoint: "minio:9000"},
		Providers: []ProviderStatus{{Name: "whisper_cpp", Available: true}},
	}
	if isReady(noMinio) {
		t.Fatalf("expected missing minio connectivity to be not ready")
	}
}

func TestProbeTCPEndpointRejectsMissingEndpoint(t *testing.T) {
	ok, errMsg := probeTCPEndpoint("", 0)
	if ok {
		t.Fatalf("expected empty endpoint probe to fail")
	}
	if errMsg == "" {
		t.Fatalf("expected empty endpoint probe to provide an error message")
	}
}
