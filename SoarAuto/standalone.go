package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func runStandaloneWithFlags(playbookFile, contextFile string) {
	// Load configuration
	config, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create rule engine
	engine := NewRuleEngine(config)

	// Set initial context
	initialContext := map[string]interface{}{
		"context": map[string]interface{}{
			"name":  "John Doe",
			"age":   40, // Changed to 40 to test the condition (should be > 37)
			"email": "john@example.com",
			"items": []string{"apple", "banana", "cherry"},
			"metadata": map[string]interface{}{
				"source":    "go_application",
				"timestamp": "2024-01-01T00:00:00Z",
			},
			"incident": map[string]interface{}{
				"id":          "123456",
				"name":        "Test Incident",
				"description": "This is a test incident",
			},
		},
	}

	// Load custom context if provided
	if contextFile != "" {
		customContext, err := loadContextFromFile(contextFile)
		if err != nil {
			log.Printf("Warning: Failed to load context file %s: %v", contextFile, err)
			log.Printf("Using default context")
		} else {
			initialContext = customContext
		}
	}

	engine.SetContext(initialContext)

	// Load playbook from file
	playbookData, err := loadPlaybookFromFile(playbookFile)
	if err != nil {
		log.Printf("Error loading playbook: %v", err)
		return
	}

	fmt.Println("=== Rules Engine ===")
	fmt.Printf("Playbook file: %s\n", playbookFile)
	if contextFile != "" {
		fmt.Printf("Context file: %s\n", contextFile)
	}
	fmt.Printf("Initial context: %+v\n", engine.GetContext())

	// Evaluate the playbook
	results, err := engine.EvaluatePlaybook(playbookData)
	if err != nil {
		log.Printf("Error evaluating playbook: %v", err)
		return
	}

	fmt.Println("\n=== Playbook Results ===")
	for i, result := range results {
		fmt.Printf("Rule %d result: %+v\n", i+1, result)
	}

	fmt.Println("\n=== Final Context ===")
	finalContext := engine.GetContext()
	contextJSON, _ := json.MarshalIndent(finalContext, "", "  ")
	fmt.Println(string(contextJSON))
}

// loadContextFromFile loads context from a JSON file
func loadContextFromFile(filename string) (map[string]interface{}, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read context file: %v", err)
	}

	// Parse JSON
	var context map[string]interface{}
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("failed to parse context JSON: %v", err)
	}

	return context, nil
}

// loadPlaybookFromFile loads a playbook from a JSON file
func loadPlaybookFromFile(filename string) ([]interface{}, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read playbook file: %v", err)
	}

	// Parse JSON
	var playbook []interface{}
	if err := json.Unmarshal(data, &playbook); err != nil {
		return nil, fmt.Errorf("failed to parse playbook JSON: %v", err)
	}

	return playbook, nil
}
