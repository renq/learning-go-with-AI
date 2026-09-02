# 🚀 Advanced Go (Golang) Interactive Learning & Mentorship

A hands-on, practical repository designed to master intermediate and advanced Go concepts through interactive coding exercises, concurrency patterns, performance optimizations, and race-free designs.

> 💡 **Pair-programming with AI Mentor:** This repository is pre-configured with **Google Antigravity** rules and skills (`.agents/skills/go-mentor` & `GEMINI.md`) to provide an interactive, Socratic pair-programming tutor directly in your editor.

---

## 📚 Curriculum Roadmap

### 1. Concurrency Core & Synchronization Patterns
- [x] **01. Channels & Select** (`01_channels/`): Buffered channels, `select`, non-blocking I/O, timeouts, and clean shutdowns.
- [x] **02. Sync & Atomic Primitives** (`02_sync/`): `sync.RWMutex`, `sync.WaitGroup`, `sync.Once`, and high-throughput lock-free metrics using `sync/atomic`.
- [x] **03. Context & Lifecycle Management** (`03_context/`): `context.Context`, cancellation propagation, timeouts/deadlines, value propagation, and preventing context leaks.
- [x] **04. Concurrency Patterns**: Worker Pools, Fan-in / Fan-out, Rate Limiting (Token Bucket / Leaky Bucket), and Semaphores.
- [x] **05. Group Error Handling**: `golang.org/x/sync/errgroup` lifecycle & error coordination.

### 2. Advanced Error Handling & Resilience
- [x] **06. Error Trees & Custom Types**: `Unwrap()`, `errors.Is()`, `errors.As()`, and structured domain errors.
- [ ] **07. Concurrency Panic & Recovery**: Safe panic handling and recovery across worker goroutines.
- [ ] **08. Resilience Patterns**: Circuit Breaker, Exponential Backoff + Retry mechanisms.

### 3. Generics & Type System
- [ ] **09. Generic Functions & Constraints**: Custom constraints, `comparable`, `any`, and type parameter inference.
- [ ] **10. Advanced Generics & Data Structures**: Type approximation (`~T`), union sets, and concurrent-safe generic collections (e.g. Generic LRU Cache / Queue).

### 4. Advanced I/O & Streaming
- [ ] **11. Custom Readers & Writers**: Custom `io.Reader` / `io.Writer` wrapping, buffering, rate-limited streams, and streaming data parsers.
- [ ] **12. Graceful Shutdown & Network Architecture**: Clean shutdown of HTTP/gRPC servers, connection pooling, and OS signal trapping (`os.Signal`).

### 5. Memory, Performance & Profiling
- [ ] **13. Memory Re-use & `sync.Pool`**: Mitigating Garbage Collector (GC) pressure with object reuse and slice preallocations.
- [ ] **14. Escape Analysis & Memory Alignment**: Stack vs. heap allocations, struct field alignment, and zero-allocation patterns.
- [ ] **15. Benchmarking & Profiling**: `testing.B`, `pprof` (CPU, heap, goroutine profiles), and `go tool trace`.

### 6. Metaprogramming & Low-Level Mechanics
- [ ] **16. Reflection (`reflect`)**: Dynamic type inspection, struct tags, and generic serializers.
- [ ] **17. Low-Level Memory (`unsafe`)**: `unsafe.Pointer`, zero-copy string-byte conversions, and memory offsets.

---

## 🛠️ How It Works

Each exercise resides in its own package directory (e.g. `01_channels`, `02_sync`, `03_context`):
1. **Task Description**: Outlined in the header comment of `main.go`.
2. **Skeleton Code**: Pre-structured types and signatures with `// TODO` tags.
3. **Execution & Race Detection**: Run and verify all solutions using the Go race detector:
   ```bash
   go run -race ./01_channels/main.go
   go run -race ./02_sync/main.go
   go run -race ./03_context/main.go
   ```

---

## 🤖 Using with Google Antigravity

This repository includes custom agent configurations:
- **`GEMINI.md`**: Project-level rules guiding the AI to act as a Socratic Go tutor (no copy-paste spoilers, hint-based teaching, thorough code reviews).
- **`.agents/skills/go-mentor/`**: Reusable Antigravity skill that can be exported or cloned across any Go workspace.

---

## 📝 License
MIT License - feel free to fork, learn, and share!
