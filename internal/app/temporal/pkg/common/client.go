package common

import (
	"fmt"
	
	"go.temporal.io/sdk/client"
)

// TemporalConfig holds Temporal client configuration
type TemporalConfig struct {
	HostPort  string
	Namespace string
	TaskQueue string
}

// DefaultTemporalConfig returns default Temporal configuration
func DefaultTemporalConfig() TemporalConfig {
	// NO FALLBACK - Check TEMPORAL_ADDRESS first (standard), then TEMPORAL_HOST (legacy)
	hostPort := GetEnv("TEMPORAL_ADDRESS", "")
	if hostPort == "" {
		hostPort = GetEnv("TEMPORAL_HOST", "")
	}
	
	// FAIL FAST - no defaults anywhere
	if hostPort == "" {
		panic("TEMPORAL_ADDRESS or TEMPORAL_HOST must be set")
	}
	
	namespace := GetEnv("TEMPORAL_NAMESPACE", "")
	if namespace == "" {
		panic("TEMPORAL_NAMESPACE must be set")
	}
	
	taskQueue := GetEnv("TASK_QUEUE", "")
	if taskQueue == "" {
		panic("TASK_QUEUE must be set")
	}
	
	return TemporalConfig{
		HostPort:  hostPort,
		Namespace: namespace,
		TaskQueue: taskQueue,
	}
}

// NewTemporalClient creates a new Temporal client with the given configuration
func NewTemporalClient(config TemporalConfig) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}
	return c, nil
}