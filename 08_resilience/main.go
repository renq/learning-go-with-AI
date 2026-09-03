package main

/*
TASK 8: Resilience Patterns – Circuit Breaker and Exponential Backoff Retry

Topic: Advanced Error Handling – Resilience Patterns

Problem Description:
You are building an Outbound Resilience Layer for interacting with unstable external services/APIs.
When downstream services become degraded or fail, hammering them with requests creates cascading failures.
You will implement two fundamental resilience patterns:
1. Exponential Backoff Retry with context cancellation support.
2. A Thread-Safe Circuit Breaker state machine (Closed, Open, Half-Open).

Requirements:

1. Exponential Backoff Retry:
   - Implement `RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, fn func() error) error`:
     * Attempts to execute `fn()` up to `maxAttempts` times.
     * If `fn()` returns `nil`, succeeds immediately and returns `nil`.
     * If `fn()` returns an error, doubles the delay before the next attempt (e.g., 50ms, 100ms, 200ms...).
     * Must respect `ctx.Done()` during sleep intervals: if cancelled or timed out, aborts immediately and returns `ctx.Err()`.
     * If all attempts fail, returns the last encountered error.

2. Circuit Breaker States & Sentinel Errors:
   - Define sentinel error `ErrCircuitOpen` (e.g., "circuit breaker is open: fast failing request").
   - Define state enum/types: `StateClosed`, `StateOpen`, `StateHalfOpen`.

3. Circuit Breaker Struct & State Machine:
   - Define `CircuitBreaker` struct (fields: mutex, state, failure count, failure threshold, reset timeout, last state change timestamp).
   - Implement constructor `NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker`.
   - Implement `(cb *CircuitBreaker) Execute(fn func() error) error`:
     * Thread-Safe: protects internal state transitions using `sync.Mutex`.
     * State behavior:
       - `StateClosed`: Allows call `fn()`. If `fn()` succeeds, resets failure count. If `fn()` fails, increments failure count; if count >= `failureThreshold`, transitions to `StateOpen` and records timestamp.
       - `StateOpen`: If `resetTimeout` has NOT elapsed since opening, fast-fails immediately returning `ErrCircuitOpen` without calling `fn()`. If `resetTimeout` HAS elapsed, transitions to `StateHalfOpen`.
       - `StateHalfOpen`: Allows a single trial call `fn()`. If `fn()` succeeds, transitions back to `StateClosed` and resets counters. If `fn()` fails, transitions back to `StateOpen` and resets timer.

4. In `main()` function:
   - Part 1 (Retry Demo):
     * Test `RetryWithBackoff` with a simulated flaky function that fails twice and succeeds on the 3rd attempt.
     * Test `RetryWithBackoff` with a context timeout that aborts retries early.
   - Part 2 (Circuit Breaker Demo):
     * Configure a breaker with threshold: 3 failures, resetTimeout: 200ms.
     * Trigger 3 failures -> verify state transitions to Open.
     * Attempt immediate execution -> verify it fast-fails with `ErrCircuitOpen`.
     * Sleep for 250ms -> execute a successful call -> verify breaker transitions through Half-Open back to Closed.
   - Run and verify with `go run -race 08_resilience/main.go`.

Good luck! Implement your solution below.
*/

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// TODO 1: Define ErrCircuitOpen and Circuit Breaker States (StateClosed, StateOpen, StateHalfOpen)

var ErrCircuitOpen = errors.New("circuit breaker is open: fast failing request")

type State int

const (
	StateClosed State = iota + 1
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "StateHalfOpen"
	}
	return ""
}

// TODO 2: Define CircuitBreaker struct and constructor NewCircuitBreaker

type CircuitBreaker struct {
	mu               sync.Mutex
	state            State
	failureCount     int
	failureThreshold int
	resetTimeout     time.Duration
	lastStateChange  time.Time
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            StateClosed,
	}
}

// TODO 3: Implement (cb *CircuitBreaker) Execute(fn func() error) error

func (cb *CircuitBreaker) Execute(fn func() error) error {
	var err error

	cb.mu.Lock()
	st := cb.state
	cb.mu.Unlock()

	switch st {
	case StateClosed:
		err = fn()
		cb.mu.Lock()
		if err == nil {
			cb.failureCount = 0
		} else {
			cb.failureCount++
			if cb.failureCount >= cb.failureThreshold {
				cb.state = StateOpen
				cb.lastStateChange = time.Now()
			}
		}
		cb.mu.Unlock()
	case StateOpen:
		cb.mu.Lock()
		if time.Now().Before(cb.lastStateChange.Add(cb.resetTimeout)) {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		cb.state = StateHalfOpen
		cb.mu.Unlock()
		fallthrough
	case StateHalfOpen:
		err = fn()
		cb.mu.Lock()
		if err == nil {
			cb.state = StateClosed
			cb.failureCount = 0
		} else {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
		cb.mu.Unlock()
	}
	return err
}

// TODO 4: Implement RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, fn func() error) error

func RetryWithBackoff(ctx context.Context, maxAttempts int, initialDelay time.Duration, fn func() error) error {
	delay := initialDelay
	var err error
	for range maxAttempts {
		err = fn()
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrCircuitOpen) {
			fmt.Println("  [RETRY] Circuit is OPEN! Aborting all remaining retries.")
		}

		select {
		case <-time.After(delay):
			fmt.Printf("  [DEBUG] delay = %d\n", delay)
			delay *= 2
		case <-ctx.Done():
			fmt.Printf("  [DEBUG] %s\n", ctx.Err())
			return ctx.Err()
		}
	}

	return err
}

func main() {
	fmt.Println("Task 8: Resilience Patterns (Circuit Breaker & Exponential Backoff)")

	// TODO 5: Test Part 1 (Exponential Backoff with success and timeout)
	ctx := context.Background()

	flakyCounter := 0
	flakyFunction := func() error {
		flakyCounter++
		if flakyCounter%3 == 0 {
			return nil
		}
		return errors.New("Flaky function failure")
	}
	errFunc := func() error {
		fmt.Printf("  [DEBUG] Execute err func\n")
		return errors.New("error")
	}
	successFunc := func() error {
		fmt.Printf("  [DEBUG] Execute success func\n")
		return nil
	}

	func() {
		fmt.Println("Part 1 (Retry Demo)")
		fmt.Println("Flaky function works on the 3rd attempt")

		err := RetryWithBackoff(ctx, 3, time.Millisecond*100, flakyFunction)
		if err == nil {
			fmt.Println("- Flaky function success\n")
		} else {
			fmt.Printf("- Flaky function error: %s\n", err.Error())
		}

		fmt.Println("Flaky function with a context timeout that aborts retries early")

		ctxT, cancel := context.WithTimeout(ctx, time.Millisecond*290) // less than 100ms + 200ms = 300ms
		defer cancel()

		err = RetryWithBackoff(ctxT, 3, time.Millisecond*100, flakyFunction)
		if err == nil {
			fmt.Println("- Flaky function success\n")
		} else {
			fmt.Printf("- Flaky function error %s\n", err.Error())
		}
	}()

	func() {
		fmt.Println("Part 2 (Circuit Breaker Demo)")
		cb := NewCircuitBreaker(3, time.Millisecond*200)

		fmt.Println("Configure a breaker with threshold: 3 failures, resetTimeout: 200ms")
		fmt.Printf("- State%s\n", cb.state)
		fmt.Printf("  Result: %w\n", cb.Execute(errFunc))
		fmt.Printf("- State%s\n", cb.state)
		fmt.Printf("  Result: %w\n", cb.Execute(errFunc))
		fmt.Printf("- State%s\n", cb.state)
		fmt.Printf("  Result: %w\n", cb.Execute(errFunc))
		fmt.Printf("- State%s\n", cb.state)
		fmt.Printf("  Result: %w\n", cb.Execute(errFunc))
		fmt.Printf("- State%s\n", cb.state)
		fmt.Println("- Sleeping for 250ms")
		time.Sleep(time.Millisecond * 250)
		fmt.Printf("  Result: %w\n", cb.Execute(successFunc))
		fmt.Printf("- State%s\n", cb.state)
	}()

	func() {
		fmt.Println("\nExtra part (Circuit Breaker + backoff function)")

		cb := NewCircuitBreaker(3, time.Millisecond*500)
		err := RetryWithBackoff(ctx, 5, 50*time.Millisecond, func() error {
			return cb.Execute(errFunc)
		})

		if err == nil {
			fmt.Println("- Success\n")
		} else {
			fmt.Printf("- Error: %s\n", err.Error())
		}
		fmt.Printf("- State%s\n", cb.state)
	}()
}
