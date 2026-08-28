package main

/*
TASK 4: Concurrency Patterns – Worker Pool, Fan-Out / Fan-In, and Rate Limiting

Topic: Concurrency Patterns

Problem Description:
You are building a high-throughput Background Job Processing Pipeline.
The pipeline must take a stream of tasks, distribute them across a fixed pool of concurrent workers (Fan-Out),
apply rate-limiting to avoid exceeding external API quotas, and aggregate all results into a single
unified stream (Fan-In) while supporting graceful cancellation via context.

Requirements:

1. Data Structures:
   - `type Job struct { ID int; Payload string }`
   - `type Result struct { JobID int; Output string; Err error }`

2. Worker Pool with Fan-Out / Fan-In:
   - Implement `StartWorkerPool(ctx context.Context, numWorkers int, jobs <-chan Job, rateLimit time.Duration) <-chan Result`:
     * Spawns `numWorkers` concurrent worker goroutines (Fan-Out).
     * Each worker listens for jobs from the `jobs` channel.
     * Rate Limiting: Ensure workers respect a rate limit (e.g., via `time.Ticker` or token bucket),
       ensuring jobs are not processed faster than the allowed rate (e.g. 1 job per `rateLimit`).
     * Workers must respect `ctx.Done()`: if the context is cancelled, workers should stop accepting new jobs and exit cleanly.
     * All results must be sent to a single shared `results` channel (Fan-In).
     * The `results` channel MUST be closed automatically once all workers have finished processing all jobs (Hint: use `sync.WaitGroup` in a separate closing goroutine).

3. Worker Processing Logic:
   - Implement `processJob(ctx context.Context, job Job) Result`:
     * Simulates processing time (e.g. 30ms).
     * If `ctx.Done()` triggers during processing, returns a Result with `Err: ctx.Err()`.
     * Otherwise returns a Result with transformed payload (e.g., uppercase or reversed string) and `Err: nil`.

4. In `main()` function:
   - Create a jobs channel and send 15 jobs into it, then close it.
   - Run the worker pool with 3 workers and a rate limit of e.g. 50ms.
   - Collect and print all results from the returned `results` channel using `for res := range results`.
   - Verify that:
     a) All 15 jobs are processed.
     b) Multiple workers processed the jobs concurrently.
     c) The program terminates cleanly without deadlocks and passes the race detector (`go run -race 04_patterns/main.go`).

Good luck! Implement your solution below.
*/

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Job struct {
	ID      int
	Payload string
}

type Result struct {
	JobID  int
	Output string
	Err    error
}

func processJob(ctx context.Context, job Job) Result {
	select {
	case <-ctx.Done():
		return Result{
			JobID: job.ID,
			Err:   ctx.Err(),
		}
	case <-time.After(30 * time.Millisecond):
		return Result{
			JobID:  job.ID,
			Output: strings.ToUpper(job.Payload),
		}
	}
}

func StartWorkerPool(ctx context.Context, numWorkers int, jobs <-chan Job, rateLimit time.Duration) <-chan Result {
	type token struct{}

	var wg sync.WaitGroup
	results := make(chan Result)
	bucket := make(chan token, numWorkers)
	stopRefill := make(chan struct{})

	for range numWorkers {
		wg.Go(func() {
			for {
				<-bucket

				select {
				case job, ok := <-jobs:
					if !ok {
						return
					}
					results <- processJob(ctx, job)
				case <-ctx.Done():
					return
				}
			}
		})
	}

	// refill tokens in the bucket
	go func() {
		// make the bucket full on start
		for range numWorkers {
			bucket <- token{}
		}
		tick := time.Tick(rateLimit)
		for {
			select {
			case <-stopRefill:
				return
			case <-tick:
				select {
				case bucket <- token{}:
				default: //bucket full, ignore
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Closing pattern, if workers are done, we close the channel
	go func() {
		wg.Wait()
		close(results)
		close(stopRefill)
	}()

	return results // this bi-directional channel is automatically converted on return to read only channel
}

func main() {
	fmt.Println("Task 4: Concurrency Patterns (Worker Pool, Fan-In/Fan-Out, Rate Limiting)")

	ctx := context.Background()
	jobs := make(chan Job)

	go func() {
		for i := range 15 {
			j := Job{
				ID:      i,
				Payload: fmt.Sprintf("Payload %d", i),
			}
			jobs <- j
		}
		close(jobs)
	}()

	for r := range StartWorkerPool(ctx, 3, jobs, 50*time.Millisecond) {
		if r.Err == nil {
			fmt.Printf("Worker result. JobID: %d, Output: %s\n", r.JobID, r.Output)
		} else {
			fmt.Printf("Worker failed. JobID: %d\n", r.JobID)
		}
	}
}
