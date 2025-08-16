package recovery

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"SoarAuto/pkg/errors"
	"SoarAuto/pkg/types"
)

// RecoveryHandler handles panic recovery and error recovery mechanisms
type RecoveryHandler struct {
	logger types.Logger
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler(logger types.Logger) *RecoveryHandler {
	return &RecoveryHandler{
		logger: logger,
	}
}

// RecoverPanic recovers from panics and converts them to SecAutoErrors
func (rh *RecoveryHandler) RecoverPanic(component string, operation string) *errors.SecAutoError {
	if r := recover(); r != nil {
		stackTrace := string(debug.Stack())
		
		var err *errors.SecAutoError
		
		// Try to convert panic to error
		switch v := r.(type) {
		case error:
			err = errors.SystemError(
				errors.ErrCodeSystemInternal,
				fmt.Sprintf("Panic recovered in %s", component),
				v,
			)
		case string:
			err = errors.SystemError(
				errors.ErrCodeSystemInternal,
				fmt.Sprintf("Panic recovered in %s: %s", component, v),
				nil,
			)
		default:
			err = errors.SystemError(
				errors.ErrCodeSystemInternal,
				fmt.Sprintf("Unknown panic recovered in %s", component),
				nil,
			)
		}
		
		err = err.WithComponent(component).
			WithOperation(operation).
			WithContext("stack_trace", stackTrace).
			WithContext("panic_value", fmt.Sprintf("%v", r))
		
		// Log the panic
		rh.logger.Error("Panic recovered", map[string]interface{}{
			"component":   component,
			"operation":   operation,
			"panic_value": r,
			"stack_trace": stackTrace,
		})
		
		return err
	}
	
	return nil
}

// SafeExecute executes a function with panic recovery
func (rh *RecoveryHandler) SafeExecute(component, operation string, fn func() error) error {
	defer func() {
		if err := rh.RecoverPanic(component, operation); err != nil {
			rh.logger.Error("Panic during safe execution", map[string]interface{}{
				"component": component,
				"operation": operation,
				"error":     err.Error(),
			})
		}
	}()
	
	return fn()
}

// SafeExecuteWithResult executes a function with panic recovery and returns result
func (rh *RecoveryHandler) SafeExecuteWithResult(component, operation string, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var err error
	
	defer func() {
		if panicErr := rh.RecoverPanic(component, operation); panicErr != nil {
			result = nil
			err = panicErr
		}
	}()
	
	result, err = fn()
	return result, err
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []errors.ErrorCode
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []errors.ErrorCode{
			errors.ErrCodeDatabaseTimeout,
			errors.ErrCodeDatabaseConnect,
			errors.ErrCodeNetworkTimeout,
			errors.ErrCodeNetworkConnection,
			errors.ErrCodeSystemTimeout,
			errors.ErrCodeSystemResource,
		},
	}
}

// RetryExecutor handles retry logic for operations
type RetryExecutor struct {
	config  *RetryConfig
	logger  types.Logger
	handler *RecoveryHandler
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(config *RetryConfig, logger types.Logger) *RetryExecutor {
	if config == nil {
		config = DefaultRetryConfig()
	}
	
	return &RetryExecutor{
		config:  config,
		logger:  logger,
		handler: NewRecoveryHandler(logger),
	}
}

// Execute executes a function with retry logic
func (re *RetryExecutor) Execute(ctx context.Context, component, operation string, fn func() error) error {
	var lastErr error
	delay := re.config.InitialDelay
	
	for attempt := 1; attempt <= re.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return errors.SystemError(
				errors.ErrCodeSystemTimeout,
				"Operation cancelled by context",
				ctx.Err(),
			).WithComponent(component).
				WithOperation(operation).
				WithContext("attempt", attempt)
		default:
		}
		
		// Execute with panic recovery
		err := re.handler.SafeExecute(component, operation, fn)
		
		if err == nil {
			// Success
			if attempt > 1 {
				re.logger.Info("Operation succeeded after retry", map[string]interface{}{
					"component": component,
					"operation": operation,
					"attempt":   attempt,
				})
			}
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !re.isRetryable(err) {
			re.logger.Debug("Error not retryable, stopping", map[string]interface{}{
				"component": component,
				"operation": operation,
				"attempt":   attempt,
				"error":     err.Error(),
			})
			break
		}
		
		// Don't sleep after the last attempt
		if attempt < re.config.MaxAttempts {
			re.logger.Debug("Retrying operation", map[string]interface{}{
				"component": component,
				"operation": operation,
				"attempt":   attempt,
				"delay":     delay,
				"error":     err.Error(),
			})
			
			// Sleep with context cancellation check
			select {
			case <-ctx.Done():
				return errors.SystemError(
					errors.ErrCodeSystemTimeout,
					"Operation cancelled during retry delay",
					ctx.Err(),
				).WithComponent(component).
					WithOperation(operation).
					WithContext("attempt", attempt)
			case <-time.After(delay):
			}
			
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * re.config.BackoffFactor)
			if delay > re.config.MaxDelay {
				delay = re.config.MaxDelay
			}
		}
	}
	
	// All attempts failed
	re.logger.Error("Operation failed after all retries", map[string]interface{}{
		"component":    component,
		"operation":    operation,
		"max_attempts": re.config.MaxAttempts,
		"final_error":  lastErr.Error(),
	})
	
	// Wrap the error with retry context
	if secAutoErr, ok := lastErr.(*errors.SecAutoError); ok {
		return secAutoErr.WithContext("retry_attempts", re.config.MaxAttempts).
			WithContext("retry_exhausted", true)
	}
	
	return errors.SystemError(
		errors.ErrCodeSystemInternal,
		"Operation failed after retries",
		lastErr,
	).WithComponent(component).
		WithOperation(operation).
		WithContext("retry_attempts", re.config.MaxAttempts).
		WithContext("retry_exhausted", true)
}

// ExecuteWithResult executes a function with retry logic and returns result
func (re *RetryExecutor) ExecuteWithResult(ctx context.Context, component, operation string, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var result interface{}
	delay := re.config.InitialDelay
	
	for attempt := 1; attempt <= re.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, errors.SystemError(
				errors.ErrCodeSystemTimeout,
				"Operation cancelled by context",
				ctx.Err(),
			).WithComponent(component).
				WithOperation(operation).
				WithContext("attempt", attempt)
		default:
		}
		
		// Execute with panic recovery
		result, err := re.handler.SafeExecuteWithResult(component, operation, fn)
		
		if err == nil {
			// Success
			if attempt > 1 {
				re.logger.Info("Operation succeeded after retry", map[string]interface{}{
					"component": component,
					"operation": operation,
					"attempt":   attempt,
				})
			}
			return result, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !re.isRetryable(err) {
			break
		}
		
		// Don't sleep after the last attempt
		if attempt < re.config.MaxAttempts {
			// Sleep with context cancellation check
			select {
			case <-ctx.Done():
				return nil, errors.SystemError(
					errors.ErrCodeSystemTimeout,
					"Operation cancelled during retry delay",
					ctx.Err(),
				).WithComponent(component).
					WithOperation(operation).
					WithContext("attempt", attempt)
			case <-time.After(delay):
			}
			
			// Calculate next delay
			delay = time.Duration(float64(delay) * re.config.BackoffFactor)
			if delay > re.config.MaxDelay {
				delay = re.config.MaxDelay
			}
		}
	}
	
	// All attempts failed
	if secAutoErr, ok := lastErr.(*errors.SecAutoError); ok {
		return nil, secAutoErr.WithContext("retry_attempts", re.config.MaxAttempts).
			WithContext("retry_exhausted", true)
	}
	
	return nil, errors.SystemError(
		errors.ErrCodeSystemInternal,
		"Operation failed after retries",
		lastErr,
	).WithComponent(component).
		WithOperation(operation).
		WithContext("retry_attempts", re.config.MaxAttempts).
		WithContext("retry_exhausted", true)
}

// isRetryable checks if an error is retryable
func (re *RetryExecutor) isRetryable(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if it's a SecAutoError with retryable flag
	if secAutoErr, ok := err.(*errors.SecAutoError); ok {
		if secAutoErr.IsRetryable() {
			return true
		}
		
		// Check if error code is in retryable list
		for _, code := range re.config.RetryableErrors {
			if secAutoErr.Code == code {
				return true
			}
		}
	}
	
	return false
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxFailures     int
	ResetTimeout    time.Duration
	HalfOpenMaxCalls int
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config       *CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	halfOpenCalls int
	logger       types.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger types.Logger) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			MaxFailures:      5,
			ResetTimeout:     30 * time.Second,
			HalfOpenMaxCalls: 3,
		}
	}
	
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
		logger: logger,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(component, operation string, fn func() error) error {
	// Check if circuit breaker allows execution
	if !cb.allowExecution() {
		return errors.SystemError(
			errors.ErrCodeSystemResource,
			"Circuit breaker is open",
			nil,
		).WithComponent(component).
			WithOperation(operation).
			WithContext("circuit_state", cb.state).
			WithContext("failures", cb.failures)
	}
	
	// Execute the function
	err := fn()
	
	// Record the result
	cb.recordResult(err == nil)
	
	return err
}

// allowExecution checks if the circuit breaker allows execution
func (cb *CircuitBreaker) allowExecution() bool {
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailTime) > cb.config.ResetTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.halfOpenCalls = 0
			cb.logger.Info("Circuit breaker transitioning to half-open", map[string]interface{}{
				"component": "circuit_breaker",
			})
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return cb.halfOpenCalls < cb.config.HalfOpenMaxCalls
	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(success bool) {
	switch cb.state {
	case CircuitBreakerClosed:
		if success {
			cb.failures = 0
		} else {
			cb.failures++
			cb.lastFailTime = time.Now()
			if cb.failures >= cb.config.MaxFailures {
				cb.state = CircuitBreakerOpen
				cb.logger.Warn("Circuit breaker opened", map[string]interface{}{
					"component": "circuit_breaker",
					"failures":  cb.failures,
				})
			}
		}
	case CircuitBreakerHalfOpen:
		cb.halfOpenCalls++
		if success {
			cb.state = CircuitBreakerClosed
			cb.failures = 0
			cb.logger.Info("Circuit breaker closed", map[string]interface{}{
				"component": "circuit_breaker",
			})
		} else {
			cb.state = CircuitBreakerOpen
			cb.failures++
			cb.lastFailTime = time.Now()
			cb.logger.Warn("Circuit breaker re-opened", map[string]interface{}{
				"component": "circuit_breaker",
				"failures":  cb.failures,
			})
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int {
	return cb.failures
}