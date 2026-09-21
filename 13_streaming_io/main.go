package main

/*
TASK 13: Custom I/O Streams, Pipeline Decorators, and Graceful Shutdown

Topic: Networking & I/O – Streaming Pipelines, io.Reader/Writer Decorators, and OS Signals

Problem Description:
In Go, `io.Reader` and `io.Writer` are the golden standards for processing large or infinite streams of data
with minimal, constant O(1) memory footprint.
Instead of loading entire files into memory (`os.ReadFile` or `io.ReadAll`), production systems chain
stream decorators to measure progress, throttle bandwidth, transform sensitive data on the fly,
and handle graceful shutdown via OS signals.

Requirements:

1. Custom `io.Reader` Decorators:
   - `CountingReader`:
     * Wraps an underlying `io.Reader`.
     * Tracks the total number of bytes read using `sync/atomic` (thread-safe).
     * Implements `Read(p []byte) (n int, err error)`: delegates to the underlying reader and increments byte counter.
     * Implements `TotalBytes() int64`: returns total bytes read so far.
   - `ThrottledReader`:
     * Wraps an underlying `io.Reader`.
     * Enforces a maximum bandwidth limit (`bytesPerSec int64`).
     * Implements `Read(p []byte) (n int, err error)`: throttles reads (e.g., sleeps proportionally based on bytes read)
       so the throughput does not exceed `bytesPerSec`.

2. Custom `io.Writer` Transformer:
   - `MaskingWriter`:
     * Wraps an underlying `io.Writer`.
     * Transforms data on the fly: replaces any ASCII digit character ('0'-'9') with an asterisk ('*').
     * Implements `Write(p []byte) (n int, err error)`: masks digits and forwards modified bytes to the underlying writer.

3. Streaming Pipeline Function:
   - Implement `StreamProcess(ctx context.Context, src io.Reader, dst io.Writer, bytesPerSec int64) (int64, error)`:
     * Wraps `src` with `CountingReader` and `ThrottledReader`.
     * Wraps `dst` with `MaskingWriter`.
     * Streams data from source to destination using `io.Copy` (or chunked reads/writes).
     * Must support cancellation: if `ctx.Done()` is signalled, terminates streaming early and returns `ctx.Err()`.
     * Returns total bytes transferred and any error (excluding `io.EOF`).

4. Demonstration in `main()` with Graceful Shutdown:
   - Set up `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` for OS signal handling.
   - Scenario 1 (Pipeline Transformation & Measurement):
     * Input data containing sensitive digits: `"User: Alice, Card: 1234-5678-9012, PIN: 9876, Score: 42\n"`.
     * Stream through pipeline into a `bytes.Buffer`.
     * Verify all digits are masked (`*`), byte count is accurate, and time taken reflects the throttle rate.
   - Scenario 2 (Context Cancellation / Early Shutdown):
     * Stream a large / repeating data source with a tight timeout (`context.WithTimeout(ctx, 50*time.Millisecond)`).
     * Verify pipeline halts early and returns `context.DeadlineExceeded` without hanging or leaking goroutines.
   - Run and verify with `go run -race 13_streaming_io/main.go`.

Good luck! Implement your solution below.
*/

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// TODO 1: Implement CountingReader struct and Read / TotalBytes methods
type CountingReader struct {
	io.Reader
	total atomic.Int64
}

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	c.total.Add(int64(n))
	return n, err
}

func (c *CountingReader) TotalBytes() int64 {
	return c.total.Load()
}

// TODO 2: Implement ThrottledReader struct and Read method
type ThrottledReader struct {
	io.Reader
	bytesPerSec int64
}

func NewThrottledReader(src io.Reader, bytesPerSec int64) *ThrottledReader {
	return &ThrottledReader{
		Reader:      src,
		bytesPerSec: bytesPerSec,
	}
}

func (c *ThrottledReader) Read(p []byte) (int, error) {
	n, err := c.Reader.Read(p)
	time.Sleep(time.Duration(n) * time.Second / time.Duration(c.bytesPerSec))
	return n, err
}

// TODO 3: Implement MaskingWriter struct and Write method
type MaskingWriter struct {
	io.Writer
}

func (m *MaskingWriter) Write(p []byte) (int, error) {
	for i := range p {
		if p[i] >= '0' && p[i] <= '9' {
			p[i] = '*'
		}
	}
	return m.Writer.Write(p)
}

// TODO 4: Implement StreamProcess(ctx context.Context, src io.Reader, dst io.Writer, bytesPerSec int64) (int64, error)
/*
 * 3. Streaming Pipeline Function:
    - Implement `StreamProcess(ctx context.Context, src io.Reader, dst io.Writer, bytesPerSec int64) (int64, error)`:
      * Wraps `src` with `CountingReader` and `ThrottledReader`.
      * Wraps `dst` with `MaskingWriter`.
      * Streams data from source to destination using `io.Copy` (or chunked reads/writes).
      * Must support cancellation: if `ctx.Done()` is signalled, terminates streaming early and returns `ctx.Err()`.
      * Returns total bytes transferred and any error (excluding `io.EOF`).
*/
func StreamProcess(ctx context.Context, src io.Reader, dst io.Writer, bytesPerSec int64) (written int64, err error) {
	r := &CountingReader{Reader: NewThrottledReader(src, bytesPerSec)}
	w := &MaskingWriter{Writer: dst}
	buf := make([]byte, 16)

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
			///
			nr, er := r.Read(buf)
			if nr > 0 {
				nw, ew := w.Write(buf[0:nr])
				if nw < 0 || nr < nw {
					nw = 0
					if ew == nil {
						return 0, errors.New("Can't write")
					}
				}
				written += int64(nw)
				if ew != nil {
					err = ew
					break
				}
				if nr != nw {
					return 0, errors.New("Short write")
				}
			}
			if er != nil {
				if er == io.EOF {
					return written, nil
				}
				return 0, er
			}
		}
	}
}

func main() {
	fmt.Println("Task 13: Custom I/O Streams, Pipeline Decorators, and Graceful Shutdown")
	ctx := context.Background()
	signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	s := "User: Alice, Card: 1234-5678-9012, PIN: 9876, Score: 42\n"

	// TODO 5: Demonstrate Scenario 1 (Streaming, masking, counting, and throttling)
	func() {
		src := strings.NewReader(s)
		dst := &bytes.Buffer{}

		totalBytes, err := StreamProcess(ctx, src, dst, 40)

		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		fmt.Printf("Total bytes: %d\n", totalBytes)
		fmt.Printf("Buffer: %s\n", dst.String())
	}()

	// TODO 6: Demonstrate Scenario 2 (Context cancellation / early shutdown)
	func() {
		ctxC, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		src := strings.NewReader(s)
		dst := &bytes.Buffer{}

		totalBytes, err := StreamProcess(ctxC, src, dst, 30)

		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		fmt.Printf("Total bytes: %d\n", totalBytes)
		fmt.Printf("Buffer: %s\n", dst.String())
	}()
}
