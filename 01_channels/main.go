package main

/*
TASK 1: Channels, select, buffering, and clean shutdown

Topic: Concurrency – Channels

Problem Description:
You are building an event stream processing module for system events (e.g. logs or metrics).
Your task is to implement a function that generates events and a function that processes
and aggregates them according to the following requirements:

Requirements:
1. Function `StartProducer(stopCh <-chan struct{}) <-chan int`:
   - Spawns a producer goroutine that periodically (e.g., every 50ms) generates sequential integers (1, 2, 3...).
   - The output channel should be buffered (e.g., capacity of 5).
   - The producer must listen to the `stopCh` channel and immediately terminate the goroutine upon receiving a signal (or on channel close), safely closing the output channel.

2. Function `FilterAndAggregate(inCh <-chan int, timeout time.Duration) ([]int, error)`:
   - Receives data from the `inCh` channel.
   - Collects only even numbers into the result slice `[]int`.
   - Has a time guard (`select` with `time.After(timeout)`): if no new data arrives from `inCh` within `timeout`, the function should stop collecting and return the items collected so far along with an error `fmt.Errorf("timeout waiting for data")`.
   - If `inCh` is closed properly by the producer (and drained of remaining items), the function should return the collected data and a `nil` error.

3. In `main()` function:
   - Demonstrate the producer and filter in two scenarios:
     a) Normal shutdown via signaling `stopCh`.
     b) Interruption due to timeout (e.g., when the producer is slower than the timeout).
   - Ensure the code does not leak goroutines and does not panic (e.g. send on closed channel).
*/

import (
	"fmt"
	"math/rand"
	"time"
)

func StartProducer(stopCh <-chan struct{}) <-chan int {
	producerValue := 0
	ch := make(chan int, 5)

	go func() {
		for {
			select {
			case <-stopCh:
				close(ch)
				return
			case <-time.After(time.Duration(rand.Int63n(300)) * time.Millisecond):
				producerValue++
				ch <- producerValue
			}
		}
	}()

	return ch
}

func FilterAndAggregate(inCh <-chan int, timeout time.Duration) ([]int, error) {
	result := []int{}
	for {
		select {
		case <-time.After(timeout):
			return result, fmt.Errorf("timeout waiting for data")
		case val, ok := <-inCh:
			if !ok {
				return result, nil
			}
			if val%2 == 0 {
				result = append(result, val)
			}
		}
	}
}

func scenarioA() {
	// Start producer; a timeout will trigger if the random duration exceeds timeout
	stopCh := make(chan struct{})
	defer close(stopCh)

	ch := StartProducer(stopCh)

	ints, err := FilterAndAggregate(ch, time.Millisecond*280)
	if err != nil {
		fmt.Printf("Consume error: %s\n", err)
	}

	fmt.Printf("Consumed values: %v\n", ints)
}

func scenarioB() {
	// Close channel after 200ms
	stopCh := make(chan struct{})

	ch := StartProducer(stopCh)

	select {
	case <-time.After(200 * time.Millisecond):
		close(stopCh)
	}

	ints, err := FilterAndAggregate(ch, time.Second)
	if err != nil {
		fmt.Printf("Consume error: %s\n", err)
	}

	fmt.Printf("Consumed values: %v\n", ints)
}

func main() {
	fmt.Println("Task 1: Channels in Go")

	fmt.Println("Scenario A")
	scenarioA()

	fmt.Println("Scenario B")
	scenarioB()
}
