package swagger

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SwaggerUIHandler handles serving the Swagger UI documentation
type SwaggerUIHandler struct {
	openAPISpec []byte
}

// NewSwaggerUIHandler creates a new Swagger UI handler
func NewSwaggerUIHandler(serverPort string) (*SwaggerUIHandler, error) {
	// Read the OpenAPI specification with dynamic server URL
	spec, err := readOpenAPISpec(serverPort)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec: %w", err)
	}

	return &SwaggerUIHandler{
		openAPISpec: spec,
	}, nil
}

// ServeHTTP handles HTTP requests for Swagger UI
func (h *SwaggerUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle API spec request
	if r.URL.Path == "/api-docs" {
		w.Header().Set("Content-Type", "application/json")
		w.Write(h.openAPISpec)
		return
	}

	// Serve the main Swagger UI page
	if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
		h.serveSwaggerUI(w, r)
		return
	}

	// Redirect root docs path
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/docs", http.StatusMovedPermanently)
		return
	}

	http.NotFound(w, r)
}

// serveSwaggerUI serves the main Swagger UI page
func (h *SwaggerUIHandler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	// Get the current server URL from the request
	serverURL := fmt.Sprintf("http://%s", r.Host)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SecAuto Modular API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
        .swagger-ui .topbar {
            background-color: #2c3e50;
        }
        .swagger-ui .topbar .download-url-wrapper .select-label {
            color: #fff;
        }
        .swagger-ui .topbar .download-url-wrapper input[type=text] {
            border: 2px solid #34495e;
        }
        .swagger-ui .info .title {
            color: #2c3e50;
        }
        .swagger-ui .scheme-container {
            background-color: #ecf0f1;
        }
        .swagger-ui .info .title {
            font-size: 36px;
            color: #2c3e50;
        }
        .swagger-ui .info .description {
            font-size: 16px;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '%s/api-docs',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                filter: true,
                displayRequestDuration: true,
                docExpansion: "list",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                validatorUrl: null
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`, serverURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// GetSpec returns the OpenAPI specification as a map
func GetSpec() (map[string]interface{}, error) {
	specBytes, err := readOpenAPISpec("8080")
	if err != nil {
		return nil, err
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// readOpenAPISpec generates the OpenAPI specification for the modular build
func readOpenAPISpec(serverPort string) ([]byte, error) {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "SecAuto Modular SOAR API",
			"description": "A modular Security Orchestration, Automation, and Response (SOAR) platform API with database-backed client management. This modular implementation provides core functionality for playbook execution, caching, multi-tenant client management, and Redis list operations.",
			"version":     "modular-2.0.0",
			"contact": map[string]interface{}{
				"name":  "SecAuto Support",
				"email": "support@secauto.com",
			},
			"license": map[string]interface{}{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         fmt.Sprintf("http://localhost:%s", serverPort),
				"description": "Development server",
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "Health",
				"description": "System health and status endpoints",
			},
			{
				"name":        "Playbooks",
				"description": "Playbook execution and management",
			},
			{
				"name":        "Cache",
				"description": "Redis cache operations",
			},
			{
				"name":        "Lists",
				"description": "Redis list operations",
			},
			{
				"name":        "Schedules",
				"description": "Job scheduling and cron management",
			},
			{
				"name":        "Authentication",
				"description": "API key authentication and management",
			},
			{
				"name":        "Clients",
				"description": "Database-backed multi-tenant client management with ACID compliance, advanced search, and isolated integrations",
			},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health Check",
					"description": "Check the health status of the SecAuto modular system",
					"tags":        []string{"Health"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "System is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/HealthResponse",
									},
								},
							},
						},
					},
				},
			},
			"/playbook": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Playbook",
					"description": "Execute a playbook with given context",
					"tags":        []string{"Playbooks"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/PlaybookRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbook executed successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PlaybookResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request or validation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ValidationResponse",
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Playbook execution failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PlaybookResponse",
									},
								},
							},
						},
					},
				},
			},
			"/cache": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Cache Keys",
					"description": "Get all cache keys with optional pattern filtering",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "pattern",
							"in":          "query",
							"description": "Pattern to filter cache keys (e.g., '*user*')",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache keys retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/cache/{key}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Cache Value",
					"description": "Retrieve a value from cache by key",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "key",
							"in":          "path",
							"description": "Cache key to retrieve",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache value retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Cache key not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Set Cache Value",
					"description": "Store a value in cache with the specified key",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "key",
							"in":          "path",
							"description": "Cache key to set",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"value": map[string]interface{}{
											"description": "The value to store in cache",
										},
									},
									"required": []string{"value"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache value set successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Cache Value",
					"description": "Remove a value from cache by key",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "key",
							"in":          "path",
							"description": "Cache key to delete",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache value deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Cache key not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
					},
				},
			},
			"/cache/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Cache Statistics",
					"description": "Retrieve detailed Redis cache statistics",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache statistics retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheStatsResponse",
									},
								},
							},
						},
					},
				},
			},
			"/cache/clear": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Clear Cache",
					"description": "Clear all cache entries",
					"tags":        []string{"Cache"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cache cleared successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/CacheResponse",
									},
								},
							},
						},
					},
				},
			},
			"/lists/{list_name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get List Items",
					"description": "Retrieve all items from a Redis list",
					"tags":        []string{"Lists"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "list_name",
							"in":          "path",
							"description": "Name of the Redis list",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List items retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListResponse",
									},
								},
							},
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete List",
					"description": "Delete an entire Redis list",
					"tags":        []string{"Lists"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "list_name",
							"in":          "path",
							"description": "Name of the Redis list to delete",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "List not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/lists/{list_name}/items": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Add Items to List",
					"description": "Add items to a Redis list at specified position",
					"tags":        []string{"Lists"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "list_name",
							"in":          "path",
							"description": "Name of the Redis list",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ListAddRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Items added to list successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Remove Items from List",
					"description": "Remove specific items from a Redis list",
					"tags":        []string{"Lists"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "list_name",
							"in":          "path",
							"description": "Name of the Redis list",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ListRemoveRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Items removed from list successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ListResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
					},
				},
			},
			"/integrations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Integrations",
					"description": "Get all configured integrations",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integrations retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "array",
										"items": map[string]interface{}{
											"$ref": "#/components/schemas/Integration",
										},
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Integration",
					"description": "Create a new integration configuration",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/Integration",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Integration created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Integration",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid integration configuration",
						},
					},
				},
			},
			"/integrations/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Integration",
					"description": "Get a specific integration by ID",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Integration ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Integration",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Integration",
					"description": "Update an existing integration",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Integration ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/Integration",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Integration",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Integration",
					"description": "Delete an integration",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Integration ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"204": map[string]interface{}{
							"description": "Integration deleted successfully",
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
					},
				},
			},
			"/integrations/{id}/test": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Test Integration",
					"description": "Test an integration connection",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Integration ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration test result",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/TestResult",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
					},
				},
			},
			"/integrations/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Integration Definition and Script",
					"description": "Upload both an integration definition (JSON) and Python script (.py). The definition contains metadata, dependencies, configuration fields, and function definitions. The script filename must match the entry_point specified in the definition. If requires_build is true, dependencies will be automatically installed.",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"definition": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "JSON integration definition file (.json) containing metadata, dependencies, configuration fields, and function definitions",
										},
										"script": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Python integration script file (.py) - filename must match the entry_point in definition",
										},
									},
									"required": []string{"definition", "script"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration uploaded and built successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/IntegrationUploadResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid integration definition JSON, missing required fields, or script filename doesn't match entry_point",
						},
						"500": map[string]interface{}{
							"description": "Internal server error during upload",
						},
					},
				},
			},
			"/integrations/build-status/{integration_name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Integration Build Status",
					"description": "Retrieve the build status of a specific integration backend, including dependency installation status and site-packages location",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "integration_name",
							"in":          "path",
							"required":    true,
							"description": "Name of the integration to check build status for",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Build status retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/IntegrationBuildStatusResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Integration not found or not built",
						},
						"500": map[string]interface{}{
							"description": "Error retrieving build status",
						},
					},
				},
			},
			"/integrations/build-status": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Integration Build Statuses",
					"description": "Retrieve build status for all integrations that have been built",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Build statuses retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/IntegrationBuildStatusListResponse",
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Error retrieving build statuses",
						},
					},
				},
			},
			"/api-keys": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List API Keys",
					"description": "Get all API keys (excluding actual key values)",
					"tags":        []string{"Authentication"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "API keys retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/APIKeyListResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create API Key",
					"description": "Create a new API key for authentication",
					"tags":        []string{"Authentication"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/APIKeyCreateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "API key created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/APIKeyCreateResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request - name required",
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
						"500": map[string]interface{}{
							"description": "Internal server error during key creation",
						},
					},
				},
			},
			"/api-keys/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "API Key Statistics",
					"description": "Get API key statistics and counts",
					"tags":        []string{"Authentication"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Statistics retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/APIKeyStatsResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
					},
				},
			},
			"/cluster": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Cluster Info",
					"description": "Get cluster status and node information",
					"tags":        []string{"Cluster"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cluster information retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClusterInfo",
									},
								},
							},
						},
					},
				},
			},
			"/cluster/submit": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Submit Job to Cluster",
					"description": "Submit a job for distributed execution",
					"tags":        []string{"Cluster"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/JobSubmitRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job submitted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobSubmitResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid job submission",
						},
						"503": map[string]interface{}{
							"description": "Cluster queue is full",
						},
					},
				},
			},
			"/cluster/job/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Job Status",
					"description": "Get the status of a specific job",
					"tags":        []string{"Cluster"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Job ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job information retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/Job",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Job not found",
						},
					},
				},
			},
			"/playbook/async": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Playbook Asynchronously",
					"description": "Submit a playbook for asynchronous execution via cluster",
					"tags":        []string{"Playbooks"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/PlaybookRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbook submitted for async execution",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AsyncPlaybookResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid playbook or validation failed",
						},
						"503": map[string]interface{}{
							"description": "Cluster queue is full",
						},
					},
				},
			},
			"/playbook/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Playbook File",
					"description": "Upload a playbook file to the server",
					"tags":        []string{"Playbooks"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"file": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Playbook JSON file",
										},
										"name": map[string]interface{}{
											"type":        "string",
											"description": "Name for the playbook (optional)",
										},
									},
									"required": []string{"file"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbook uploaded successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/UploadResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid file or upload error",
						},
					},
				},
			},
			"/playbooks": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Playbooks",
					"description": "Get all stored playbooks with metadata",
					"tags":        []string{"Playbooks"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbooks retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/PlaybooksListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/playbook/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "Delete Playbook",
					"description": "Delete a specific playbook by name",
					"tags":        []string{"Playbooks"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"description": "Playbook name",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbook deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Playbook not found",
						},
					},
				},
			},
			"/automation": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Automation Script",
					"description": "Upload a Python automation script with optional metadata",
					"tags":        []string{"Automations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"automation": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Python automation script (.py file)",
										},
										"name": map[string]interface{}{
											"type":        "string",
											"description": "Override automation name (optional)",
										},
									},
									"required": []string{"automation"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation uploaded successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationUploadResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid file or upload error",
						},
					},
				},
			},
			"/automations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Automation Scripts",
					"description": "Get all automation scripts with detailed analysis",
					"tags":        []string{"Automations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automations retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationListResponse",
									},
								},
							},
						},
					},
				},
			},
			"/automation/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "Delete Automation Script",
					"description": "Delete an automation script and its metadata",
					"tags":        []string{"Automations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"description": "Automation script name",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationDeleteResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Automation not found",
						},
					},
				},
			},
			"/automation/metadata": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Automation Metadata",
					"description": "Get metadata for all automation scripts",
					"tags":        []string{"Automation Metadata"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation metadata retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationMetadataListResponse",
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Automation Metadata",
					"description": "Create metadata for an automation script",
					"tags":        []string{"Automation Metadata"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AutomationMetadata",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Automation metadata created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationMetadataResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid metadata or validation failed",
						},
					},
				},
			},
			"/automation/metadata/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Automation Metadata",
					"description": "Get metadata for a specific automation script",
					"tags":        []string{"Automation Metadata"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"description": "Automation script name",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation metadata retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationMetadataResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Automation metadata not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Automation Metadata",
					"description": "Update metadata for an automation script",
					"tags":        []string{"Automation Metadata"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"description": "Automation script name",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/AutomationMetadata",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation metadata updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/AutomationMetadataResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid metadata or validation failed",
						},
						"404": map[string]interface{}{
							"description": "Automation metadata not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Automation Metadata",
					"description": "Delete metadata for an automation script",
					"tags":        []string{"Automation Metadata"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"description": "Automation script name",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automation metadata deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"name": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Automation metadata not found",
						},
					},
				},
			},
			"/jobs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Jobs",
					"description": "Get all jobs with optional filtering by status and limit",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "status",
							"in":          "query",
							"description": "Filter jobs by status (pending, running, completed, failed)",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"pending", "running", "completed", "failed"},
							},
						},
						{
							"name":        "limit",
							"in":          "query",
							"description": "Maximum number of jobs to return (default: 50)",
							"required":    false,
							"schema": map[string]interface{}{
								"type":    "integer",
								"minimum": 1,
								"maximum": 1000,
								"default": 50,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Jobs retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobListResponse",
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Job",
					"description": "Create a new job for playbook execution",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/JobCreateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Job created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
					},
				},
			},
			"/jobs/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Job Statistics",
					"description": "Get comprehensive job statistics and metrics",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job statistics retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobStatsResponse",
									},
								},
							},
						},
					},
				},
			},
			"/job/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Job",
					"description": "Get a specific job by ID",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Job ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Job not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Job",
					"description": "Update job status or other properties",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Job ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/JobUpdateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request body",
						},
						"404": map[string]interface{}{
							"description": "Job not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Job",
					"description": "Delete a specific job by ID",
					"tags":        []string{"Jobs"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Job ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Job deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/JobDeleteResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Job not found",
						},
					},
				},
			},
			"/schedules": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Schedules",
					"description": "Get all schedules with optional status filtering",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "status",
							"in":          "query",
							"description": "Filter by schedule status (enabled, disabled)",
							"required":    false,
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"enabled", "disabled"},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedules retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleListResponse",
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Schedule",
					"description": "Create a new job schedule with cron expression",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ScheduleCreateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Schedule created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request or cron expression",
						},
					},
				},
			},
			"/schedules/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Schedule Statistics",
					"description": "Get schedule statistics and counts",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Statistics retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleStatsResponse",
									},
								},
							},
						},
					},
				},
			},
			"/schedule/{id}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Schedule",
					"description": "Get a specific schedule by ID",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Schedule ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedule retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Schedule not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Schedule",
					"description": "Update an existing schedule",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Schedule ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ScheduleUpdateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedule updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request",
						},
						"404": map[string]interface{}{
							"description": "Schedule not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Schedule",
					"description": "Delete a specific schedule by ID",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Schedule ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedule deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleDeleteResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Schedule not found",
						},
					},
				},
			},
			"/schedule/execute/{id}": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Schedule",
					"description": "Manually execute a schedule immediately",
					"tags":        []string{"Schedules"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "id",
							"in":          "path",
							"description": "Schedule ID",
							"required":    true,
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedule executed successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ScheduleExecuteResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Schedule not found",
						},
						"500": map[string]interface{}{
							"description": "Execution failed",
						},
					},
				},
			},
			"/clients": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Clients",
					"description": "Get all clients in the system",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Clients retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientListResponse",
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Client",
					"description": "Create a new client with isolated integrations (stored in PostgreSQL database)",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ClientCreateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Client created successfully in database",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientCreateResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request - name required",
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
						"500": map[string]interface{}{
							"description": "Internal server error during client creation",
						},
					},
				},
			},
			"/clients/search": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Search Clients",
					"description": "Search clients by name or metadata using database full-text search",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "q",
							"in":          "query",
							"required":    true,
							"description": "Search query (searches name, description, and metadata)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "limit",
							"in":          "query",
							"required":    false,
							"description": "Maximum number of results to return (default: 50)",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 50,
								"minimum": 1,
								"maximum": 100,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Search results retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientSearchResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid search query",
						},
						"401": map[string]interface{}{
							"description": "Unauthorized - invalid or missing API key",
						},
					},
				},
			},
			"/clients/{clientId}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Client",
					"description": "Get details of a specific client",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Client retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Client not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Client",
					"description": "Update client details and settings",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ClientUpdateRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Client updated successfully",
						},
						"404": map[string]interface{}{
							"description": "Client not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Client",
					"description": "Delete a client and all associated data",
					"tags":        []string{"Clients"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Client deleted successfully",
						},
						"404": map[string]interface{}{
							"description": "Client not found",
						},
					},
				},
			},
			"/clients/{clientId}/integrations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List Client Integrations",
					"description": "Get all integrations for a specific client",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Client integrations retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientIntegrationListResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Client not found",
						},
					},
				},
			},
			"/clients/{clientId}/integrations/{integrationName}/config": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Client Integration Configuration",
					"description": "Get configuration for a specific client integration",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "integrationName",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., postgresql, virustotal)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration configuration retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientIntegrationConfigResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Client or integration not found",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Client Integration Configuration",
					"description": "Create configuration for a specific client integration",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "integrationName",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., postgresql, virustotal)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ClientIntegrationConfigRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration configuration created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientIntegrationConfigResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request or configuration",
						},
						"404": map[string]interface{}{
							"description": "Client or integration not found",
						},
					},
				},
				"put": map[string]interface{}{
					"summary":     "Update Client Integration Configuration",
					"description": "Update configuration for a specific client integration",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "integrationName",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., postgresql, virustotal)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/ClientIntegrationConfigRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration configuration updated successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientIntegrationConfigResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request or configuration",
						},
						"404": map[string]interface{}{
							"description": "Client or integration not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Client Integration Configuration",
					"description": "Delete configuration for a specific client integration",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "integrationName",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., postgresql, virustotal)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration configuration deleted successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/ClientIntegrationConfigResponse",
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Client or integration not found",
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
						},
					},
				},
			},
			"/clients/{clientId}/integrations/{integrationName}/execute": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Client Integration Function",
					"description": "Execute a specific function on a client integration",
					"tags":        []string{"Clients", "Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "clientId",
							"in":          "path",
							"required":    true,
							"description": "Client ID",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
						{
							"name":        "integrationName",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., postgresql, virustotal)",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"$ref": "#/components/schemas/IntegrationExecuteRequest",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration function executed successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"$ref": "#/components/schemas/IntegrationExecuteResponse",
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request or function parameters",
						},
						"404": map[string]interface{}{
							"description": "Client, integration, or function not found",
						},
						"500": map[string]interface{}{
							"description": "Integration execution failed",
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"ApiKeyAuth": map[string]interface{}{
					"type":        "apiKey",
					"in":          "header",
					"name":        "X-API-Key",
					"description": "API key for authentication",
				},
			},
			"schemas": map[string]interface{}{
				"HealthResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Health status of the system",
							"enum":        []string{"healthy", "unhealthy"},
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of the health check",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "Version of the modular system",
						},
						"modules": map[string]interface{}{
							"type":        "object",
							"description": "Status of loaded modules",
							"additionalProperties": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
				"PlaybookRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"playbook": map[string]interface{}{
							"type":        "array",
							"description": "Direct playbook rules array. Each rule can be a simple action like {\"run\": \"automation_name\", \"client\": \"client_id\"}",
							"items": map[string]interface{}{
								"type": "object",
							},
							"example": []map[string]interface{}{
								{"run": "security_scan", "client": "{{client_name}}"},
							},
						},
						"playbook_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of playbook file to load",
							"example":     "incident_response",
						},
						"context": map[string]interface{}{
							"type":                 "object",
							"description":          "Context data for playbook execution",
							"additionalProperties": true,
							"example": map[string]interface{}{
								"incident": map[string]interface{}{
									"severity": "high",
									"source":   "network_monitor",
								},
								"client_name": "client_001",
							},
						},
					},
					"example": map[string]interface{}{
						"playbook": []map[string]interface{}{
							{"run": "security_scan", "client": "client_001"},
						},
						"context": map[string]interface{}{
							"incident": map[string]interface{}{
								"severity": "high",
							},
						},
					},
				},
				"PlaybookResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the playbook execution was successful",
						},
						"result": map[string]interface{}{
							"description": "Result of playbook execution",
						},
						"job_id": map[string]interface{}{
							"type":        "string",
							"description": "Job ID for async execution",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Success or error message",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of response",
						},
					},
				},
				"ValidationResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the validation was successful",
						},
						"valid": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the request is valid",
						},
						"errors": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ValidationError",
							},
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Validation message",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of validation",
						},
					},
				},
				"ValidationError": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"field": map[string]interface{}{
							"type":        "string",
							"description": "Field that failed validation",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Validation error message",
						},
						"value": map[string]interface{}{
							"description": "Value that failed validation",
						},
					},
				},
				"CacheResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the cache operation was successful",
						},
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Cache key",
						},
						"value": map[string]interface{}{
							"description": "Cache value (for get/set operations)",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Success message",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "Error message if operation failed",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of operation",
						},
					},
				},
				"CacheListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the operation was successful",
						},
						"keys": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "List of cache keys",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Operation message",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of operation",
						},
					},
				},
				"CacheStatsResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the operation was successful",
						},
						"stats": map[string]interface{}{
							"type":                 "object",
							"description":          "Redis cache statistics",
							"additionalProperties": true,
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Operation message",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of operation",
						},
					},
				},
				"ListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the list operation was successful",
						},
						"list_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the Redis list",
						},
						"items": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "List items",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of items in list or affected by operation",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Operation message",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "Error message if operation failed",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Timestamp of operation",
						},
					},
				},
				"ListAddRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"items"},
					"properties": map[string]interface{}{
						"items": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Items to add to the list",
						},
						"position": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"left", "right"},
							"default":     "right",
							"description": "Position to add items (left or right)",
						},
					},
				},
				"ListRemoveRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"items"},
					"properties": map[string]interface{}{
						"items": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Items to remove from the list",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"default":     1,
							"description": "Number of occurrences to remove per item",
						},
					},
				},
				"Integration": map[string]interface{}{
					"type":     "object",
					"required": []string{"id", "name", "type"},
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique integration identifier",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Integration name",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Integration type (e.g., splunk, elastic, siem)",
						},
						"config": map[string]interface{}{
							"type":        "object",
							"description": "Integration configuration parameters",
						},
						"enabled": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the integration is enabled",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Integration status",
						},
						"last_tested": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last test timestamp",
						},
					},
				},
				"TestResult": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the test was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Test result message",
						},
						"response_time": map[string]interface{}{
							"type":        "number",
							"description": "Response time in milliseconds",
						},
						"details": map[string]interface{}{
							"type":        "object",
							"description": "Additional test details",
						},
					},
				},
				"IntegrationUploadResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether upload was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"integration_name": map[string]interface{}{
							"type":        "string",
							"description": "Uploaded integration name",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "Integration version from definition",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "Uploaded file name",
						},
						"built": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the integration environment was built successfully",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "File size in bytes",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Upload timestamp",
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Additional metadata including build results (for JSON configuration files)",
							"properties": map[string]interface{}{
								"success": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether backend build was successful (for JSON configs)",
								},
								"integration": map[string]interface{}{
									"type":        "string",
									"description": "Integration name that was built",
								},
								"site_packages": map[string]interface{}{
									"type":        "string",
									"description": "Path to integration-specific site-packages directory",
								},
								"dependencies_installed": map[string]interface{}{
									"type":        "integer",
									"description": "Number of dependencies successfully installed",
								},
								"error": map[string]interface{}{
									"type":        "string",
									"description": "Error message if build failed",
								},
							},
						},
					},
				},
				"IntegrationBuildStatusResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the status retrieval was successful",
						},
						"integration": map[string]interface{}{
							"type":        "string",
							"description": "Integration name",
						},
						"status": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"integration": map[string]interface{}{
									"type":        "string",
									"description": "Integration name",
								},
								"version": map[string]interface{}{
									"type":        "string",
									"description": "Integration version",
								},
								"status": map[string]interface{}{
									"type":        "string",
									"enum":        []string{"building", "completed", "failed"},
									"description": "Current build status",
								},
								"site_packages": map[string]interface{}{
									"type":        "string",
									"description": "Path to integration site-packages directory",
								},
								"dependencies": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"package": map[string]interface{}{
												"type":        "string",
												"description": "Package name and version",
											},
											"status": map[string]interface{}{
												"type":        "string",
												"description": "Installation status",
											},
											"location": map[string]interface{}{
												"type":        "string",
												"description": "Installation location",
											},
										},
									},
									"description": "List of installed dependencies",
								},
							},
							"description": "Build status details",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"IntegrationBuildStatusListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the status retrieval was successful",
						},
						"status": map[string]interface{}{
							"type": "object",
							"additionalProperties": map[string]interface{}{
								"$ref": "#/components/schemas/IntegrationBuildStatusResponse/properties/status",
							},
							"description": "Build status for all integrations",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ClusterInfo": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"node_id": map[string]interface{}{
							"type":        "string",
							"description": "Current node identifier",
						},
						"total_nodes": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of nodes in cluster",
						},
						"active_nodes": map[string]interface{}{
							"type":        "integer",
							"description": "Number of active nodes",
						},
						"running_jobs": map[string]interface{}{
							"type":        "integer",
							"description": "Number of currently running jobs",
						},
						"queued_jobs": map[string]interface{}{
							"type":        "integer",
							"description": "Number of jobs in queue",
						},
						"nodes": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ClusterNode",
							},
							"description": "List of cluster nodes",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Cluster status",
						},
						"config": map[string]interface{}{
							"type":        "object",
							"description": "Cluster configuration",
						},
					},
				},
				"ClusterNode": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Node identifier",
						},
						"host": map[string]interface{}{
							"type":        "string",
							"description": "Node hostname",
						},
						"port": map[string]interface{}{
							"type":        "integer",
							"description": "Node port",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Node status",
						},
						"jobs_running": map[string]interface{}{
							"type":        "integer",
							"description": "Jobs currently running on node",
						},
						"jobs_complete": map[string]interface{}{
							"type":        "integer",
							"description": "Total jobs completed by node",
						},
						"load_average": map[string]interface{}{
							"type":        "number",
							"description": "Node load average",
						},
					},
				},
				"JobSubmitRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"playbook"},
					"properties": map[string]interface{}{
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook rules to execute",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Execution context variables",
						},
					},
				},
				"JobSubmitResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether job was submitted successfully",
						},
						"job_id": map[string]interface{}{
							"type":        "string",
							"description": "Unique job identifier",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
					},
				},
				"Job": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Job identifier",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Job status (pending, running, completed, failed)",
						},
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook being executed",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Job execution context",
						},
						"results": map[string]interface{}{
							"type":        "object",
							"description": "Job execution results",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "Error message if job failed",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job creation timestamp",
						},
						"started_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job start timestamp",
						},
						"completed_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job completion timestamp",
						},
					},
				},
				"AsyncPlaybookResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether playbook was submitted successfully",
							"example":     true,
						},
						"job_id": map[string]interface{}{
							"type":        "string",
							"description": "Job ID for tracking async execution",
							"example":     "e6eb4383-2a9c-4e1b-9106-d2dee28ca009",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
							"example":     "Playbook submitted for asynchronous execution",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
							"example":     "2025-08-15T05:45:04Z",
						},
					},
					"required": []string{"success", "job_id", "message", "timestamp"},
					"example": map[string]interface{}{
						"success":   true,
						"job_id":    "e6eb4383-2a9c-4e1b-9106-d2dee28ca009",
						"message":   "Playbook submitted for asynchronous execution",
						"timestamp": "2025-08-15T05:45:04Z",
					},
				},
				"UploadResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether upload was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "Uploaded file name",
						},
						"playbook_name": map[string]interface{}{
							"type":        "string",
							"description": "Saved playbook name",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "File size in bytes",
						},
					},
				},
				"PlaybooksListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of playbooks",
						},
						"playbooks": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/PlaybookInfo",
							},
							"description": "List of playbooks",
						},
					},
				},
				"PlaybookInfo": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Playbook name",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "File name",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "File size in bytes",
						},
						"rule_count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of rules in playbook",
						},
						"modified_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last modification timestamp",
						},
						"is_valid": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether playbook is valid JSON",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Playbook description",
						},
						"author": map[string]interface{}{
							"type":        "string",
							"description": "Playbook author",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "Playbook version",
						},
						"tags": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Playbook tags",
						},
					},
				},
				"AutomationInfo": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Automation script name",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "File name with extension",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "File size in bytes",
						},
						"file_type": map[string]interface{}{
							"type":        "string",
							"description": "Type of file (automation, integration, script)",
						},
						"language": map[string]interface{}{
							"type":        "string",
							"description": "Programming language",
						},
						"line_count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of lines in script",
						},
						"function_count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of function definitions",
						},
						"import_count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of import statements",
						},
						"modified_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last modification timestamp",
						},
						"is_valid": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether script has valid syntax",
						},
					},
				},
				"AutomationListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"automations": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/AutomationInfo",
							},
							"description": "List of automation scripts",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of automations",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"AutomationUploadResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether upload was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"filename": map[string]interface{}{
							"type":        "string",
							"description": "Uploaded file name",
						},
						"automation_name": map[string]interface{}{
							"type":        "string",
							"description": "Saved automation name",
						},
						"size": map[string]interface{}{
							"type":        "integer",
							"description": "File size in bytes",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Upload timestamp",
						},
					},
				},
				"AutomationDeleteResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether deletion was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"automation_name": map[string]interface{}{
							"type":        "string",
							"description": "Deleted automation name",
						},
						"dependencies": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Dependencies that were affected",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Deletion timestamp",
						},
					},
				},
				"AutomationMetadata": map[string]interface{}{
					"type":     "object",
					"required": []string{"name", "description", "version"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Automation script name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Automation description",
						},
						"version": map[string]interface{}{
							"type":        "string",
							"description": "Automation version",
						},
						"author": map[string]interface{}{
							"type":        "string",
							"description": "Automation author",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Automation category",
						},
						"tags": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Automation tags",
						},
						"parameters": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/AutomationParameter",
							},
							"description": "Automation parameters",
						},
						"dependencies": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Required dependencies",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Creation timestamp",
						},
						"updated_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last update timestamp",
						},
						"config": map[string]interface{}{
							"type":        "object",
							"description": "Additional configuration",
						},
					},
				},
				"AutomationParameter": map[string]interface{}{
					"type":     "object",
					"required": []string{"name", "type"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Parameter name",
						},
						"type": map[string]interface{}{
							"type":        "string",
							"description": "Parameter type (string, integer, boolean, object, array)",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Parameter description",
						},
						"required": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether parameter is required",
						},
						"default": map[string]interface{}{
							"description": "Default parameter value",
						},
					},
				},
				"AutomationMetadataResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"metadata": map[string]interface{}{
							"$ref":        "#/components/schemas/AutomationMetadata",
							"description": "Automation metadata",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"AutomationMetadataListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"metadata": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/AutomationMetadata",
							},
							"description": "List of automation metadata",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"JobCreateRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"playbook"},
					"properties": map[string]interface{}{
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook rules to execute",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Execution context variables",
						},
						"priority": map[string]interface{}{
							"type":        "integer",
							"description": "Job priority (higher numbers = higher priority)",
							"minimum":     1,
							"maximum":     10,
							"default":     1,
						},
					},
				},
				"JobUpdateRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"status"},
					"properties": map[string]interface{}{
						"status": map[string]interface{}{
							"type":        "string",
							"description": "New job status",
							"enum":        []string{"pending", "running", "completed", "failed"},
						},
					},
				},
				"JobResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"job": map[string]interface{}{
							"$ref":        "#/components/schemas/JobDetails",
							"description": "Job details",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"JobListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"jobs": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/JobDetails",
							},
							"description": "List of jobs",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of jobs returned",
						},
						"filters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Status filter applied",
								},
								"limit": map[string]interface{}{
									"type":        "integer",
									"description": "Limit applied",
								},
							},
							"description": "Applied filters",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"JobStatsResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"stats": map[string]interface{}{
							"$ref":        "#/components/schemas/JobStatistics",
							"description": "Job statistics",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"JobDeleteResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether deletion was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"job_id": map[string]interface{}{
							"type":        "string",
							"description": "ID of deleted job",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"JobDetails": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique job identifier",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Current job status",
							"enum":        []string{"pending", "running", "completed", "failed"},
						},
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook being executed",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Job execution context",
						},
						"results": map[string]interface{}{
							"type":        "object",
							"description": "Job execution results",
						},
						"error": map[string]interface{}{
							"type":        "string",
							"description": "Error message if job failed",
						},
						"priority": map[string]interface{}{
							"type":        "integer",
							"description": "Job priority",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job creation timestamp",
						},
						"started_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job start timestamp",
						},
						"completed_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Job completion timestamp",
						},
					},
				},
				"JobStatistics": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"total_jobs": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of jobs",
						},
						"completed": map[string]interface{}{
							"type":        "integer",
							"description": "Number of completed jobs",
						},
						"failed": map[string]interface{}{
							"type":        "integer",
							"description": "Number of failed jobs",
						},
						"running": map[string]interface{}{
							"type":        "integer",
							"description": "Number of currently running jobs",
						},
						"pending": map[string]interface{}{
							"type":        "integer",
							"description": "Number of pending jobs",
						},
						"avg_duration_seconds": map[string]interface{}{
							"type":        "number",
							"description": "Average job execution duration in seconds",
						},
						"recent_jobs": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/JobDetails",
							},
							"description": "Recent jobs for analysis",
						},
					},
				},
				"ScheduleCreateRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"name", "cron_expr", "playbook"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Schedule name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Schedule description",
						},
						"cron_expr": map[string]interface{}{
							"type":        "string",
							"description": "Cron expression (e.g., '0 */5 * * * *' for every 5 minutes)",
						},
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook to execute",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Execution context variables",
						},
						"enabled": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether schedule is enabled",
							"default":     true,
						},
					},
				},
				"ScheduleUpdateRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Schedule name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Schedule description",
						},
						"cron_expr": map[string]interface{}{
							"type":        "string",
							"description": "Cron expression",
						},
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook to execute",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Execution context variables",
						},
						"enabled": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether schedule is enabled",
						},
					},
				},
				"ScheduleResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"schedule": map[string]interface{}{
							"$ref":        "#/components/schemas/ScheduleDetails",
							"description": "Schedule details",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ScheduleListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"schedules": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/ScheduleDetails",
							},
							"description": "List of schedules",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of schedules",
						},
						"filters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Status filter applied",
								},
							},
							"description": "Applied filters",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ScheduleStatsResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"stats": map[string]interface{}{
							"$ref":        "#/components/schemas/ScheduleStatistics",
							"description": "Schedule statistics",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ScheduleDeleteResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether deletion was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"schedule_id": map[string]interface{}{
							"type":        "string",
							"description": "ID of deleted schedule",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ScheduleExecuteResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether execution was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"schedule_id": map[string]interface{}{
							"type":        "string",
							"description": "Schedule ID",
						},
						"execution_time": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Manual execution timestamp",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ScheduleDetails": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Unique schedule identifier",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Schedule name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Schedule description",
						},
						"cron_expr": map[string]interface{}{
							"type":        "string",
							"description": "Cron expression",
						},
						"playbook": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{},
							"description": "Playbook to execute",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Execution context variables",
						},
						"enabled": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether schedule is enabled",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Current schedule status",
							"enum":        []string{"created", "scheduled", "running", "completed", "failed"},
						},
						"next_run": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Next scheduled execution time",
						},
						"last_run": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last execution time",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Schedule creation timestamp",
						},
						"updated_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last update timestamp",
						},
					},
				},
				"ScheduleStatistics": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"total": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of schedules",
						},
						"enabled": map[string]interface{}{
							"type":        "integer",
							"description": "Number of enabled schedules",
						},
						"disabled": map[string]interface{}{
							"type":        "integer",
							"description": "Number of disabled schedules",
						},
						"running": map[string]interface{}{
							"type":        "integer",
							"description": "Number of currently running schedules",
						},
						"failed": map[string]interface{}{
							"type":        "integer",
							"description": "Number of failed schedules",
						},
					},
				},
				"APIKeyCreateRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"name"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "API key name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "API key description",
						},
					},
				},
				"APIKeyCreateResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"api_key": map[string]interface{}{
							"$ref":        "#/components/schemas/APIKey",
							"description": "Created API key details",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"APIKeyListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"api_keys": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"$ref": "#/components/schemas/APIKeySummary",
							},
							"description": "List of API keys",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of API keys",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"APIKeyStatsResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"stats": map[string]interface{}{
							"$ref":        "#/components/schemas/APIKeyStats",
							"description": "API key statistics",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"APIKey": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "API key value (only shown when created)",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "API key name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "API key description",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Creation timestamp",
						},
						"created_by": map[string]interface{}{
							"type":        "string",
							"description": "Creator of the API key",
						},
						"active": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether API key is active",
						},
						"source": map[string]interface{}{
							"type":        "string",
							"description": "Source of API key (config or api)",
						},
						"last_used": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last usage timestamp",
						},
					},
				},
				"APIKeySummary": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key_prefix": map[string]interface{}{
							"type":        "string",
							"description": "API key prefix (for identification)",
						},
						"name": map[string]interface{}{
							"type":        "string",
							"description": "API key name",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "API key description",
						},
						"created_at": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Creation timestamp",
						},
						"created_by": map[string]interface{}{
							"type":        "string",
							"description": "Creator of the API key",
						},
						"active": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether API key is active",
						},
						"source": map[string]interface{}{
							"type":        "string",
							"description": "Source of API key (config or api)",
						},
						"last_used": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Last usage timestamp",
						},
					},
				},
				"APIKeyStats": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"total": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of API keys",
						},
						"active": map[string]interface{}{
							"type":        "integer",
							"description": "Number of active API keys",
						},
						"inactive": map[string]interface{}{
							"type":        "integer",
							"description": "Number of inactive API keys",
						},
						"config_keys": map[string]interface{}{
							"type":        "integer",
							"description": "Number of configuration API keys",
						},
						"generated_keys": map[string]interface{}{
							"type":        "integer",
							"description": "Number of API-generated keys",
						},
					},
				},
				"ClientIntegrationConfigRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"name"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Integration name",
							"example":     "postgresql",
						},
						"config": map[string]interface{}{
							"type":        "object",
							"description": "Integration configuration parameters",
							"properties": map[string]interface{}{
								"host": map[string]interface{}{
									"type":        "string",
									"description": "Database host",
									"example":     "localhost",
								},
								"port": map[string]interface{}{
									"type":        "integer",
									"description": "Database port",
									"example":     5432,
								},
								"database": map[string]interface{}{
									"type":        "string",
									"description": "Database name",
									"example":     "mydb",
								},
								"username": map[string]interface{}{
									"type":        "string",
									"description": "Database username",
									"example":     "postgres",
								},
								"password": map[string]interface{}{
									"type":        "string",
									"description": "Database password",
									"example":     "password",
								},
								"ssl_mode": map[string]interface{}{
									"type":        "string",
									"description": "SSL connection mode",
									"example":     "prefer",
									"enum":        []string{"disable", "allow", "prefer", "require"},
								},
							},
						},
						"credentials": map[string]interface{}{
							"type":        "object",
							"description": "Encrypted credentials (optional, can be in config)",
							"properties": map[string]interface{}{
								"username": map[string]interface{}{
									"type":        "string",
									"description": "Database username",
								},
								"password": map[string]interface{}{
									"type":        "string",
									"description": "Database password",
								},
							},
						},
					},
				},
				"ClientIntegrationConfigResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"config": map[string]interface{}{
							"type":        "object",
							"description": "Integration configuration (credentials excluded)",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"IntegrationExecuteRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"function"},
					"properties": map[string]interface{}{
						"function": map[string]interface{}{
							"type":        "string",
							"description": "Function name to execute",
							"example":     "test_connection",
						},
						"params": map[string]interface{}{
							"type":        "object",
							"description": "Function parameters",
							"example": map[string]interface{}{
								"table":  "users",
								"limit":  10,
								"offset": 0,
							},
						},
					},
				},
				"IntegrationExecuteResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether execution was successful",
						},
						"result": map[string]interface{}{
							"type":        "object",
							"description": "Function execution result",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ClientListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"clients": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"id": map[string]interface{}{
										"type":        "string",
										"description": "Client unique identifier",
									},
									"name": map[string]interface{}{
										"type":        "string",
										"description": "Client name",
									},
									"description": map[string]interface{}{
										"type":        "string",
										"description": "Client description",
									},
									"created_at": map[string]interface{}{
										"type":        "string",
										"format":      "date-time",
										"description": "Client creation timestamp",
									},
									"updated_at": map[string]interface{}{
										"type":        "string",
										"format":      "date-time",
										"description": "Client last update timestamp",
									},
									"status": map[string]interface{}{
										"type":        "string",
										"description": "Client status",
										"enum":        []string{"active", "inactive", "suspended"},
									},
								},
							},
							"description": "List of clients in the system",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Total number of clients",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
					"required": []string{"success", "message", "clients", "count", "timestamp"},
				},
				"ClientResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"client": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id": map[string]interface{}{
									"type":        "string",
									"description": "Client unique identifier",
								},
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Client name",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Client description",
								},
								"active": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether the client is active",
								},
								"api_keys": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "string",
									},
									"description": "Client API keys (partial for security)",
								},
								"encryption_key_id": map[string]interface{}{
									"type":        "string",
									"description": "Client-specific encryption key ID",
								},
								"rate_limits": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"requests_per_minute": map[string]interface{}{
											"type":        "integer",
											"description": "Rate limit requests per minute",
										},
										"burst_limit": map[string]interface{}{
											"type":        "integer",
											"description": "Rate limit burst capacity",
										},
										"enabled": map[string]interface{}{
											"type":        "boolean",
											"description": "Whether rate limiting is enabled",
										},
									},
									"description": "Client rate limiting configuration",
								},
								"metadata": map[string]interface{}{
									"type":        "object",
									"description": "Client-specific metadata (searchable)",
								},
								"created_at": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Client creation timestamp",
								},
								"updated_at": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Client last update timestamp",
								},
								"last_accessed_at": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Client last access timestamp",
								},
								"integration_count": map[string]interface{}{
									"type":        "integer",
									"description": "Number of configured integrations",
								},
							},
							"description": "Database-backed client information with full details",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
					"required": []string{"success", "message", "client", "timestamp"},
				},
				"ClientCreateResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether client creation was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"client": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id": map[string]interface{}{
									"type":        "string",
									"description": "Newly created client unique identifier",
								},
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Client name",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Client description",
								},
								"created_at": map[string]interface{}{
									"type":        "string",
									"format":      "date-time",
									"description": "Client creation timestamp",
								},
								"status": map[string]interface{}{
									"type":        "string",
									"description": "Client status",
									"enum":        []string{"active", "inactive", "suspended"},
								},
							},
							"description": "Newly created client information",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
					"required": []string{"success", "message", "client", "timestamp"},
				},
				"ClientIntegrationListResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether request was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"integrations": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name": map[string]interface{}{
										"type":        "string",
										"description": "Integration name",
									},
									"configured": map[string]interface{}{
										"type":        "boolean",
										"description": "Whether integration is configured for this client",
									},
									"last_used": map[string]interface{}{
										"type":        "string",
										"format":      "date-time",
										"description": "Last usage timestamp",
									},
								},
							},
							"description": "List of client integrations",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of integrations",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
				},
				"ClientCreateRequest": map[string]interface{}{
					"type":     "object",
					"required": []string{"name"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Client name (required)",
							"minLength":   1,
							"maxLength":   255,
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Client description (optional)",
							"maxLength":   1000,
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Client-specific metadata (optional, searchable)",
						},
					},
				},
				"ClientUpdateRequest": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Updated client name",
							"minLength":   1,
							"maxLength":   255,
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Updated client description",
							"maxLength":   1000,
						},
						"active": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether the client should be active",
						},
						"metadata": map[string]interface{}{
							"type":        "object",
							"description": "Updated client metadata (searchable)",
						},
						"rate_limits": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"requests_per_minute": map[string]interface{}{
									"type":        "integer",
									"description": "Rate limit requests per minute",
									"minimum":     1,
									"maximum":     10000,
								},
								"burst_limit": map[string]interface{}{
									"type":        "integer",
									"description": "Rate limit burst capacity",
									"minimum":     1,
									"maximum":     1000,
								},
								"enabled": map[string]interface{}{
									"type":        "boolean",
									"description": "Whether rate limiting is enabled",
								},
							},
							"description": "Updated rate limiting configuration",
						},
					},
				},
				"ClientSearchResponse": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"success": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether search was successful",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Response message",
						},
						"clients": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"id": map[string]interface{}{
										"type":        "string",
										"description": "Client unique identifier",
									},
									"name": map[string]interface{}{
										"type":        "string",
										"description": "Client name",
									},
									"description": map[string]interface{}{
										"type":        "string",
										"description": "Client description",
									},
									"active": map[string]interface{}{
										"type":        "boolean",
										"description": "Whether the client is active",
									},
									"metadata": map[string]interface{}{
										"type":        "object",
										"description": "Client metadata that matched search",
									},
									"created_at": map[string]interface{}{
										"type":        "string",
										"format":      "date-time",
										"description": "Client creation timestamp",
									},
									"integration_count": map[string]interface{}{
										"type":        "integer",
										"description": "Number of configured integrations",
									},
								},
							},
							"description": "Search results with matching clients",
						},
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Original search query",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"description": "Number of matching clients found",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum results requested",
						},
						"timestamp": map[string]interface{}{
							"type":        "string",
							"format":      "date-time",
							"description": "Response timestamp",
						},
					},
					"required": []string{"success", "message", "clients", "query", "count", "limit", "timestamp"},
				},
			},
		},
	}

	return json.MarshalIndent(spec, "", "  ")
}
