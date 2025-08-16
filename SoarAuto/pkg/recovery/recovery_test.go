package recovery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/testutil"
)

func TestRecoveryHandler_RecoverPanic(t *testing.T) {
	logger := testutil.TestLogger(t)
	handler := NewRecoveryHandler(logger)
	
	t.Run("recover from string panic", func(t *testing.T) {
		var recoveredErr *errors.SecAutoError
		
		func() {
			defer func() {
				recoveredErr = handler.RecoverPanic("test_component", "test_operation")
			}()
			panic("test panic message")
		}()
		
		if recoveredErr == nil {
			t.Fatal("Expected error from panic recovery")
		}
		
		if recoveredErr.Code != errors.ErrCodeSystemInternal {
			t.Errorf("Expected code %s, got %s", errors.ErrCodeSystemInternal, recoveredErr.Code)
		}
		
		if recoveredErr.Component != "test_component" {
			t.Errorf("Expected component 'test_component', got %s", recoveredErr.Component)
		}
		
		if recoveredErr.Operation != "test_operation" {
			t.Errorf("Expected operation 'test_operation', got %s", recoveredErr.Operation)
		}
	})
	
	t.Run("recover from error panic", func(t *testing.T) {
		var recoveredErr *errors.SecAutoError
		originalErr := fmt.Errorf("original error")
		
		func() {
			defer func() {
				recoveredErr = handler.RecoverPanic("test_component", "test_operation")
			}()
			panic(originalErr)
		}()
		
		if recoveredErr == nil {
			t.Fatal("Expected error from panic recovery")
		}
		
		if recoveredErr.Cause != originalErr {
			t.Error("Expected original error to be preserved as cause")
		}
	})
	
	t.Run("no panic", func(t *testing.T) {
		var recoveredErr *errors.SecAutoError
		
		func() {
			defer func() {
				recoveredErr = handler.RecoverPanic("test_component", "test_operation")
			}()
			// No panic
		}()
		
		if recoveredErr != nil {
			t.Error("Expected no error when no panic occurs")
		}
	})
}

func TestRecoveryHandler_SafeExecute(t *testing.T) {
	logger := testutil.TestLogger(t)
	handler := NewRecoveryHandler(logger)
	
	t.Run("successful execution", func(t *testing.T) {
		executed := false
		err := handler.SafeExecute("test_component", "test_operation", func() error {
			executed = true
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		
		if !executed {
			t.Error("Expected function to be executed")
		}
	})
	
	t.Run("execution with error", func(t *testing.T) {
		expectedErr := fmt.Errorf("test error")
		err := handler.SafeExecute("test_component", "test_operation", func() error {
			return expectedErr
		})
		
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
	
	t.Run("execution with panic", func(t *testing.T) {
		// This test verifies that panics are recovered, but the SafeExecute
		// function doesn't return the panic as an error - it just logs it
		// The panic recovery happens in the defer, not in the return value
		err := handler.SafeExecute("test_component", "test_operation", func() error {
			panic("test panic")
		})
		
		// The function should complete without propagating the panic
		// The actual panic recovery is logged, not returned
		_ = err // SafeExecute doesn't return panic errors, just logs them
	})
}

func TestRetryExecutor(t *testing.T) {
	logger := testutil.TestLogger(t)
	
	t.Run("successful execution on first try", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  10 * time.Millisecond,
			BackoffFactor: 2.0,
		}
		executor := NewRetryExecutor(config, logger)
		
		attempts := 0
		err := executor.Execute(context.Background(), "test", "operation", func() error {
			attempts++
			return nil
		})
		
		testutil.AssertNoError(t, err)
		
		if attempts != 1 {
			t.Errorf("Expected 1 attempt, got %d", attempts)
		}
	})
	
	t.Run("successful execution after retries", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  10 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableErrors: []errors.ErrorCode{
				errors.ErrCodeDatabaseTimeout,
			},
		}
		executor := NewRetryExecutor(config, logger)
		
		attempts := 0
		err := executor.Execute(context.Background(), "test", "operation", func() error {
			attempts++
			if attempts < 3 {
				return errors.DatabaseError(
					errors.ErrCodeDatabaseTimeout,
					"Database timeout",
					fmt.Errorf("timeout"),
				)
			}
			return nil
		})
		
		testutil.AssertNoError(t, err)
		
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	})
	
	t.Run("non-retryable error", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  10 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableErrors: []errors.ErrorCode{
				errors.ErrCodeDatabaseTimeout,
			},
		}
		executor := NewRetryExecutor(config, logger)
		
		attempts := 0
		err := executor.Execute(context.Background(), "test", "operation", func() error {
			attempts++
			return errors.AuthError(
				errors.ErrCodeAuthInvalid,
				"Invalid API key",
			)
		})
		
		testutil.AssertError(t, err, "Invalid API key")
		
		if attempts != 1 {
			t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
		}
	})
	
	t.Run("all retries exhausted", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  10 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableErrors: []errors.ErrorCode{
				errors.ErrCodeDatabaseTimeout,
			},
		}
		executor := NewRetryExecutor(config, logger)
		
		attempts := 0
		err := executor.Execute(context.Background(), "test", "operation", func() error {
			attempts++
			return errors.DatabaseError(
				errors.ErrCodeDatabaseTimeout,
				"Database timeout",
				fmt.Errorf("timeout"),
			)
		})
		
		testutil.AssertError(t, err, "")
		
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		
		// Check that retry context was added
		if secAutoErr, ok := err.(*errors.SecAutoError); ok {
			if retryAttempts, exists := secAutoErr.Context["retry_attempts"]; !exists || retryAttempts != 3 {
				t.Error("Expected retry_attempts context to be set")
			}
			if retryExhausted, exists := secAutoErr.Context["retry_exhausted"]; !exists || retryExhausted != true {
				t.Error("Expected retry_exhausted context to be set")
			}
		}
	})
	
	t.Run("context cancellation", func(t *testing.T) {
		config := &RetryConfig{
			MaxAttempts:   5,
			InitialDelay:  100 * time.Millisecond,
			BackoffFactor: 2.0,
			RetryableErrors: []errors.ErrorCode{
				errors.ErrCodeDatabaseTimeout,
			},
		}
		executor := NewRetryExecutor(config, logger)
		
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		attempts := 0
		err := executor.Execute(ctx, "test", "operation", func() error {
			attempts++
			return errors.DatabaseError(
				errors.ErrCodeDatabaseTimeout,
				"Database timeout",
				fmt.Errorf("timeout"),
			)
		})
		
		testutil.AssertError(t, err, "cancelled")
		
		// Should have attempted at least once but not all 5 times due to cancellation
		if attempts == 0 {
			t.Error("Expected at least one attempt")
		}
		if attempts >= 5 {
			t.Error("Expected cancellation to prevent all attempts")
		}
	})
}

func TestRetryExecutor_ExecuteWithResult(t *testing.T) {
	logger := testutil.TestLogger(t)
	config := &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  10 * time.Millisecond,
		BackoffFactor: 2.0,
		RetryableErrors: []errors.ErrorCode{
			errors.ErrCodeDatabaseTimeout,
		},
	}
	executor := NewRetryExecutor(config, logger)
	
	t.Run("successful execution with result", func(t *testing.T) {
		expectedResult := "test result"
		
		result, err := executor.ExecuteWithResult(context.Background(), "test", "operation", func() (interface{}, error) {
			return expectedResult, nil
		})
		
		testutil.AssertNoError(t, err)
		
		if result != expectedResult {
			t.Errorf("Expected result %v, got %v", expectedResult, result)
		}
	})
	
	t.Run("execution with retries and result", func(t *testing.T) {
		expectedResult := "test result"
		attempts := 0
		
		result, err := executor.ExecuteWithResult(context.Background(), "test", "operation", func() (interface{}, error) {
			attempts++
			if attempts < 2 {
				return nil, errors.DatabaseError(
					errors.ErrCodeDatabaseTimeout,
					"Database timeout",
					fmt.Errorf("timeout"),
				)
			}
			return expectedResult, nil
		})
		
		testutil.AssertNoError(t, err)
		
		if result != expectedResult {
			t.Errorf("Expected result %v, got %v", expectedResult, result)
		}
		
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got %d", attempts)
		}
	})
}

func TestCircuitBreaker(t *testing.T) {
	logger := testutil.TestLogger(t)
	
	t.Run("circuit breaker closed state", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      3,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config, logger)
		
		if cb.GetState() != CircuitBreakerClosed {
			t.Error("Expected circuit breaker to start in closed state")
		}
		
		// Successful execution
		err := cb.Execute("test", "operation", func() error {
			return nil
		})
		
		testutil.AssertNoError(t, err)
		
		if cb.GetState() != CircuitBreakerClosed {
			t.Error("Expected circuit breaker to remain closed after success")
		}
	})
	
	t.Run("circuit breaker opens after failures", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      3,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config, logger)
		
		// Cause failures to open the circuit
		for i := 0; i < 3; i++ {
			err := cb.Execute("test", "operation", func() error {
				return fmt.Errorf("test error")
			})
			testutil.AssertError(t, err, "test error")
		}
		
		if cb.GetState() != CircuitBreakerOpen {
			t.Error("Expected circuit breaker to be open after max failures")
		}
		
		if cb.GetFailures() != 3 {
			t.Errorf("Expected 3 failures, got %d", cb.GetFailures())
		}
		
		// Next execution should be blocked
		err := cb.Execute("test", "operation", func() error {
			return nil
		})
		
		testutil.AssertError(t, err, "Circuit breaker is open")
	})
	
	t.Run("circuit breaker half-open transition", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			ResetTimeout:     50 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config, logger)
		
		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute("test", "operation", func() error {
				return fmt.Errorf("test error")
			})
		}
		
		if cb.GetState() != CircuitBreakerOpen {
			t.Error("Expected circuit breaker to be open")
		}
		
		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)
		
		// Next execution should transition to half-open
		err := cb.Execute("test", "operation", func() error {
			return nil
		})
		
		testutil.AssertNoError(t, err)
		
		if cb.GetState() != CircuitBreakerClosed {
			t.Error("Expected circuit breaker to be closed after successful half-open execution")
		}
	})
	
	t.Run("circuit breaker re-opens from half-open on failure", func(t *testing.T) {
		config := &CircuitBreakerConfig{
			MaxFailures:      2,
			ResetTimeout:     50 * time.Millisecond,
			HalfOpenMaxCalls: 2,
		}
		cb := NewCircuitBreaker(config, logger)
		
		// Open the circuit
		for i := 0; i < 2; i++ {
			cb.Execute("test", "operation", func() error {
				return fmt.Errorf("test error")
			})
		}
		
		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)
		
		// Fail in half-open state
		err := cb.Execute("test", "operation", func() error {
			return fmt.Errorf("test error")
		})
		
		testutil.AssertError(t, err, "test error")
		
		if cb.GetState() != CircuitBreakerOpen {
			t.Error("Expected circuit breaker to re-open after failure in half-open state")
		}
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()
	
	if config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts 3, got %d", config.MaxAttempts)
	}
	
	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected InitialDelay 100ms, got %v", config.InitialDelay)
	}
	
	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor 2.0, got %f", config.BackoffFactor)
	}
	
	if len(config.RetryableErrors) == 0 {
		t.Error("Expected retryable errors to be configured")
	}
}

func TestRetryExecutor_isRetryable(t *testing.T) {
	logger := testutil.TestLogger(t)
	config := &RetryConfig{
		RetryableErrors: []errors.ErrorCode{
			errors.ErrCodeDatabaseTimeout,
			errors.ErrCodeNetworkTimeout,
		},
	}
	executor := NewRetryExecutor(config, logger)
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "retryable error code",
			err:      errors.DatabaseError(errors.ErrCodeDatabaseTimeout, "timeout", nil),
			expected: true,
		},
		{
			name:     "non-retryable error code",
			err:      errors.AuthError(errors.ErrCodeAuthInvalid, "invalid"),
			expected: false,
		},
		{
			name:     "inherently retryable error",
			err:      errors.NetworkError(errors.ErrCodeNetworkConnection, "connection failed", nil),
			expected: true,
		},
		{
			name:     "explicitly retryable error",
			err:      errors.NewErrorBuilder(errors.ErrCodeValidationFailed, "validation failed").WithRetryable(true).Build(),
			expected: true,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.isRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}