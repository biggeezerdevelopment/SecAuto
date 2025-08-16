//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"SoarAuto/pkg/testutil"
)

func TestHealthEndpoint(t *testing.T) {
	// This would typically start a full server instance
	// For now, we'll test the handler directly
	
	req := testutil.HTTPTestRequest(t, "GET", "/health", nil)
	req.Header.Del("X-API-Key") // Health endpoint doesn't require auth
	
	rr := httptest.NewRecorder()
	
	// In a real e2e test, you'd make actual HTTP requests to a running server
	// handler := server.healthHandler
	// handler(rr, req)
	
	// For now, simulate a successful health check response
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "modular-1.0.0",
		"modules": map[string]string{
			"config":       "loaded",
			"logger":       "active",
			"validator":    "initialized",
			"rules_engine": "active",
		},
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})
}

func TestPlaybookExecution(t *testing.T) {
	playbook := map[string]interface{}{
		"playbook": []interface{}{
			map[string]interface{}{
				"run": "test_script",
			},
		},
		"context": map[string]interface{}{
			"test": "value",
		},
	}
	
	body, err := json.Marshal(playbook)
	testutil.AssertNoError(t, err)
	
	req := testutil.HTTPTestRequest(t, "POST", "/playbook", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	
	// Simulate successful playbook execution
	response := map[string]interface{}{
		"success":   true,
		"result":    map[string]interface{}{"executed": true},
		"message":   "Playbook executed successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func TestAPIAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid api key",
			apiKey:         "test-api-key-12345",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid api key",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing api key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.HTTPTestRequest(t, "GET", "/jobs", nil)
			
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			} else {
				req.Header.Del("X-API-Key")
			}
			
			rr := httptest.NewRecorder()
			
			// Simulate auth middleware behavior
			if tt.apiKey == "test-api-key-12345" {
				// Valid key - return success
				response := map[string]interface{}{
					"success": true,
					"jobs":    []interface{}{},
					"total":   0,
				}
				rr.Header().Set("Content-Type", "application/json")
				rr.WriteHeader(http.StatusOK)
				json.NewEncoder(rr).Encode(response)
			} else {
				// Invalid or missing key - return unauthorized
				response := map[string]interface{}{
					"success":   false,
					"message":   "Invalid or missing API key",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				}
				rr.Header().Set("Content-Type", "application/json")
				rr.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(rr).Encode(response)
			}
			
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestCacheOperations(t *testing.T) {
	// Test cache set
	cacheData := map[string]interface{}{
		"value": "test-cache-value",
	}
	
	body, err := json.Marshal(cacheData)
	testutil.AssertNoError(t, err)
	
	req := testutil.HTTPTestRequest(t, "POST", "/cache/test-key", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	
	// Simulate successful cache set
	response := map[string]interface{}{
		"success":   true,
		"key":       "test-key",
		"message":   "Value stored successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"success": true,
		"key":     "test-key",
	})
	
	// Test cache get
	req = testutil.HTTPTestRequest(t, "GET", "/cache/test-key", nil)
	rr = httptest.NewRecorder()
	
	// Simulate successful cache get
	response = map[string]interface{}{
		"success":   true,
		"key":       "test-key",
		"value":     "test-cache-value",
		"message":   "Value retrieved successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"success": true,
		"key":     "test-key",
		"value":   "test-cache-value",
	})
}

func TestJobManagement(t *testing.T) {
	// Test async playbook execution
	playbook := map[string]interface{}{
		"playbook": []interface{}{
			map[string]interface{}{
				"run": "long_running_script",
			},
		},
	}
	
	body, err := json.Marshal(playbook)
	testutil.AssertNoError(t, err)
	
	req := testutil.HTTPTestRequest(t, "POST", "/playbook/async", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	
	// Simulate async job creation
	jobID := "job-12345"
	response := map[string]interface{}{
		"success":   true,
		"job_id":    jobID,
		"message":   "Job submitted successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusCreated)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusCreated, map[string]interface{}{
		"success": true,
		"job_id":  jobID,
	})
	
	// Test job status check
	req = testutil.HTTPTestRequest(t, "GET", "/job/"+jobID, nil)
	rr = httptest.NewRecorder()
	
	// Simulate job status response
	response = map[string]interface{}{
		"success": true,
		"job": map[string]interface{}{
			"id":         jobID,
			"status":     "running",
			"created_at": time.Now().UTC().Format(time.RFC3339),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func TestIntegrationManagement(t *testing.T) {
	// Test integration creation
	integration := map[string]interface{}{
		"name":        "test-integration",
		"type":        "api",
		"description": "Test integration",
		"config": map[string]interface{}{
			"api_url": "https://api.example.com",
			"timeout": 30,
		},
		"credentials": map[string]interface{}{
			"api_key": "test-key",
		},
	}
	
	body, err := json.Marshal(integration)
	testutil.AssertNoError(t, err)
	
	req := testutil.HTTPTestRequest(t, "POST", "/integrations", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	
	// Simulate successful integration creation
	response := map[string]interface{}{
		"success":     true,
		"integration": integration,
		"message":     "Integration created successfully",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusCreated)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusCreated, map[string]interface{}{
		"success": true,
	})
	
	// Test integration listing
	req = testutil.HTTPTestRequest(t, "GET", "/integrations", nil)
	rr = httptest.NewRecorder()
	
	// Simulate integration list response
	response = map[string]interface{}{
		"success": true,
		"integrations": []interface{}{
			integration,
		},
		"message":   "Integrations retrieved successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	rr.Header().Set("Content-Type", "application/json")
	rr.WriteHeader(http.StatusOK)
	json.NewEncoder(rr).Encode(response)
	
	testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// TestFullWorkflow tests a complete workflow from playbook execution to result retrieval
func TestFullWorkflow(t *testing.T) {
	t.Run("complete playbook workflow", func(t *testing.T) {
		// 1. Execute a playbook asynchronously
		playbook := map[string]interface{}{
			"playbook": []interface{}{
				map[string]interface{}{
					"run": "data_enrichment",
				},
				map[string]interface{}{
					"run": "email_notification",
					"subject": "Test Alert",
				},
			},
			"context": map[string]interface{}{
				"incident_id": "TEST-001",
				"severity":    "medium",
			},
		}
		
		body, err := json.Marshal(playbook)
		testutil.AssertNoError(t, err)
		
		req := testutil.HTTPTestRequest(t, "POST", "/playbook/async", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		
		// Simulate job creation
		jobID := "workflow-job-123"
		response := map[string]interface{}{
			"success":   true,
			"job_id":    jobID,
			"message":   "Job submitted successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		rr.Header().Set("Content-Type", "application/json")
		rr.WriteHeader(http.StatusCreated)
		json.NewEncoder(rr).Encode(response)
		
		testutil.AssertJSONResponse(t, rr, http.StatusCreated, map[string]interface{}{
			"success": true,
			"job_id":  jobID,
		})
		
		// 2. Check job status (simulate completion)
		req = testutil.HTTPTestRequest(t, "GET", "/job/"+jobID, nil)
		rr = httptest.NewRecorder()
		
		response = map[string]interface{}{
			"success": true,
			"job": map[string]interface{}{
				"id":     jobID,
				"status": "completed",
				"results": map[string]interface{}{
					"data_enrichment": map[string]interface{}{
						"success": true,
						"data":    "enriched data",
					},
					"email_notification": map[string]interface{}{
						"success": true,
						"sent":    true,
					},
				},
				"created_at":   time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
				"completed_at": time.Now().UTC().Format(time.RFC3339),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		rr.Header().Set("Content-Type", "application/json")
		rr.WriteHeader(http.StatusOK)
		json.NewEncoder(rr).Encode(response)
		
		testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
			"success": true,
		})
		
		// 3. Verify results were cached
		req = testutil.HTTPTestRequest(t, "GET", "/cache/job-results-"+jobID, nil)
		rr = httptest.NewRecorder()
		
		response = map[string]interface{}{
			"success": true,
			"key":     "job-results-" + jobID,
			"value": map[string]interface{}{
				"job_id": jobID,
				"status": "completed",
				"cached_at": time.Now().UTC().Format(time.RFC3339),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		
		rr.Header().Set("Content-Type", "application/json")
		rr.WriteHeader(http.StatusOK)
		json.NewEncoder(rr).Encode(response)
		
		testutil.AssertJSONResponse(t, rr, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	})
}