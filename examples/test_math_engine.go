package main

import (
	"encoding/json"
	"fmt"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/rules"
)

func main() {
	// Load config
	cfg := &config.Config{
		RulesEngine: config.RulesEngineConfig{
			Caching: config.RulesEngineCacheConfig{
				MaxContexts:           1000,
				MaxExpressions:        500,
				MaxVariables:          100,
				EnableLazyEval:        true,
				EnableExpressionCache: true,
				MaxFieldSize:          1024,
				ContextTTL:            "30m",
				ExpressionTTL:         "15m",
				VariableTTL:           "10m",
				CleanupInterval:       "5m",
			},
		},
	}

	// Create rules engine
	engine := rules.NewEngine(cfg)
	defer engine.Close()

	// Test basic math operations
	fmt.Println("Testing Math Operations...")

	// Test 1: Simple addition
	additionRule := map[string]interface{}{
		"math": map[string]interface{}{
			"operation": "add",
			"operands":  []interface{}{10, 20, 5},
		},
	}

	result, err := engine.EvaluateRule(additionRule)
	if err != nil {
		fmt.Printf("Addition test failed: %v\n", err)
		return
	}
	fmt.Printf("Addition (10 + 20 + 5): %v\n", result)

	// Test 2: Variable-based calculation
	engine.SetContext(map[string]interface{}{
		"score1": 85.0,
		"score2": 72.0,
		"score3": 68.0,
	})

	averageRule := map[string]interface{}{
		"math": map[string]interface{}{
			"operation": "average",
			"operands": []interface{}{
				map[string]interface{}{"var": "score1"},
				map[string]interface{}{"var": "score2"},
				map[string]interface{}{"var": "score3"},
			},
		},
	}

	result, err = engine.EvaluateRule(averageRule)
	if err != nil {
		fmt.Printf("Average test failed: %v\n", err)
		return
	}
	fmt.Printf("Average of scores: %v\n", result)

	// Test 3: Complex nested calculation
	complexRule := map[string]interface{}{
		"math": map[string]interface{}{
			"operation": "multiply",
			"operands": []interface{}{
				map[string]interface{}{
					"math": map[string]interface{}{
						"operation": "add",
						"operands": []interface{}{
							map[string]interface{}{"var": "score1"},
							map[string]interface{}{"var": "score2"},
						},
					},
				},
				0.6,
			},
		},
	}

	result, err = engine.EvaluateRule(complexRule)
	if err != nil {
		fmt.Printf("Complex calculation test failed: %v\n", err)
		return
	}
	fmt.Printf("Complex calculation ((score1 + score2) * 0.6): %v\n", result)

	// Test 4: Simple variable assignment and lookup
	playbook := []interface{}{
		map[string]interface{}{
			"math": map[string]interface{}{
				"operation": "add",
				"operands": []interface{}{
					map[string]interface{}{"var": "score1"},
					map[string]interface{}{"var": "score2"},
				},
			},
			"var": "sum_result",
		},
		map[string]interface{}{
			"var": "sum_result",
		},
	}

	results, err := engine.EvaluatePlaybook(playbook)
	if err != nil {
		fmt.Printf("Playbook test failed: %v\n", err)
		return
	}

	fmt.Println("\nPlaybook Results:")
	for i, result := range results {
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Printf("Rule %d: %s\n", i+1, resultJSON)
	}

	fmt.Println("\nFinal Context:")
	finalContext := engine.GetContext()
	for key, value := range finalContext {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Println("\nAll math operation tests completed successfully!")
}
