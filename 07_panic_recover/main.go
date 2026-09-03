package main

/*
TASK 7: Panic & Recover in Concurrency – Safe Goroutines, Crash Prevention, and Stack Capturing

Topic: Advanced Error Handling – Panic & Recover

Problem Description:
You are building a Resilient Plugin & Job Execution Engine in Go.
In Go's runtime, if ANY spawned goroutine panics (e.g., nil pointer dereference, index out of range,
or explicit panic) and does not recover within that specific goroutine, the ENTIRE application process
crashes immediately. A parent goroutine CANNOT catch a panic that occurred in a child goroutine.

Your goal is to build a panic-safe execution layer that isolates panics per goroutine, converts them
into standard Go errors with captured stack traces, and prevents service crashes.

Requirements:

1. Panic Error Type:
   - Define a `PanicError` struct:
     * Must store: `TaskID int`, `Value any` (the value recovered from `recover()`), and `StackTrace string`.
     * Implements the `Error() string` method formatting the panic message and task ID.

2. Panic-Safe Function Wrapper:
   - Implement `SafeExecute(taskID int, fn func() error) (err error)`:
     * Executes `fn()`.
     * Uses `defer` and `recover()` to catch any runtime or explicit panic.
     * If a panic occurs:
       - Captures the current stack trace using `debug.Stack()` from `runtime/debug`.
       - Wraps the panic value and stack trace into a `*PanicError` and assigns it to the named return `err`.
     * If `fn()` returns a normal error, returns that error.
     * If `fn()` succeeds, returns `nil`.

3. Concurrent Batch Runner with Panic Isolation:
   - Implement `RunBatchSafely(tasks []func() error) []error`:
     * Runs all tasks concurrently in separate goroutines (e.g., using `sync.WaitGroup`).
     * Wraps every task execution using `SafeExecute` so that NO worker goroutine can crash the process.
     * Collects all errors into a slice of `[]error` (preserving order per task index).
     * Waits for all goroutines to finish and returns the collected slice.

4. In `main()` function:
   - Run a batch containing 4 different tasks:
     1) Task 0: A healthy task that succeeds and returns `nil`.
     2) Task 1: A task that returns a normal error (e.g., `errors.New("db query failed")`).
     3) Task 2: A task that explicitly calls `panic("unexpected external api payload")`.
     4) Task 3: A task that causes a runtime panic (e.g., nil pointer dereference or division by zero).
   - In `main()`, iterate over the returned errors:
     * If an error is a `*PanicError` (check using `errors.As`), print a warning and its captured stack trace.
     * If it's a standard error, print it normally.
     * If `nil`, print success.
   - Verify that the program finishes cleanly with exit code 0 and passes `go run -race 07_panic_recover/main.go`.

Good luck! Implement your solution below.
*/

import (
	"errors"
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// 1

type PanicError struct {
	TaskID     int
	Value      any
	StackTrace string
}

func (p PanicError) Error() string {
	return fmt.Sprintf("Panic message, task %d", p.TaskID)
}

// 2
func SafeExecute(taskID int, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = PanicError{
				TaskID:     taskID,
				StackTrace: string(debug.Stack()),
				Value:      r,
			}
		}
	}()
	return fn()
}

// 3
func RunBatchSafely(tasks []func() error) []error {
	g := errgroup.Group{}
	g.SetLimit(10)
	errs := make([]error, len(tasks))

	for i := range tasks {
		g.Go(func() error {
			errs[i] = SafeExecute(i, tasks[i])
			return errs[i]
		})
	}

	g.Wait()

	return errs
}

/*
* 4. In `main()` function:
  - Run a batch containing 4 different tasks:
    1) Task 0: A healthy task that succeeds and returns `nil`.
    2) Task 1: A task that returns a normal error (e.g., `errors.New("db query failed")`).
    3) Task 2: A task that explicitly calls `panic("unexpected external api payload")`.
    4) Task 3: A task that causes a runtime panic (e.g., nil pointer dereference or division by zero).
  - In `main()`, iterate over the returned errors:
  - If an error is a `*PanicError` (check using `errors.As`), print a warning and its captured stack trace.
  - If it's a standard error, print it normally.
  - If `nil`, print success.
  - Verify that the program finishes cleanly with exit code 0 and passes `go run -race 07_panic_recover/main.go`.
*/
func main() {
	fmt.Println("Task 7: Panic & Recover in Concurrency")
	// 4
	success := func() error {
		return nil
	}
	normalErr := func() error {
		return errors.New("normal error")
	}
	explicitPanic := func() error {
		panic("AaaaAAA!!!")
	}
	runtimePanic := func() error {
		arr := []string{}
		fmt.Print(arr[2])
		return nil
	}

	// 5
	errs := RunBatchSafely([]func() error{
		success,
		normalErr,
		explicitPanic,
		runtimePanic,
	})

	for i := range errs {
		if errs[i] == nil {
			fmt.Printf("[%d] Success\n", i)
			continue
		}

		var panicErr PanicError
		if errors.As(errs[i], &panicErr) {
			fmt.Printf("[%d] Warning: %s\nValue: %w\nStack trace: \n------\n%s\n------\n", i, panicErr.Error(), panicErr.Value, panicErr.StackTrace)
			continue
		}

		fmt.Printf("[%d] Error: %s\n", i, errs[i].Error())
	}
}
