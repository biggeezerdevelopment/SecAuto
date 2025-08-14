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

// readOpenAPISpec generates the OpenAPI specification for the modular build
func readOpenAPISpec(serverPort string) ([]byte, error) {
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "SecAuto Modular SOAR API",
			"description": "A modular Security Orchestration, Automation, and Response (SOAR) platform API. This modular implementation provides core functionality for playbook execution, caching, and Redis list operations.",
			"version":     "modular-1.0.0",
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
		},
		"components": map[string]interface{}{
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
							"description": "Direct playbook rules array or single rule",
						},
						"playbook_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of playbook file to load",
						},
						"context": map[string]interface{}{
							"type":        "object",
							"description": "Context data for playbook execution",
							"additionalProperties": true,
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
							"type":        "object",
							"description": "Redis cache statistics",
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
							"type": "array",
							"items": map[string]interface{}{},
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
					"type": "object",
					"required": []string{"items"},
					"properties": map[string]interface{}{
						"items": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{},
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
					"type": "object",
					"required": []string{"items"},
					"properties": map[string]interface{}{
						"items": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{},
							"description": "Items to remove from the list",
						},
						"count": map[string]interface{}{
							"type":        "integer",
							"default":     1,
							"description": "Number of occurrences to remove per item",
						},
					},
				},
			},
		},
	}

	return json.MarshalIndent(spec, "", "  ")
}