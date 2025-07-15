package main

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
func NewSwaggerUIHandler() (*SwaggerUIHandler, error) {
	// Read the OpenAPI specification
	spec, err := readOpenAPISpec()
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
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SecAuto API Documentation</title>
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
                url: '/api-docs',
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
                validatorUrl: null,
                onComplete: function() {
                    console.log('SecAuto API Documentation loaded successfully');
                },
                onFailure: function(data) {
                    console.error('Failed to load API documentation:', data);
                }
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// readOpenAPISpec reads the OpenAPI specification from file
func readOpenAPISpec() ([]byte, error) {
	// For now, we'll return a basic spec
	// In a real implementation, you would read from openapi.yaml
	spec := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":       "SecAuto SOAR Rules Engine API",
			"description": "A powerful Security Orchestration, Automation, and Response (SOAR) rules engine API. This API provides comprehensive endpoints for job management, distributed clustering, plugin system, job scheduling, and automation workflows.",
			"version":     "1.0.0",
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
				"url":         "http://localhost:8080",
				"description": "Development server",
			},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health Check",
					"description": "Check the health status of the SecAuto system",
					"tags":        []string{"Health"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "System is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type": "string",
												"enum": []string{"healthy", "unhealthy"},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
											"version": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/playbook": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Playbook Synchronously",
					"description": "Execute a playbook immediately and return results",
					"tags":        []string{"Playbooks"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"playbook": map[string]interface{}{
											"type":        "array",
											"items":       map[string]interface{}{"type": "object"},
											"description": "Array of playbook rules",
										},
										"context": map[string]interface{}{
											"type":        "object",
											"description": "Initial context data",
										},
										"options": map[string]interface{}{
											"type":        "object",
											"description": "Execution options",
											"properties": map[string]interface{}{
												"timeout": map[string]interface{}{
													"type":        "integer",
													"description": "Execution timeout in seconds",
												},
												"priority": map[string]interface{}{
													"type":        "string",
													"enum":        []string{"low", "normal", "high", "critical"},
													"description": "Job priority",
												},
											},
										},
									},
									"required": []string{"playbook"},
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"results": map[string]interface{}{
												"type": "array",
											},
											"context": map[string]interface{}{
												"type": "object",
											},
											"timestamp": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid request",
						},
					},
				},
			},
			"/playbook/async": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Execute Playbook Asynchronously",
					"description": "Submit a playbook for asynchronous execution",
					"tags":        []string{"Playbooks"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"playbook": map[string]interface{}{
											"type":  "array",
											"items": map[string]interface{}{"type": "object"},
										},
										"context": map[string]interface{}{
											"type": "object",
										},
									},
									"required": []string{"playbook"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"202": map[string]interface{}{
							"description": "Job submitted successfully",
						},
						"400": map[string]interface{}{
							"description": "Invalid request",
						},
					},
				},
			},
			"/jobs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Jobs",
					"description": "Retrieve a list of all jobs with optional filtering and pagination",
					"tags":        []string{"Jobs"},
					"parameters": []map[string]interface{}{
						{
							"name":        "status",
							"in":          "query",
							"description": "Filter by job status",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"pending", "running", "completed", "failed", "cancelled"},
							},
						},
						{
							"name":        "limit",
							"in":          "query",
							"description": "Maximum number of jobs to return",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 50,
								"maximum": 100,
							},
						},
						{
							"name":        "offset",
							"in":          "query",
							"description": "Number of jobs to skip",
							"schema": map[string]interface{}{
								"type":    "integer",
								"default": 0,
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Jobs retrieved successfully",
						},
					},
				},
			},
			"/jobs/stats": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Job Statistics",
					"description": "Get comprehensive job statistics and metrics",
					"tags":        []string{"Jobs"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Statistics retrieved successfully",
						},
					},
				},
			},
			"/jobs/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Database Performance Metrics",
					"description": "Get database performance metrics and connection pool statistics",
					"tags":        []string{"Jobs"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Metrics retrieved successfully",
						},
					},
				},
			},
			"/plugins": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Plugins",
					"description": "Retrieve a list of all available plugins",
					"tags":        []string{"Plugins"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Plugins retrieved successfully",
						},
					},
				},
			},
			"/automation": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Automation Script",
					"description": "Upload a new automation script to the system. The script will be validated for security and stored in the automations directory.",
					"tags":        []string{"Automations"},
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
											"description": "Python automation script file (.py)",
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"automation_name": map[string]interface{}{
												"type": "string",
											},
											"filename": map[string]interface{}{
												"type": "string",
											},
											"size": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"errors": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"field": map[string]interface{}{
															"type": "string",
														},
														"message": map[string]interface{}{
															"type": "string",
														},
														"value": map[string]interface{}{
															"type": "string",
														},
													},
												},
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/playbook/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Playbook File",
					"description": "Upload a new playbook file to the system. The file will be validated for structure and stored in the playbooks directory.",
					"tags":        []string{"Playbooks"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"playbook": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Playbook JSON file (.json)",
										},
									},
									"required": []string{"playbook"},
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"playbook_name": map[string]interface{}{
												"type": "string",
											},
											"filename": map[string]interface{}{
												"type": "string",
											},
											"size": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"errors": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"field": map[string]interface{}{
															"type": "string",
														},
														"message": map[string]interface{}{
															"type": "string",
														},
														"value": map[string]interface{}{
															"type": "string",
														},
													},
												},
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/playbooks": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Playbooks",
					"description": "Retrieve a list of all available playbooks with their metadata including rule counts, operation types, and modification dates.",
					"tags":        []string{"Playbooks"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Playbooks retrieved successfully",
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
											"playbooks": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"name": map[string]interface{}{
															"type": "string",
														},
														"filename": map[string]interface{}{
															"type": "string",
														},
														"size": map[string]interface{}{
															"type": "integer",
														},
														"rule_count": map[string]interface{}{
															"type": "integer",
														},
														"operations": map[string]interface{}{
															"type": "object",
															"additionalProperties": map[string]interface{}{
																"type": "integer",
															},
														},
														"modified_at": map[string]interface{}{
															"type":   "string",
															"format": "date-time",
														},
														"is_valid": map[string]interface{}{
															"type": "boolean",
														},
													},
												},
											},
											"count": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/automations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Automations",
					"description": "Retrieve a list of all available automation scripts with their metadata including file type, language, line count, function count, and modification dates.",
					"tags":        []string{"Automations"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Automations retrieved successfully",
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
											"automations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"name": map[string]interface{}{
															"type": "string",
														},
														"filename": map[string]interface{}{
															"type": "string",
														},
														"size": map[string]interface{}{
															"type": "integer",
														},
														"file_type": map[string]interface{}{
															"type": "string",
														},
														"language": map[string]interface{}{
															"type": "string",
														},
														"line_count": map[string]interface{}{
															"type": "integer",
														},
														"function_count": map[string]interface{}{
															"type": "integer",
														},
														"import_count": map[string]interface{}{
															"type": "integer",
														},
														"modified_at": map[string]interface{}{
															"type":   "string",
															"format": "date-time",
														},
														"is_valid": map[string]interface{}{
															"type": "boolean",
														},
													},
												},
											},
											"count": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"500": map[string]interface{}{
							"description": "Internal server error",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/automation/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "Delete Automation",
					"description": "Delete an automation script. If the automation is used by any playbooks, the deletion will be blocked and the dependent playbooks will be listed.",
					"tags":        []string{"Automations"},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Name of the automation to delete",
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"automation_name": map[string]interface{}{
												"type": "string",
											},
											"dependencies": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"409": map[string]interface{}{
							"description": "Automation cannot be deleted - dependencies exist",
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
											"automation_name": map[string]interface{}{
												"type": "string",
											},
											"dependencies": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Automation not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error": map[string]interface{}{
												"type": "string",
											},
										},
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
					"description": "Delete a playbook file from the system.",
					"tags":        []string{"Playbooks"},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Name of the playbook to delete",
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
											"playbook_name": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Playbook not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/plugin/{type}/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "Delete Plugin",
					"description": "Delete a plugin file from the system for a specific platform type.",
					"tags":        []string{"Plugins"},
					"parameters": []map[string]interface{}{
						{
							"name":        "type",
							"in":          "path",
							"required":    true,
							"description": "Plugin type (linux, windows, python, go)",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"linux", "windows", "python", "go"},
							},
						},
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Name of the plugin to delete",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Plugin deleted successfully",
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
											"plugin_name": map[string]interface{}{
												"type": "string",
											},
											"plugin_type": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"404": map[string]interface{}{
							"description": "Plugin not found",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"error": map[string]interface{}{
												"type": "string",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/cluster": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Cluster Information",
					"description": "Retrieve information about the distributed cluster",
					"tags":        []string{"Cluster"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Cluster information retrieved successfully",
						},
					},
				},
			},
			"/schedules": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Schedules",
					"description": "Retrieve a list of all job schedules",
					"tags":        []string{"Schedules"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Schedules retrieved successfully",
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create New Schedule",
					"description": "Create a new job schedule",
					"tags":        []string{"Schedules"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
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
										"schedule_type": map[string]interface{}{
											"type":        "string",
											"enum":        []string{"cron", "interval", "one_time"},
											"description": "Type of schedule",
										},
										"cron_expression": map[string]interface{}{
											"type":        "string",
											"description": "Cron expression (for cron type)",
										},
										"interval_seconds": map[string]interface{}{
											"type":        "integer",
											"description": "Interval in seconds (for interval type)",
										},
										"playbook": map[string]interface{}{
											"type":        "array",
											"items":       map[string]interface{}{"type": "object"},
											"description": "Playbook rules to execute",
										},
										"context": map[string]interface{}{
											"type":        "object",
											"description": "Initial context for the job",
										},
										"status": map[string]interface{}{
											"type":        "string",
											"enum":        []string{"active", "inactive"},
											"description": "Schedule status",
										},
									},
									"required": []string{"name", "schedule_type", "playbook"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Schedule created successfully",
						},
						"400": map[string]interface{}{
							"description": "Invalid schedule configuration",
						},
					},
				},
			},
			"/webhooks": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Configure Webhooks",
					"description": "Configure webhook notifications for job events",
					"tags":        []string{"Webhooks"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"url": map[string]interface{}{
											"type":        "string",
											"format":      "uri",
											"description": "Webhook URL",
										},
										"events": map[string]interface{}{
											"type": "array",
											"items": map[string]interface{}{
												"type": "string",
												"enum": []string{"job_started", "job_completed", "job_failed", "job_cancelled"},
											},
											"description": "Events to trigger webhook",
										},
										"headers": map[string]interface{}{
											"type":        "object",
											"description": "Custom headers",
										},
										"timeout": map[string]interface{}{
											"type":        "integer",
											"description": "Request timeout in seconds",
										},
										"retry_count": map[string]interface{}{
											"type":        "integer",
											"description": "Number of retry attempts",
										},
									},
									"required": []string{"url", "events"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Webhook configured successfully",
						},
						"400": map[string]interface{}{
							"description": "Invalid webhook configuration",
						},
					},
				},
			},
			"/validate": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Validate Playbook/Context",
					"description": "Validate a playbook and context without executing",
					"tags":        []string{"Validation"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"playbook": map[string]interface{}{
											"type":        "array",
											"items":       map[string]interface{}{"type": "object"},
											"description": "Playbook rules to validate",
										},
										"context": map[string]interface{}{
											"type":        "object",
											"description": "Context to validate",
										},
									},
									"required": []string{"playbook"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Validation completed successfully",
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
						},
					},
				},
			},
			"/plugin/{type}": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Plugin File",
					"description": "Upload a new plugin file to the system for a specific platform type (linux, windows, python, go). The file will be validated and stored in the appropriate plugins directory.",
					"tags":        []string{"Plugins"},
					"parameters": []map[string]interface{}{
						{
							"name":        "type",
							"in":          "path",
							"required":    true,
							"description": "Plugin type (linux, windows, python, go)",
							"schema": map[string]interface{}{
								"type": "string",
								"enum": []string{"linux", "windows", "python", "go"},
							},
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"plugin": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Plugin file (binary or script)",
										},
									},
									"required": []string{"plugin"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Plugin uploaded successfully",
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
											"plugin_name": map[string]interface{}{
												"type": "string",
											},
											"plugin_type": map[string]interface{}{
												"type": "string",
											},
											"filename": map[string]interface{}{
												"type": "string",
											},
											"size": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"errors": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"field": map[string]interface{}{
															"type": "string",
														},
														"message": map[string]interface{}{
															"type": "string",
														},
														"value": map[string]interface{}{
															"type": "string",
														},
													},
												},
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/integrations": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "List All Integrations",
					"description": "Retrieve a list of all configured integrations with their status and basic information",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integrations retrieved successfully",
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
											"integrations": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"name": map[string]interface{}{
															"type": "string",
														},
														"type": map[string]interface{}{
															"type": "string",
														},
														"enabled": map[string]interface{}{
															"type": "boolean",
														},
														"description": map[string]interface{}{
															"type": "string",
														},
														"version": map[string]interface{}{
															"type": "string",
														},
														"created_at": map[string]interface{}{
															"type":   "string",
															"format": "date-time",
														},
														"updated_at": map[string]interface{}{
															"type":   "string",
															"format": "date-time",
														},
													},
												},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
				"post": map[string]interface{}{
					"summary":     "Create Integration",
					"description": "Create a new integration configuration with encrypted credentials",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name": map[string]interface{}{
											"type":        "string",
											"description": "Integration name",
										},
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Integration type (virustotal, slack, email, etc.)",
										},
										"url": map[string]interface{}{
											"type":        "string",
											"description": "Integration API URL",
										},
										"apikey": map[string]interface{}{
											"type":        "string",
											"description": "API key (will be encrypted)",
										},
										"username": map[string]interface{}{
											"type":        "string",
											"description": "Username for authentication",
										},
										"password": map[string]interface{}{
											"type":        "string",
											"description": "Password (will be encrypted)",
										},
										"token": map[string]interface{}{
											"type":        "string",
											"description": "Access token (will be encrypted)",
										},
										"secret": map[string]interface{}{
											"type":        "string",
											"description": "Secret key (will be encrypted)",
										},
										"enabled": map[string]interface{}{
											"type":        "boolean",
											"description": "Whether the integration is enabled",
										},
										"description": map[string]interface{}{
											"type":        "string",
											"description": "Integration description",
										},
										"version": map[string]interface{}{
											"type":        "string",
											"description": "Integration version",
										},
										"settings": map[string]interface{}{
											"type":        "object",
											"description": "Additional configuration settings",
										},
									},
									"required": []string{"name", "type"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration created successfully",
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
											"integration": map[string]interface{}{
												"type": "object",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
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
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/integrations/upload": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Integration File",
					"description": "Upload a Python integration module file to the integrations directory. The file will be validated and stored for use in automations.",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"integration": map[string]interface{}{
											"type":        "string",
											"format":      "binary",
											"description": "Python integration file (.py)",
										},
									},
									"required": []string{"integration"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration uploaded successfully",
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
											"integration_name": map[string]interface{}{
												"type": "string",
											},
											"filename": map[string]interface{}{
												"type": "string",
											},
											"size": map[string]interface{}{
												"type": "integer",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"errors": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "object",
													"properties": map[string]interface{}{
														"field": map[string]interface{}{
															"type": "string",
														},
														"message": map[string]interface{}{
															"type": "string",
														},
														"value": map[string]interface{}{
															"type": "string",
														},
													},
												},
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/integrations/delete/{name}": map[string]interface{}{
				"delete": map[string]interface{}{
					"summary":     "Delete Integration File",
					"description": "Delete a Python integration module file from the integrations directory. Checks for dependencies in automations before deletion.",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., 'virustotal', 'slack')",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration deleted successfully",
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
											"integration_name": map[string]interface{}{
												"type": "string",
											},
											"dependencies": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid integration name",
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
						"409": map[string]interface{}{
							"description": "Cannot delete integration - it is used by automations",
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
											"integration_name": map[string]interface{}{
												"type": "string",
											},
											"dependencies": map[string]interface{}{
												"type": "array",
												"items": map[string]interface{}{
													"type": "string",
												},
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/integrations/{name}": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get Integration",
					"description": "Retrieve a specific integration configuration by name",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., 'virustotal', 'slack')",
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"integration": map[string]interface{}{
												"type": "object",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
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
					"description": "Update an existing integration configuration by name",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., 'virustotal', 'slack')",
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
										"name": map[string]interface{}{
											"type":        "string",
											"description": "Integration name (optional, will use path parameter if not provided)",
										},
										"type": map[string]interface{}{
											"type":        "string",
											"description": "Integration type",
										},
										"url": map[string]interface{}{
											"type":        "string",
											"description": "Integration API URL",
										},
										"apikey": map[string]interface{}{
											"type":        "string",
											"description": "API key (will be encrypted)",
										},
										"username": map[string]interface{}{
											"type":        "string",
											"description": "Username for authentication",
										},
										"password": map[string]interface{}{
											"type":        "string",
											"description": "Password (will be encrypted)",
										},
										"token": map[string]interface{}{
											"type":        "string",
											"description": "Access token (will be encrypted)",
										},
										"secret": map[string]interface{}{
											"type":        "string",
											"description": "Secret key (will be encrypted)",
										},
										"enabled": map[string]interface{}{
											"type":        "boolean",
											"description": "Whether the integration is enabled",
										},
										"description": map[string]interface{}{
											"type":        "string",
											"description": "Integration description",
										},
										"version": map[string]interface{}{
											"type":        "string",
											"description": "Integration version",
										},
										"settings": map[string]interface{}{
											"type":        "object",
											"description": "Additional configuration settings",
										},
									},
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
										"type": "object",
										"properties": map[string]interface{}{
											"success": map[string]interface{}{
												"type": "boolean",
											},
											"message": map[string]interface{}{
												"type": "string",
											},
											"integration": map[string]interface{}{
												"type": "object",
											},
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Validation failed",
						},
						"404": map[string]interface{}{
							"description": "Integration not found",
						},
					},
				},
				"delete": map[string]interface{}{
					"summary":     "Delete Integration",
					"description": "Delete an integration configuration by name",
					"tags":        []string{"Integrations"},
					"security":    []map[string]interface{}{{"ApiKeyAuth": []string{}}},
					"parameters": []map[string]interface{}{
						{
							"name":        "name",
							"in":          "path",
							"required":    true,
							"description": "Integration name (e.g., 'virustotal', 'slack')",
							"schema": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Integration deleted successfully",
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
											"timestamp": map[string]interface{}{
												"type":   "string",
												"format": "date-time",
											},
										},
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
		},
		"security": []map[string]interface{}{
			{"ApiKeyAuth": []string{}},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "Health",
				"description": "Health monitoring endpoints",
			},
			{
				"name":        "Playbooks",
				"description": "Playbook execution endpoints",
			},
			{
				"name":        "Jobs",
				"description": "Job management endpoints",
			},
			{
				"name":        "Plugins",
				"description": "Plugin system endpoints",
			},
			{
				"name":        "Cluster",
				"description": "Distributed cluster endpoints",
			},
			{
				"name":        "Schedules",
				"description": "Job scheduling endpoints",
			},
			{
				"name":        "Webhooks",
				"description": "Webhook configuration endpoints",
			},
			{
				"name":        "Validation",
				"description": "Validation endpoints",
			},
			{
				"name":        "Automations",
				"description": "Automation script management endpoints",
			},
			{
				"name":        "Integrations",
				"description": "Integration management endpoints",
			},
		},
	}

	return json.Marshal(spec)
}
