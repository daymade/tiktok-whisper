package metrics

import "sync"

var (
	globalDouyinMetrics *DouyinMetrics
	metricsOnce         sync.Once
)

// SetGlobalDouyinMetrics sets the global Douyin metrics instance
func SetGlobalDouyinMetrics(m *DouyinMetrics) {
	metricsOnce.Do(func() {
		globalDouyinMetrics = m
	})
}

// GetGlobalDouyinMetrics returns the global Douyin metrics instance
// Returns nil if not initialized (safe to call)
func GetGlobalDouyinMetrics() *DouyinMetrics {
	return globalDouyinMetrics
}
