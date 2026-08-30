package main

/*
TASK 5: Group Error Handling & Concurrency Lifecycle with `errgroup.Group`

Topic: Concurrency – `golang.org/x/sync/errgroup`

Problem Description:
You are building a distributed Data Ingestion & Validation Service.
The service receives a batch of data items to validate and process.
Tasks must run concurrently, but you need strict error handling:
if ANY task fails, the entire batch must fail immediately (Fast-Fail),
cancelling all other in-flight tasks via context and returning the first error encountered.
Additionally, you must enforce a maximum concurrency limit across the tasks.

Requirements:

1. Data Structures:
   - `type Task struct { ID int; Data string; ShouldFail bool }`
   - `type TaskResult struct { TaskID int; ProcessedData string }`

2. Task Processing Logic:
   - Implement `processItem(ctx context.Context, task Task) (TaskResult, error)`:
     * Simulates processing latency (e.g. 50ms).
     * Listens to `ctx.Done()`: if cancelled before finishing, returns an empty `TaskResult` and `ctx.Err()`.
     * If `task.ShouldFail` is true, returns an error: `fmt.Errorf("task %d failed validation", task.ID)`.
     * Otherwise returns `TaskResult{TaskID: task.ID, ProcessedData: "PROCESSED: " + task.Data}` and `nil`.

3. Group Processor with `errgroup`:
   - Implement `ProcessBatch(ctx context.Context, tasks []Task, maxConcurrency int) ([]TaskResult, error)`:
     * Uses `errgroup.WithContext(ctx)` to create an `errgroup.Group` and a derived context.
     * Uses `g.SetLimit(maxConcurrency)` to ensure no more than `maxConcurrency` goroutines run concurrently.
     * Launches each task using `g.Go(func() error { ... })`.
     * Safely collects successful `TaskResult` items (e.g. using a mutex or pre-allocated slice with task index).
     * Waits for all tasks with `g.Wait()`.
     * If any task fails, returns `nil` for results and the error returned by `g.Wait()`.

4. In `main()` function:
   - Demonstrate two distinct scenarios:
     a) Scenario A (All Succeed): Batch of 6 tasks (all `ShouldFail: false`) with `maxConcurrency: 3`.
        All tasks succeed, results are printed, error is `nil`.
     b) Scenario B (Fast-Fail on Error): Batch of 6 tasks where Task #2 has `ShouldFail: true` with `maxConcurrency: 3`.
        The error is returned, remaining tasks are cancelled via context, and failure is reported cleanly.
   - Verify race-free execution with `go run -race 05_errgroup/main.go`.

Good luck! Implement your solution below.
*/

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

type Task struct {
	ID         int
	Data       string
	ShouldFail bool
}

type TaskResult struct {
	TaskID        int
	ProcessedData string
}

func processItem(ctx context.Context, task Task) (TaskResult, error) {
	select {
	case <-time.After(time.Millisecond * 30):
		if task.ShouldFail {
			return TaskResult{}, fmt.Errorf("task %d failed validation", task.ID)
		}
		return TaskResult{
			TaskID:        task.ID,
			ProcessedData: "PROCESSED: " + task.Data,
		}, nil
	case <-ctx.Done():
		return TaskResult{}, ctx.Err()
	}
}

func ProcessBatch(ctx context.Context, tasks []Task, maxConcurrency int) ([]TaskResult, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	results := make([]TaskResult, len(tasks))

	for i := range tasks {
		g.Go(func() error {
			// Fail fast if context is canceled
			if ctx.Err() != nil {
				return ctx.Err()
			}

			tr, err := processItem(ctx, tasks[i])
			results[i] = tr
			return err
		})
	}

	err := g.Wait()
	if err == nil {
		return results, nil
	}

	return nil, err
}

func main() {
	fmt.Println("Task 5: Group Error Handling with errgroup")
	ctx := context.Background()
	scenarioA(ctx)
	scenarioB(ctx)

}

func scenarioA(ctx context.Context) {
	fmt.Println("Scenario A")
	result, err := ProcessBatch(ctx, []Task{
		{ID: 1, Data: "Lorem"},
		{ID: 2, Data: "ipsum"},
		{ID: 3, Data: "dolor"},
		{ID: 4, Data: "sit"},
		{ID: 5, Data: "amet"},
		{ID: 6, Data: "consectetur"},
	}, 3)
	if err != nil {
		fmt.Printf("Process batch failed %v", err)
	}
	printResult(result)
}

func scenarioB(ctx context.Context) {
	fmt.Println("Scenario B")
	result, err := ProcessBatch(ctx, []Task{
		{ID: 1, Data: "Lorem"},
		{ID: 2, Data: "ipsum", ShouldFail: true},
		{ID: 3, Data: "dolor"},
		{ID: 4, Data: "sit"},
		{ID: 5, Data: "amet"},
		{ID: 6, Data: "consectetur"},
	}, 3)
	if err != nil {
		fmt.Printf("Process batch failed %v", err)
	}
	printResult(result)
}

func printResult(results []TaskResult) {
	for i := range results {
		fmt.Printf("[TaskID=%d] ProcessedData: %s\n", results[i].TaskID, results[i].ProcessedData)
	}
}
