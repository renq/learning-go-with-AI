package main

/*
TASK 2: Concurrent Synchronization – sync.RWMutex, sync.WaitGroup, sync.Once, and sync/atomic

Topic: Concurrency – `sync` and `sync/atomic` packages

Problem Description:
You are building a thread-safe in-memory cache for frequently queried data in a web service.
The system must handle heavy read traffic (multiple concurrent readers) as well as periodic writes,
while tracking query statistics with minimal performance overhead.

Requirements:

1. Structure `Cache`:
   - Stores data in a `map[string]string`.
   - Uses `sync.RWMutex` to protect the map against concurrent reads and writes
     (multiple readers can read concurrently, but writes require exclusive access).
   - Methods:
     * `Set(key, value string)`: safely writes or updates a value.
     * `Get(key string) (string, bool)`: safely reads a value; if the key exists,
       returns (value, true), otherwise ("", false).
     * `Delete(key string)`: deletes a key from the cache.

2. Statistics using `sync/atomic`:
   - The cache must count:
     * `totalRequests` (total number of Get calls)
     * `hits` (number of cache hits)
     * `misses` (number of cache misses)
   - Use atomic operations from the `sync/atomic` package for modifying and reading these counters
     (e.g., `atomic.Int64` or `atomic.AddInt64`/`atomic.LoadInt64`).
     This ensures statistics tracking does not block the main cache mutex!
   - Method `Stats() Stats` returns a struct with the copied, current values of the counters.

3. One-time initialization with `sync.Once`:
   - Method `InitExpensiveResource()` simulates heavy initialization of an external resource
     (e.g., database connection, loading a large dictionary from disk - fmt.Println and time.Sleep are sufficient).
   - Calling this method multiple times from different goroutines must execute the actual initialization
     code EXACTLY ONCE, using `sync.Once`.

4. In `main()` function:
   - Create a `Cache` instance.
   - Launch e.g. 50 concurrent goroutines (using `sync.WaitGroup` to wait for their completion):
     * Some goroutines write data (`Set`).
     * Most goroutines read data (`Get`) for existing and non-existing keys.
     * Each goroutine attempts to call `InitExpensiveResource()`.
   - After all goroutines complete (`wg.Wait()`), display the final statistics (`Stats()`).
   - Test the code for data races: `go run -race 02_sync/main.go`.
     The program must not report any data race conditions!
*/

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Stats struct {
	TotalRequests int64
	Hits          int64
	Misses        int64
}

type stats struct {
	totalRequests atomic.Int64
	hits          atomic.Int64
	misses        atomic.Int64
}

type Cache struct {
	data  map[string]string
	mu    sync.RWMutex
	so    sync.Once
	stats stats
}

func NewCache() *Cache {
	return &Cache{
		data:  make(map[string]string),
		mu:    sync.RWMutex{},
		stats: stats{},
	}
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	if ok {
		c.stats.hits.Add(1)
	} else {
		c.stats.misses.Add(1)
	}
	c.stats.totalRequests.Add(1)
	return val, ok
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *Cache) Stats() Stats {
	return Stats{
		Hits:          c.stats.hits.Load(),
		Misses:        c.stats.misses.Load(),
		TotalRequests: c.stats.totalRequests.Load(),
	}
}

func (c *Cache) InitExpensiveResource() {
	c.so.Do(func() {
		fmt.Println("Initializing expensive resource!")
		time.Sleep(time.Second / 2)
	})
}

func main() {
	fmt.Println("Task 2: sync and sync/atomic packages")
	var wg sync.WaitGroup

	c := NewCache()

	for i := 0; i < 50; i++ {
		wg.Go(func() {
			c.InitExpensiveResource()

			key := fmt.Sprintf("key-%d", rand.Intn(2))
			shouldSet := rand.Intn(5) == 1
			if shouldSet {
				value := fmt.Sprintf("random value is %d", rand.Intn(100))
				fmt.Printf("[%d] Set %s: '%s'\n", i, key, value)
				c.Set(key, value)
			} else {
				value, exists := c.Get(key)
				if exists {
					fmt.Printf("[%d] Hit get %s: '%s'\n", i, key, value)
				} else {
					fmt.Printf("[%d] Missed get %s\n", i, key)
				}
			}
		})
	}

	wg.Wait()

	fmt.Printf("Stats:\nHits: %d\nMisses: %d\nTotal requests: %d\n", c.Stats().Hits, c.Stats().Misses, c.Stats().TotalRequests)
}
