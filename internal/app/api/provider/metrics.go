package provider

import (
	"sync"
	"time"
)

// DefaultProviderMetrics implements ProviderMetrics interface
type DefaultProviderMetrics struct {
	mu            sync.RWMutex
	providerStats map[string]*ProviderStats
}

// NewProviderMetrics creates a new provider metrics instance
func NewProviderMetrics() *DefaultProviderMetrics {
	return &DefaultProviderMetrics{
		providerStats: make(map[string]*ProviderStats),
	}
}

// RecordSuccess records a successful transcription
func (m *DefaultProviderMetrics) RecordSuccess(provider string, latencyMs int64, audioLengthSec float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats := m.getOrCreateStats(provider)
	stats.TotalRequests++
	stats.SuccessfulRequests++
	stats.TotalAudioProcessed += audioLengthSec
	stats.LastUsed = time.Now().Unix()
	stats.IsHealthy = true
	
	// Update average latency (simple moving average)
	if stats.AverageLatencyMs == 0 {
		stats.AverageLatencyMs = float64(latencyMs)
	} else {
		// Weighted average favoring recent results
		stats.AverageLatencyMs = (stats.AverageLatencyMs*0.8) + (float64(latencyMs)*0.2)
	}
	
	// Update success rate
	stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
}

// RecordFailure records a failed transcription
func (m *DefaultProviderMetrics) RecordFailure(provider string, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	stats := m.getOrCreateStats(provider)
	stats.TotalRequests++
	stats.FailedRequests++
	stats.LastUsed = time.Now().Unix()
	
	// Record error type
	if stats.ErrorBreakdown == nil {
		stats.ErrorBreakdown = make(map[string]int64)
	}
	stats.ErrorBreakdown[errorType]++
	
	// Update success rate
	stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
	
	// Mark as unhealthy if failure rate is too high
	if stats.TotalRequests >= 10 && stats.SuccessRate < 0.5 {
		stats.IsHealthy = false
	}
}

// GetProviderMetrics returns metrics for a specific provider
func (m *DefaultProviderMetrics) GetProviderMetrics(provider string) ProviderStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := m.getOrCreateStats(provider)
	
	// Return a copy to avoid race conditions
	return ProviderStats{
		Provider:             stats.Provider,
		TotalRequests:        stats.TotalRequests,
		SuccessfulRequests:   stats.SuccessfulRequests,
		FailedRequests:       stats.FailedRequests,
		SuccessRate:          stats.SuccessRate,
		AverageLatencyMs:     stats.AverageLatencyMs,
		TotalAudioProcessed:  stats.TotalAudioProcessed,
		LastUsed:             stats.LastUsed,
		IsHealthy:            stats.IsHealthy,
		ErrorBreakdown:       m.copyErrorBreakdown(stats.ErrorBreakdown),
	}
}

// GetOverallMetrics returns overall metrics across all providers
func (m *DefaultProviderMetrics) GetOverallMetrics() OverallStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var totalRequests, successfulRequests int64
	var fastestProvider, mostReliableProvider string
	var fastestLatency, highestReliability float64
	activeProviders := 0
	
	providerStats := make(map[string]ProviderStats)
	
	for name, stats := range m.providerStats {
		totalRequests += stats.TotalRequests
		successfulRequests += stats.SuccessfulRequests
		
		// Copy stats
		providerStats[name] = m.GetProviderMetrics(name)
		
		// Track fastest provider (lowest latency)
		if stats.AverageLatencyMs > 0 && (fastestLatency == 0 || stats.AverageLatencyMs < fastestLatency) {
			fastestLatency = stats.AverageLatencyMs
			fastestProvider = name
		}
		
		// Track most reliable provider (highest success rate with meaningful volume)
		if stats.TotalRequests >= 5 && stats.SuccessRate > highestReliability {
			highestReliability = stats.SuccessRate
			mostReliableProvider = name
		}
		
		// Count active providers (used recently)
		if stats.LastUsed > 0 && time.Now().Unix()-stats.LastUsed < 3600 { // Active within last hour
			activeProviders++
		}
	}
	
	var overallSuccessRate float64
	if totalRequests > 0 {
		overallSuccessRate = float64(successfulRequests) / float64(totalRequests)
	}
	
	return OverallStats{
		TotalProviders:       len(m.providerStats),
		ActiveProviders:      activeProviders,
		TotalRequests:        totalRequests,
		SuccessfulRequests:   successfulRequests,
		OverallSuccessRate:   overallSuccessRate,
		FastestProvider:      fastestProvider,
		MostReliableProvider: mostReliableProvider,
		ProviderStats:        providerStats,
	}
}

// getOrCreateStats gets existing stats or creates new ones (must be called with lock held)
func (m *DefaultProviderMetrics) getOrCreateStats(provider string) *ProviderStats {
	stats, exists := m.providerStats[provider]
	if !exists {
		stats = &ProviderStats{
			Provider:       provider,
			ErrorBreakdown: make(map[string]int64),
		}
		m.providerStats[provider] = stats
	}
	return stats
}

// copyErrorBreakdown creates a copy of the error breakdown map
func (m *DefaultProviderMetrics) copyErrorBreakdown(original map[string]int64) map[string]int64 {
	if original == nil {
		return nil
	}
	
	copy := make(map[string]int64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// ResetStats resets all statistics (useful for testing or maintenance)
func (m *DefaultProviderMetrics) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.providerStats = make(map[string]*ProviderStats)
}

// GetProviderNames returns all provider names that have metrics
func (m *DefaultProviderMetrics) GetProviderNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.providerStats))
	for name := range m.providerStats {
		names = append(names, name)
	}
	return names
}