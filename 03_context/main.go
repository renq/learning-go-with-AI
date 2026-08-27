package main

/*
TASK 3: Context – Cancellation, Timeouts, Values, and Leak Prevention

Topic: Concurrency – `context.Context`

Problem Description:
You are building an API Request Coordinator for a microservices architecture.
When an incoming request is received, it must propagate request metadata (e.g., RequestID),
enforce a strict SLA/Timeout, query multiple downstream microservices concurrently,
and immediately cancel all pending operations if the timeout is exceeded or if an error occurs.

Requirements:

1. Context Values with Safe Custom Keys:
   - Define custom unexported key types (e.g., `type ctxKey string`) to avoid key collisions.
   - Implement helper functions:
     * `WithRequestID(ctx context.Context, requestID string) context.Context`
     * `GetRequestID(ctx context.Context) (string, bool)`

2. Downstream Service Worker Simulation:
   - Implement `fetchService(ctx context.Context, serviceName string, delay time.Duration) (string, error)`:
     * Simulates network latency using `delay`.
     * Must monitor `ctx.Done()`. If the context is cancelled or times out while waiting,
       it must immediately abort and return `ctx.Err()`.
     * If completed within time, returns a formatted result (e.g., "data from " + serviceName).

3. Concurrent Multi-Source Aggregator:
   - Implement `FetchAllServices(ctx context.Context, services map[string]time.Duration) ([]string, error)`:
     * Queries all services concurrently in separate goroutines.
     * Propagates the parent `ctx` to all child goroutines.
     * Uses `sync.WaitGroup` (or channels) to wait for workers.
     * If the context expires (timeout/cancellation), returns the error and ensures NO goroutines are leaked in the background.

4. In `main()` function:
   - Demonstrate two distinct scenarios:
     a) Scenario A (Success): Query 3 services with delays (50ms, 80ms, 100ms) with a 200ms timeout -> all succeed.
     b) Scenario B (Timeout & Fast Abort): Query 3 services where one has a 500ms delay with a 150ms timeout -> times out, long tasks abort immediately.
   - Ensure proper cleanup with `defer cancel()` to prevent context/timer leaks.

Good luck! Implement your solution below.
*/

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ctxKey struct{}

var requestIDKey = ctxKey{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(requestIDKey).(string)
	return value, ok
}

func fetchService(ctx context.Context, serviceName string, delay time.Duration) (string, error) {
	reqID, _ := GetRequestID(ctx)
	fmt.Printf("[log] Fetching service %s, request %s\n", serviceName, reqID)

	select {
	case <-time.After(delay):
		return "data from " + serviceName, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func FetchAllServices(ctx context.Context, services map[string]time.Duration) ([]string, error) {
	var wg sync.WaitGroup

	results := make([]string, len(services))
	var rErr error
	mu := sync.Mutex{}
	i := 0

	for service, timeout := range services {
		wg.Go(func() {
			result, err := fetchService(ctx, service, timeout)

			mu.Lock()
			if err != nil {
				rErr = err
			}
			results[i] = result
			i++
			mu.Unlock()
		})
	}

	wg.Wait()
	if rErr != nil {
		return nil, rErr
	}

	return results, rErr
}

func main() {
	fmt.Println("Task 3: context.Context in Go")
	ctx := context.Background()

	fmt.Println("Scenario A")
	scenarioA(ctx)

	fmt.Println("\nScenario B")
	scenarioB(ctx)
}

func scenarioA(ctx context.Context) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	ctx = WithRequestID(ctx, "Scenario A")

	services := map[string]time.Duration{
		"srvA": time.Millisecond * 50,
		"srvB": time.Millisecond * 80,
		"srvC": time.Millisecond * 100,
	}

	results, err := FetchAllServices(ctx, services)
	if err != nil {
		fmt.Printf("Error returned: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", results)
}

func scenarioB(ctx context.Context) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	ctx = WithRequestID(ctx, "Scenario B")

	services := map[string]time.Duration{
		"srvA": time.Millisecond * 480,
		"srvB": time.Millisecond * 480,
		"srvC": time.Millisecond * 480,
	}

	results, err := FetchAllServices(ctx, services)
	if err != nil {
		fmt.Printf("Error returned: %v\n", err)
		return
	}

	fmt.Printf("Result: %v\n", results)
}
