package main

/*
TASK 10: Concurrent-Safe Generic LRU Cache & Data Structures

Topic: Generics & Type System – Generic Data Structures

Problem Description:
You are building a high-performance in-memory LRU (Least Recently Used) Cache library.
An LRU cache maintains a fixed capacity. When full and a new item is added, it evicts the least recently
accessed item.

By leveraging Go Generics `[K comparable, V any]`, your cache can store any comparable key (e.g., `string`,
`int`, custom ID types) and any value (`*User`, `[]byte`, structs) with compile-time type safety,
zero runtime type assertions, and thread-safe operations.

Requirements:

1. Data Structures:
   - Define generic `node[K comparable, V any]` struct (storing key, value, and pointers to prev/next nodes for a doubly-linked list).
   - Define generic `LRUCache[K comparable, V any]` struct (storing capacity, mutex, hash map of keys to nodes, and head/tail list pointers).
   - Implement constructor `NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V]`.

2. Core LRU Operations (Thread-Safe with Mutex):
   - Implement `(c *LRUCache[K, V]) Get(key K) (V, bool)`:
     * If key exists: moves the node to the front (most recently used) and returns `(value, true)`.
     * If key does not exist: returns `(zeroValue, false)`.
   - Implement `(c *LRUCache[K, V]) Put(key K, value V)`:
     * If key exists: updates the value and moves node to the front.
     * If key does not exist: adds new node to the front and records in the map.
     * If capacity is exceeded: evicts the oldest node from the tail and removes it from the map.
   - Implement `(c *LRUCache[K, V]) Delete(key K) bool`:
     * Removes key and node from cache, returns `true` if found and deleted, `false` otherwise.
   - Implement `(c *LRUCache[K, V]) Len() int`:
     * Returns the current number of stored elements.
   - Implement `(c *LRUCache[K, V]) Keys() []K`:
     * Returns a slice of all keys ordered from most recently used to least recently used.

3. In `main()` function:
   - Demonstrate:
     a) String keys with struct values: `*LRUCache[string, User]` with capacity 3.
     b) Eviction: Insert 3 items ("u1", "u2", "u3"), then insert "u4" -> verify "u1" is evicted.
     c) Access refresh: Access "u2" with `Get`, then insert "u5" -> verify "u3" is evicted instead of "u2" (since "u2" was refreshed!).
     d) Concurrent safety: Spawn 20 concurrent goroutines performing simultaneous `Put` and `Get` operations on an `*LRUCache[int, string]` to ensure zero race conditions.
   - Run and verify with `go run -race 10_generic_structures/main.go`.

Good luck! Implement your solution below.
*/

import (
	"fmt"
	"math/rand"
	"sync"
)

type node[K comparable, V any] struct {
	key   K
	value V
	prev  *node[K, V]
	next  *node[K, V]
}

type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	hashMap  map[K]*node[K, V]
	head     *node[K, V]
	tail     *node[K, V]
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		hashMap:  make(map[K]*node[K, V], capacity+1),
	}
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.toHead(key) {
		return c.head.value, true
	}

	var n V
	return n, false
}

func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.toHead(key) {
		c.head.value = value
	} else {
		newNode := new(node[K, V]{
			key:   key,
			value: value,
			next:  c.head,
		})

		if c.head != nil {
			c.head.prev = newNode
		} else {
			c.tail = newNode
		}
		c.head = newNode
		c.hashMap[key] = newNode
	}

	if c.capacity < len(c.hashMap) {
		newLast := c.tail.prev
		c.tail.prev = nil
		c.tail.next = nil
		delete(c.hashMap, c.tail.key)
		newLast.next = nil
		c.tail = newLast
	}
}

func (c *LRUCache[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	ok := c.toHead(key)
	if ok {
		n := c.hashMap[key]
		c.head = n.next
		n.next = nil
		delete(c.hashMap, key)
		return true
	}
	return false
}

func (c *LRUCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.hashMap)
}

func (c *LRUCache[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]K, 0, len(c.hashMap))
	n := c.head
	for n != nil {
		keys = append(keys, n.key)
		n = n.next
	}
	return keys
}

func (c *LRUCache[K, V]) toHead(key K) bool {
	node, ok := c.hashMap[key]
	if !ok {
		return false
	}
	if c.head == c.hashMap[key] {
		return true
	}

	prev, next := node.prev, node.next

	if prev != nil {
		prev.next = next
	}
	if next != nil {
		next.prev = prev
	} else {
		c.tail = prev
	}

	c.head.prev = node
	node.next = c.head
	node.prev = nil
	c.head = node
	return true
}

type User struct {
	Email string
}

func main() {
	fmt.Println("Task 10: Concurrent-Safe Generic LRU Cache")

	cache := NewLRUCache[string, User](3)

	fmt.Println("Adding 3 users")
	cache.Put("u1", User{Email: "u1@lipek.net"})
	cache.Put("u2", User{Email: "u2@lipek.net"})
	cache.Put("u3", User{Email: "u3@lipek.net"})
	fmt.Printf("Len: %d, Keys: %v\n", cache.Len(), cache.Keys())

	fmt.Println("Adding 4th user")
	cache.Put("u4", User{Email: "u4@lipek.net"})
	fmt.Printf("Len: %d, Keys: %v\n", cache.Len(), cache.Keys())

	fmt.Println("Get u2")
	v, ok := cache.Get("u2")
	fmt.Printf("User u2: %v, ok: %b\n", v, ok)
	fmt.Printf("Len: %d, Keys: %v\n", cache.Len(), cache.Keys())

	fmt.Println("Insert u5")
	cache.Put("u5", User{Email: "u5@lipek.net"})
	fmt.Printf("Len: %d, Keys: %v\n", cache.Len(), cache.Keys())

	// race conditions check
	g := sync.WaitGroup{}
	for range 20 {
		g.Go(func() {
			userID1 := fmt.Sprintf("%d", rand.Intn(5)+1)
			userID2 := fmt.Sprintf("%d", rand.Intn(5)+1)
			cache.Put(userID1, User{Email: userID1 + "@lipek.net"})
			_, _ = cache.Get(userID2)
		})
	}
	g.Wait()
}
