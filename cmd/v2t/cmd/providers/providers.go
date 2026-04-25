package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
	"tiktok-whisper/internal/app/api/provider"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ProvidersCmd represents the providers command
var ProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage transcription providers",
	Long: `Manage transcription providers including listing, configuring, and checking status.
	
Examples:
  v2t providers list                    # List all providers
  v2t providers status                  # Check provider health
  v2t providers info openai             # Get detailed info about a provider
  v2t providers config                  # Show current configuration
  v2t providers test openai             # Test a specific provider`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available providers",
	Long:  "List all registered transcription providers with their basic information",
	RunE:  runListProviders,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check provider health status",
	Long:  "Check the health status of all registered providers",
	RunE:  runProvidersStatus,
}

var infoCmd = &cobra.Command{
	Use:   "info [provider-name]",
	Short: "Get detailed information about a provider",
	Long:  "Get detailed information about a specific provider including capabilities and configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderInfo,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current provider configuration",
	Long:  "Display the current provider configuration file",
	RunE:  runShowConfig,
}

var testCmd = &cobra.Command{
	Use:   "test [provider-name]",
	Short: "Test a specific provider",
	Long:  "Test a specific provider with a sample audio file or health check",
	Args:  cobra.ExactArgs(1),
	RunE:  runTestProvider,
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show provider usage statistics",
	Long:  "Display usage statistics and metrics for all providers",
	RunE:  runProviderStats,
}

// Command flags
var (
	outputFormat string
	configPath   string
	testFile     string
	verbose      bool
)

func init() {
	ProvidersCmd.AddCommand(listCmd)
	ProvidersCmd.AddCommand(statusCmd)
	ProvidersCmd.AddCommand(infoCmd)
	ProvidersCmd.AddCommand(configCmd)
	ProvidersCmd.AddCommand(testCmd)
	ProvidersCmd.AddCommand(statsCmd)

	// Global flags
	ProvidersCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, yaml)")
	ProvidersCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to provider configuration file")
	ProvidersCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Test command specific flags
	testCmd.Flags().StringVarP(&testFile, "file", "f", "", "Audio file to use for testing")
}

func runListProviders(cmd *cobra.Command, args []string) error {
	factory := provider.NewProviderFactory()
	availableTypes := factory.GetAvailableProviders()

	if outputFormat == "json" {
		return outputJSON(availableTypes)
	}

	if outputFormat == "yaml" {
		return outputYAML(availableTypes)
	}

	// Table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "PROVIDER\tTYPE\tREQUIRES API KEY\tREQUIRES INTERNET\tSUPPORTS TIMESTAMPS\n")

	for _, providerType := range availableTypes {
		info, err := factory.GetProviderInfo(providerType)
		if err != nil {
			continue
		}
		
		fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%v\n",
			info.DisplayName,
			info.Type,
			info.RequiresAPIKey,
			info.RequiresInternet,
			info.SupportsTimestamps,
		)
	}

	return w.Flush()
}

func runProvidersStatus(cmd *cobra.Command, args []string) error {
	// Load configuration and create registry
	configManager := getConfigManager()
	config, err := configManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	registry, err := buildProviderRegistry(config)
	if err != nil {
		return fmt.Errorf("failed to build provider registry: %w", err)
	}

	// Perform health checks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := registry.HealthCheckAll(ctx)

	if outputFormat == "json" {
		return outputJSON(results)
	}

	if outputFormat == "yaml" {
		return outputYAML(results)
	}

	// Table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "PROVIDER\tSTATUS\tERROR\n")

	providerNames := make([]string, 0, len(results))
	for name := range results {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)

	for _, name := range providerNames {
		err := results[name]
		status := "✓ Healthy"
		errorMsg := ""
		
		if err != nil {
			status = "✗ Unhealthy"
			errorMsg = err.Error()
			if len(errorMsg) > 50 {
				errorMsg = errorMsg[:47] + "..."
			}
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, status, errorMsg)
	}

	return w.Flush()
}

func runProviderInfo(cmd *cobra.Command, args []string) error {
	providerName := args[0]
	
	factory := provider.NewProviderFactory()
	info, err := factory.GetProviderInfo(providerName)
	if err != nil {
		return fmt.Errorf("failed to get provider info: %w", err)
	}

	if outputFormat == "json" {
		return outputJSON(info)
	}

	if outputFormat == "yaml" {
		return outputYAML(info)
	}

	// Human-readable format
	fmt.Printf("Provider: %s\n", info.DisplayName)
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Type: %s\n", info.Type)
	fmt.Printf("Version: %s\n", info.Version)
	
	fmt.Printf("\nCapabilities:\n")
	fmt.Printf("  Supported Formats: %s\n", strings.Join(formatAudioFormats(info.SupportedFormats), ", "))
	if len(info.SupportedLanguages) > 0 {
		fmt.Printf("  Supported Languages: %s\n", strings.Join(info.SupportedLanguages, ", "))
	} else {
		fmt.Printf("  Supported Languages: All languages\n")
	}
	fmt.Printf("  Max File Size: %d MB\n", info.MaxFileSizeMB)
	fmt.Printf("  Supports Timestamps: %v\n", info.SupportsTimestamps)
	fmt.Printf("  Supports Word-level: %v\n", info.SupportsWordLevel)
	fmt.Printf("  Supports Confidence: %v\n", info.SupportsConfidence)
	fmt.Printf("  Supports Streaming: %v\n", info.SupportsStreaming)
	
	fmt.Printf("\nRequirements:\n")
	fmt.Printf("  Requires Internet: %v\n", info.RequiresInternet)
	fmt.Printf("  Requires API Key: %v\n", info.RequiresAPIKey)
	fmt.Printf("  Requires Binary: %v\n", info.RequiresBinary)
	
	if info.DefaultModel != "" {
		fmt.Printf("\nModels:\n")
		fmt.Printf("  Default Model: %s\n", info.DefaultModel)
		if len(info.AvailableModels) > 0 {
			fmt.Printf("  Available Models: %s\n", strings.Join(info.AvailableModels, ", "))
		}
	}
	
	fmt.Printf("\nPerformance:\n")
	if info.TypicalLatencyMs > 0 {
		fmt.Printf("  Typical Latency: %d ms per minute of audio\n", info.TypicalLatencyMs)
	}
	if info.CostPerMinute != "" {
		fmt.Printf("  Cost: %s per minute\n", info.CostPerMinute)
	}

	if verbose && len(info.ConfigSchema) > 0 {
		fmt.Printf("\nConfiguration Schema:\n")
		for key, schema := range info.ConfigSchema {
			if schemaMap, ok := schema.(map[string]string); ok {
				fmt.Printf("  %s:\n", key)
				for k, v := range schemaMap {
					fmt.Printf("    %s: %s\n", k, v)
				}
			}
		}
	}

	return nil
}

func runShowConfig(cmd *cobra.Command, args []string) error {
	configManager := getConfigManager()
	config, err := configManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if outputFormat == "json" {
		return outputJSON(config)
	}

	// YAML format (default for config)
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Print(string(data))
	return nil
}

func runTestProvider(cmd *cobra.Command, args []string) error {
	providerName := args[0]
	
	// Load configuration and create registry
	configManager := getConfigManager()
	config, err := configManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	registry, err := buildProviderRegistry(config)
	if err != nil {
		return fmt.Errorf("failed to build provider registry: %w", err)
	}

	// Get the provider
	provider, err := registry.GetProvider(providerName)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	fmt.Printf("Testing provider: %s\n", providerName)

	// Health check
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Print("Running health check... ")
	if err := provider.HealthCheck(ctx); err != nil {
		fmt.Printf("✗ Failed: %v\n", err)
		return err
	}
	fmt.Println("✓ Passed")

	// Test with file if provided
	if testFile != "" {
		fmt.Printf("Testing transcription with file: %s\n", testFile)
		
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			return fmt.Errorf("test file not found: %s", testFile)
		}

		start := time.Now()
		result, err := provider.Transcript(testFile)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("✗ Transcription failed: %v\n", err)
			return err
		}

		fmt.Printf("✓ Transcription completed in %v\n", duration)
		fmt.Printf("Result: %s\n", result)
	}

	return nil
}

func runProviderStats(cmd *cobra.Command, args []string) error {
	// This would require access to the metrics system
	fmt.Println("Provider statistics not yet implemented")
	fmt.Println("This would show usage statistics, success rates, and performance metrics")
	return nil
}

// Helper functions

func getConfigManager() *provider.ConfigManager {
	path := configPath
	if path == "" {
		path = provider.GetDefaultConfigPath()
	}
	return provider.NewConfigManager(path)
}

func buildProviderRegistry(config *provider.ProviderConfiguration) (provider.ProviderRegistry, error) {
	registry := provider.NewProviderRegistry()
	
	for name, providerConfig := range config.Providers {
		if !providerConfig.Enabled {
			continue
		}
		
		provider, err := provider.BuildProviderFromConfig(name, providerConfig)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to create provider %s: %v\n", name, err)
			}
			continue
		}
		
		if err := registry.RegisterProvider(name, provider); err != nil {
			if verbose {
				fmt.Printf("Warning: Failed to register provider %s: %v\n", name, err)
			}
			continue
		}
	}
	
	return registry, nil
}

func formatAudioFormats(formats []provider.AudioFormat) []string {
	result := make([]string, len(formats))
	for i, format := range formats {
		result[i] = string(format)
	}
	return result
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func outputYAML(data interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(data)
}