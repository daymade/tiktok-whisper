# Swagger Documentation Development Guide

This guide explains how to work with Swagger/OpenAPI documentation in the TikTok Whisper API project.

## Overview

The project uses [swaggo/swag](https://github.com/swaggo/swag) to automatically generate OpenAPI/Swagger documentation from Go code annotations. The Swagger UI is accessible at `/swagger/index.html` and `/docs/index.html` when the API server is running.

## Generated Files

The following files are automatically generated and should **NOT** be edited manually:

- `docs/docs.go` - Go code for embedding the swagger spec
- `docs/swagger.json` - OpenAPI specification in JSON format  
- `docs/swagger.yaml` - OpenAPI specification in YAML format

## Development Workflow

### 1. Adding New API Endpoints

When adding new API endpoints, ensure you include comprehensive Swagger annotations:

```go
// @Summary Brief description of the endpoint
// @Description Detailed description of what the endpoint does
// @Tags tag-name
// @Accept json
// @Produce json
// @Param paramName path/query/body type required "Description" example("value")
// @Success 200 {object} ResponseType "Success description"
// @Failure 400 {object} errors.APIError "Error description"
// @Router /path/to/endpoint [method]
func HandlerFunction(c *gin.Context) {
    // Implementation
}
```

### 2. Updating Existing Endpoints

When modifying existing endpoints:

1. Update the Swagger annotations in the handler function
2. Regenerate documentation (see step 3)
3. Test the updated documentation

### 3. Regenerating Documentation

After making changes to Swagger annotations, regenerate the documentation:

```bash
# From project root
swag init -g cmd/v2t/cmd/api.go -o docs --parseDependency
```

**Important**: Always regenerate documentation before committing changes to ensure the docs are up-to-date.

### 4. Testing Documentation

Start the API server and verify the documentation:

```bash
# Start the API server
go run cmd/v2t/main.go api --port 8081

# Access Swagger UI in browser
open http://localhost:8081/swagger/index.html

# Or use curl to test endpoints
curl http://localhost:8081/swagger/doc.json
```

## Swagger Annotation Reference

### Common Annotations

- `@Summary` - Brief one-line description
- `@Description` - Detailed multi-line description
- `@Tags` - Groups endpoints in the UI
- `@Accept` - Request content type (usually `json`)
- `@Produce` - Response content type (usually `json`)
- `@Router` - HTTP method and path

### Parameter Types

- `@Param name path type required "description"` - URL path parameter
- `@Param name query type required "description"` - Query parameter
- `@Param name body type required "description"` - Request body
- `@Param name header type required "description"` - HTTP header

### Response Codes

- `@Success 200 {object} Type "description"` - Success response
- `@Failure 400 {object} errors.APIError "description"` - Error response

### Data Types

- `string`, `integer`, `boolean`, `number` - Basic types
- `{object} StructName` - Go struct type
- `{array} Type` - Array of type

## Best Practices

### 1. Comprehensive Documentation

- Always include `@Summary` and `@Description`
- Document all parameters with clear descriptions
- Include examples for complex parameters
- Document all possible response codes

### 2. Consistent Formatting

- Use consistent tag names across related endpoints
- Follow the same description format
- Include proper error responses

### 3. DTO Documentation

Ensure all DTOs (Data Transfer Objects) have proper struct tags:

```go
type ExampleRequest struct {
    Name     string `json:"name" example:"John Doe" validate:"required"`
    Email    string `json:"email" example:"john@example.com" validate:"required,email"`
    Optional string `json:"optional,omitempty" example:"optional value"`
}
```

### 4. Error Handling

Always document standard error responses:

```go
// @Failure 400 {object} errors.APIError "Bad request"
// @Failure 404 {object} errors.APIError "Not found"
// @Failure 500 {object} errors.APIError "Internal server error"
```

## API Configuration

The main API configuration is in `cmd/v2t/cmd/api.go`:

```go
// @title TikTok Whisper API
// @version 1.0
// @description RESTful API for audio transcription services
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
```

Update these annotations when changing API metadata.

## Integration with CI/CD

Consider adding documentation generation to your build process:

```bash
# In your build script or Makefile
.PHONY: docs
docs:
	swag init -g cmd/v2t/cmd/api.go -o docs --parseDependency

.PHONY: test-docs
test-docs: docs
	# Start server and test documentation endpoints
	go run cmd/v2t/main.go api --port 8082 &
	sleep 2
	curl -f http://localhost:8082/swagger/doc.json > /dev/null
	pkill -f "go run cmd/v2t/main.go api"
```

## Troubleshooting

### Common Issues

1. **Documentation not updating**: Ensure you regenerated with `swag init`
2. **Server won't start**: Check for port conflicts, try different port
3. **Missing types in docs**: Verify struct tags and import statements
4. **Invalid Swagger spec**: Check annotation syntax and parameter types

### Debugging

- Check `swag init` output for warnings or errors
- Validate generated `swagger.json` with online validators
- Test individual endpoints with curl or Postman
- Review browser console for Swagger UI errors

## Related Documentation

- [Swagger/OpenAPI Specification](https://swagger.io/specification/)
- [swaggo/swag Documentation](https://github.com/swaggo/swag)
- [Gin Swagger Integration](https://github.com/swaggo/gin-swagger)