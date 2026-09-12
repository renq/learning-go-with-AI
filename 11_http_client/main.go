package main

/*
TASK 11: HTTP Client & RoundTripper Middleware

Topic: Networking – HTTP Client, Middleware, and Context

Problem Description:
This exercise is about learning how Go's `http.Client`, `http.RoundTripper`,
`context.Context`, and atomic counters work together. You will build a small
client around a test server and compose cross-cutting behavior as
`http.RoundTripper` middleware.

Learning Goals:
   - Reuse one `http.Client` and one underlying `http.Transport`.
   - Compose and delegate through `http.RoundTripper` decorators.
   - Clone a request before middleware changes its headers.
   - Propagate a caller's context to an outgoing request.
   - Distinguish an HTTP status response from a transport error.
   - Collect concurrent metrics safely and verify them with the race detector.

Scope:
   - This is a learning exercise, not a complete production API-client design.
   - Passing a full URL to the request method is acceptable. The client does not
     need to retain, validate, or resolve a configured base URL.
   - Because the request method returns `[]byte`, reading the complete response
     body is acceptable for this exercise. Streaming and body-size limits are
     useful follow-up topics, not requirements here.
   - Fine-grained transport tuning and exhaustive configuration validation are
     optional exploration, not acceptance criteria.

Requirements:

1. Client Configuration:
   - Define a `ClientConfig` type containing only the values you will use, such
     as a bearer token and request timeout.
   - Define an `APIClient` type that stores the long-lived dependencies required
     to issue requests. It must be safe for concurrent use.
   - Implement `NewAPIClient(config ClientConfig) (*APIClient, error)`.
     Construct one reusable `http.Transport` and one reusable `http.Client`.
   - Optionally explore selected transport timeouts or pooling settings after the
     core exercise works.

2. RoundTripper Middleware:
   - Implement `AuthTransport`, which injects a bearer token into every outgoing
     request. It must not mutate a request that may be reused by its caller;
     clone the request before changing its headers.
   - Implement `LoggingTransport`, which records the HTTP method, URL path,
     status code (when available), elapsed duration, and any transport error.
     It must never log the authorization value.
   - Implement `MetricsTransport`, which safely counts total requests, failed
     transport requests, and responses grouped by status-code class (2xx, 4xx,
     5xx). Choose synchronization appropriate for high-concurrency use.
   - Each middleware must delegate to a wrapped `http.RoundTripper`. Correctly
     handle a nil wrapped transport by falling back to `http.DefaultTransport`.
   - Compose the transports so all three behaviors are active for every request.

3. Request Lifecycle:
   - Implement a method that creates and executes a JSON GET request for a full
     URL. Accept a `context.Context` supplied by the caller.
   - Return an error for a non-2xx response that includes the status code and a
     useful response-body diagnostic.
   - Close every response body after reading it.

4. Demonstration in main:
   - Start a local `httptest.Server` with at least one successful endpoint and
     one endpoint returning a non-2xx response.
   - Create one configured client and make concurrent requests to both endpoints.
   - Add a deliberately slow endpoint and demonstrate that a caller's context
     cancellation reaches the outgoing request.
   - Print a metrics snapshot after all requests complete.
   - Run and verify the program with:
       go run -race ./11_http_client

Acceptance Criteria:
   - The program has no data races under the race detector.
   - The same client and transport are reused across all concurrent requests.
   - Middleware composition preserves context cancellation.
   - No authorization credential appears in logs.

Good luck! Implement your solution below.
*/

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

type ClientConfig struct {
	BaseURL        string
	BearerToken    string
	RequestTimeout time.Duration
}

type APIClient struct {
	httpClient *http.Client
	transport  http.RoundTripper
	metricsRT  *MetricsTransport
}

func NewAPIClient(config ClientConfig) (*APIClient, error) {
	if config.BaseURL == "" {
		return nil, errors.New("BaseURL cannot be empty")
	}
	m := &MetricsTransport{
		RoundTripper: &http.Transport{
			TLSHandshakeTimeout:   time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
		},
	}
	tr := &AuthTransport{
		BearerToken: config.BearerToken,
		RoundTripper: &LoggingTransport{
			RoundTripper: m,
		},
	}

	return &APIClient{
		transport: tr,
		metricsRT: m,
		httpClient: &http.Client{
			Timeout:   config.RequestTimeout,
			Transport: tr,
		},
	}, nil
}

func (a *APIClient) PrintSummary() {
	a.metricsRT.printSummary()
}

type AuthTransport struct {
	http.RoundTripper
	BearerToken string
}

func (t *AuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}
	r := request.Clone(request.Context())
	r.Header.Set("Authorization", "Bearer "+t.BearerToken)
	return rt.RoundTrip(r)
}

type LoggingTransport struct {
	http.RoundTripper
}

func (t *LoggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}
	start := time.Now()
	response, err := rt.RoundTrip(request)
	duration := time.Since(start)

	if err != nil {
		slog.ErrorContext(
			request.Context(),
			fmt.Sprintf("Request failed %s %s", request.Method, request.URL.Path),
			"method", request.Method,
			"path", request.URL.Path,
			"error", err.Error(),
			"duration", duration,
		)
	} else {
		slog.InfoContext(
			request.Context(),
			fmt.Sprintf("Request %s %s", request.Method, request.URL.Path),
			"method", request.Method,
			"path", request.URL.Path,
			"status_code", response.StatusCode,
			"duration", duration,
		)
	}

	return response, err
}

type MetricsTransport struct {
	http.RoundTripper
	TotalRequests           atomic.Int64
	FailedTransportRequests atomic.Int64
	ResponseCodes           struct {
		Status2xx atomic.Int64
		Status4xx atomic.Int64
		Status5xx atomic.Int64
	}
}

func (r *MetricsTransport) printSummary() {
	fmt.Printf("Total Requests: %d\n", r.TotalRequests.Load())
	fmt.Printf("Failed Transport Requests: %d\n", r.FailedTransportRequests.Load())
	fmt.Printf("Status 2xx: %d\n", r.ResponseCodes.Status2xx.Load())
	fmt.Printf("Status 4xx: %d\n", r.ResponseCodes.Status4xx.Load())
	fmt.Printf("Status 5xx: %d\n", r.ResponseCodes.Status5xx.Load())
}

func (t *MetricsTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}

	response, err := rt.RoundTrip(request)

	t.TotalRequests.Add(1)
	if err != nil {
		t.FailedTransportRequests.Add(1)
		return nil, err
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		t.ResponseCodes.Status2xx.Add(1)
	} else if response.StatusCode >= 400 && response.StatusCode < 500 {
		t.ResponseCodes.Status4xx.Add(1)
	} else if response.StatusCode >= 500 && response.StatusCode < 600 {
		t.ResponseCodes.Status5xx.Add(1)
	}

	return response, err
}

type APIError struct {
	StatusCode int
	Message    string
}

func (a APIError) Error() string {
	return fmt.Sprintf("API error %d. Message: %s", a.StatusCode, a.Message[0:min(len(a.Message), 30)])
}

func (c *APIClient) GetJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)

	if err == nil {
		if resp.StatusCode >= 300 {
			return nil, APIError{StatusCode: resp.StatusCode, Message: string(respBytes)}
		}
	} else {
		if resp.StatusCode >= 300 {
			return nil, APIError{StatusCode: resp.StatusCode, Message: "invalid response JSON: " + string(respBytes)}
		}
	}

	return respBytes, err
}

func main() {
	ctx, cancelF := context.WithCancel(context.Background())
	defer cancelF()

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(200)
			w.Write([]byte(`{"success":true}`))
			return
		}

		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(404)
		w.Write([]byte(`{"success":false}`))
	}))
	server.Start()
	defer server.Close()

	config := ClientConfig{
		BaseURL:        server.URL,
		BearerToken:    "TestToken",
		RequestTimeout: 20 * time.Second,
	}
	fmt.Printf("Server URL%s\n", server.URL)

	apiClient, err := NewAPIClient(config)
	if err != nil {
		panic(err)
	}

	// Test two sample responses
	response, err := apiClient.GetJSON(ctx, config.BaseURL+"/ok")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Response: %s\n", string(response))

	response, err = apiClient.GetJSON(ctx, config.BaseURL+"/bad")
	if err != nil {
		fmt.Printf("Response error: %s\n", err.Error())
	}
	fmt.Printf("Response: %s\n", string(response))

	// Concurrent requests
	w := sync.WaitGroup{}
	for range 50 {
		w.Go(func() {
			_, _ = apiClient.GetJSON(ctx, config.BaseURL+"/ok")
		})
		w.Go(func() {
			_, _ = apiClient.GetJSON(ctx, config.BaseURL+"/bad")
		})
	}
	w.Go(func() {
		time.Sleep(100 * time.Millisecond)
		cancelF()
	})
	w.Wait()

	apiClient.PrintSummary()
}
