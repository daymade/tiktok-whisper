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
	return TemporalConfig{
		HostPort:  GetEnv("TEMPORAL_HOST", DefaultTemporalHost),
		Namespace: GetEnv("TEMPORAL_NAMESPACE", DefaultNamespace),
		TaskQueue: GetEnv("TASK_QUEUE", DefaultTaskQueue),
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