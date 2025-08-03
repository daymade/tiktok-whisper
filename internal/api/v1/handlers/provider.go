package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tiktok-whisper/internal/api/errors"
	"tiktok-whisper/internal/api/middleware"
	"tiktok-whisper/internal/api/v1/dto"
	"tiktok-whisper/internal/api/v1/services"
)

// ProviderHandler handles provider-related API endpoints
type ProviderHandler struct {
	service services.ProviderService
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(service services.ProviderService) *ProviderHandler {
	return &ProviderHandler{
		service: service,
	}
}

// List handles GET /api/v1/providers
// Lists all available transcription providers
//
// @Summary List all available providers
// @Description Retrieves a list of all registered transcription providers with their capabilities and status
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "List of providers" SchemaExample({"providers": [{"id": "openai/whisper", "name": "OpenAI Whisper", "type": "remote", "available": true, "health_status": "healthy"}]})
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /providers [get]
func (h *ProviderHandler) List(c *gin.Context) {
	providers, err := h.service.ListProviders(c.Request.Context())
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

// Get handles GET /api/v1/providers/:id
// Gets detailed information about a specific provider
//
// @Summary Get provider details
// @Description Retrieves detailed information about a specific transcription provider including capabilities and configuration
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID" example(openai/whisper)
// @Success 200 {object} dto.ProviderResponse "Provider details"
// @Failure 400 {object} errors.APIError "Bad request - provider ID required"
// @Failure 404 {object} errors.APIError "Provider not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /providers/{id} [get]
func (h *ProviderHandler) Get(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		middleware.HandleError(c, errors.NewBadRequestError("Provider ID is required"))
		return
	}

	provider, err := h.service.GetProvider(c.Request.Context(), providerID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, provider)
}

// GetStatus handles GET /api/v1/providers/:id/status
// Gets the current health status of a provider
//
// @Summary Get provider health status
// @Description Performs a health check on a specific provider and returns current status with response time
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID" example(openai/whisper)
// @Success 200 {object} dto.ProviderStatusResponse "Provider health status"
// @Failure 400 {object} errors.APIError "Bad request - provider ID required"
// @Failure 404 {object} errors.APIError "Provider not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /providers/{id}/status [get]
func (h *ProviderHandler) GetStatus(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		middleware.HandleError(c, errors.NewBadRequestError("Provider ID is required"))
		return
	}

	status, err := h.service.GetProviderStatus(c.Request.Context(), providerID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetStats handles GET /api/v1/providers/:id/stats
// Gets usage statistics for a provider
//
// @Summary Get provider usage statistics
// @Description Retrieves detailed usage statistics for a provider including request counts, success rates, and performance metrics
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID" example(openai/whisper)
// @Success 200 {object} dto.ProviderStatsResponse "Provider usage statistics"
// @Failure 400 {object} errors.APIError "Bad request - provider ID required"
// @Failure 404 {object} errors.APIError "Provider not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /providers/{id}/stats [get]
func (h *ProviderHandler) GetStats(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		middleware.HandleError(c, errors.NewBadRequestError("Provider ID is required"))
		return
	}

	stats, err := h.service.GetProviderStats(c.Request.Context(), providerID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Test handles POST /api/v1/providers/:id/test
// @Summary Test a provider
// @Description Tests a provider with an optional audio file to verify it's working correctly
// @Tags providers
// @Accept json
// @Produce json
// @Param id path string true "Provider ID" example("openai/whisper")
// @Param request body dto.TestProviderRequest false "Test request with optional audio file path"
// @Success 200 {object} dto.TestProviderResponse "Test result"
// @Failure 400 {object} errors.APIError "Bad request (invalid provider ID)"
// @Failure 404 {object} errors.APIError "Provider not found"
// @Failure 500 {object} errors.APIError "Internal server error"
// @Router /providers/{id}/test [post]
func (h *ProviderHandler) Test(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		middleware.HandleError(c, errors.NewBadRequestError("Provider ID is required"))
		return
	}

	var req dto.TestProviderRequest
	// Allow empty body for default test
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.TestProvider(c.Request.Context(), providerID, &req)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}