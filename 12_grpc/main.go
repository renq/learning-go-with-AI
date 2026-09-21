package main

/*
TASK 12: gRPC Client & Server Architecture, Interceptors, and Context Metadata

Topic: Networking – gRPC Services, Interceptors, Metadata, and Deadlines

Problem Description:
gRPC is a high-performance RPC framework widely used for inter-service communication.
In Go, production gRPC services rely on:
1. Context-driven Metadata propagation (request IDs, auth tokens, tracing headers).
2. Unary Client and Server Interceptors (middleware for authentication, metrics, logging).
3. Rich error handling with standard gRPC status codes (`google.golang.org/grpc/status` and `codes`).
4. Context timeouts / deadline propagation across network boundaries.
5. In-process listener testing (`google.golang.org/grpc/test/bufconn`) and graceful shutdown (`GracefulStop`).

Requirements:

1. Service Definition & Data Types:
   - Define request/response payload types for an `OrderService`:
     * `OrderRequest` (fields: `OrderID string`, `Amount float64`, `CustomerID string`)
     * `OrderResponse` (fields: `OrderID string`, `Status string`, `ProcessedAt int64`)
   - Define `OrderServer` struct implementing the business logic:
     * `CreateOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)`:
       - Checks if `req.Amount <= 0`: returns error with `codes.InvalidArgument`.
       - Reads incoming metadata from `ctx` using `metadata.FromIncomingContext(ctx)`:
         * Extracts `x-request-id`.
       - If `req.OrderID == "slow"`: simulates slow processing (`time.Sleep(200 * time.Millisecond)`).
       - Returns a successful `*OrderResponse` with `Status: "CONFIRMED"`.

2. Server-Side Unary Interceptor (`grpc.UnaryServerInterceptor`):
   - Implement `AuthAndLoggingServerInterceptor()`:
     * Extracts incoming metadata using `metadata.FromIncomingContext(ctx)`.
     * Checks `authorization` header:
       - If missing or not equal to `"Bearer secret-token"`, rejects request immediately with `status.Errorf(codes.Unauthenticated, "invalid or missing auth token")`.
     * Measures method execution time and logs method name, request ID, and duration.
     * Calls `handler(ctx, req)` to proceed.

3. Client-Side Unary Interceptor (`grpc.UnaryClientInterceptor`):
   - Implement `RequestIDAndAuthClientInterceptor(token string)`:
     * Automatically injects `authorization: Bearer <token>` and a generated `x-request-id` into outgoing context using `metadata.NewOutgoingContext(ctx, md)`.
     * Invokes `invoker(ctx, method, req, reply, cc, opts...)`.

4. In-Memory gRPC Test Infrastructure (`bufconn`):
   - Use `google.golang.org/grpc/test/bufconn` buffer listener (`bufconn.Listen(1024 * 1024)`):
     * Creates an in-memory gRPC server with the server interceptor.
     * Creates an in-memory gRPC client connection (`grpc.NewClient` or `grpc.DialContext` with `grpc.WithContextDialer` and `insecure.NewCredentials()`).

5. In `main()` function:
   - Start the in-memory gRPC server in a background goroutine.
   - Setup client with interceptors.
   - Run test scenarios:
     a) Scenario 1 (Happy Path): Send a valid order (`OrderID: "ord-1", Amount: 99.99`) -> verify success.
     b) Scenario 2 (Validation Error): Send invalid order (`Amount: -10`) -> verify client receives `codes.InvalidArgument` using `status.FromError(err)`.
     c) Scenario 3 (Authentication Failure): Call without valid auth token -> verify client receives `codes.Unauthenticated`.
     d) Scenario 4 (Deadline Exceeded): Call `OrderID: "slow"` with a short context timeout (e.g., `50ms`) -> verify client receives `codes.DeadlineExceeded`.
   - Perform graceful shutdown: `grpcServer.GracefulStop()` and `conn.Close()`.
   - Verify that the program finishes cleanly and passes `go run -race 12_grpc/main.go`.

Good luck! Implement your solution below.
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"time"
	"uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type OrderRequest struct {
	OrderID    string
	Amount     float64
	CustomerID string
}

type OrderResponse struct {
	OrderID     string
	Status      string
	ProcessedAt int64
}

type OrderServiceServer interface {
	CreateOrder(context.Context, *OrderRequest) (*OrderResponse, error)
}

type OrderServer struct {
}

func (o *OrderServer) CreateOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	if req.Amount <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "amount must be greater than zero")
	}
	md, _ := metadata.FromIncomingContext(ctx) // should I do anything with second return value?
	requestID, ok := md["x-request-id"]        // why do I need to extract it?
	if ok {
		fmt.Printf("RequestID: %s\n", requestID)
	} else {
		fmt.Printf("RequestID not found in the context\n")
	}

	if req.OrderID == "slow" {
		time.Sleep(200 * time.Millisecond)
	}
	return &OrderResponse{
		OrderID:     req.OrderID,
		Status:      "CONFIRMED",
		ProcessedAt: time.Now().UnixMicro(),
	}, nil
}

func AuthAndLoggingServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "no metadata")
	}
	auth, ok := md["authorization"]
	if !ok || !slices.Contains(auth, "Bearer secret-token") {
		return nil, status.Errorf(codes.Unauthenticated, "invalid or missing auth token")
	}
	start := time.Now()
	resp, err := handler(ctx, req)

	requestIDs := md.Get("x-request-id")
	var reqID string
	if len(requestIDs) > 0 {
		reqID = requestIDs[0]
	}

	slog.InfoContext(ctx, "auth and logging", "duration", time.Since(start), "method", info.FullMethod, "request_id", reqID)

	return resp, err
}

func RequestIDAndAuthClientInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cctx := metadata.NewOutgoingContext(ctx, metadata.MD{
			"authorization": []string{"Bearer " + token},
			"x-request-id":  []string{uuid.New().String()},
		})

		return invoker(cctx, method, req, reply, cc, opts...)
	}
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (jsonCodec) Name() string                       { return "json" }

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

func main() {
	ctx, cancelF := context.WithTimeout(context.Background(), time.Millisecond*190)
	defer cancelF()

	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(AuthAndLoggingServerInterceptor),
	)
	server.RegisterService(
		&grpc.ServiceDesc{
			ServiceName: "OrderService",
			HandlerType: (*OrderServiceServer)(nil),
			Methods: []grpc.MethodDesc{
				{
					MethodName: "CreateOrder",
					Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
						req := new(OrderRequest)
						if err := dec(req); err != nil {
							return nil, err
						}
						if interceptor == nil {
							return srv.(*OrderServer).CreateOrder(ctx, req)
						}
						info := &grpc.UnaryServerInfo{
							Server:     srv,
							FullMethod: "/OrderService/CreateOrder",
						}
						handler := func(ctx context.Context, req any) (any, error) {
							return srv.(*OrderServer).CreateOrder(ctx, req.(*OrderRequest))
						}
						return interceptor(ctx, req, info, handler)
					},
				},
			},
		},
		&OrderServer{},
	)

	go server.Serve(listener)

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, token string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(RequestIDAndAuthClientInterceptor("secret-token")),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)

	if err != nil {
		panic(err)
	}

	// Scenario 1 (Happy Path)
	req := &OrderRequest{OrderID: "ord-1", Amount: 100, CustomerID: "cust-123"}
	var resp OrderResponse
	err = conn.Invoke(ctx, "/OrderService/CreateOrder", req, &resp)
	if err != nil {
		panic(err)
	}

	// Validation error
	req = &OrderRequest{OrderID: "ord-1", Amount: -10, CustomerID: "cust-123"}
	err = conn.Invoke(ctx, "/OrderService/CreateOrder", req, &resp)
	if err != nil {
		fmt.Printf("Validation error %s", err.Error())
	}

	// Scenario 2 (Validation Error)
	req = &OrderRequest{OrderID: "ord-1", Amount: -10, CustomerID: "cust-123"}
	err = conn.Invoke(ctx, "/OrderService/CreateOrder", req, &resp)
	if err != nil {
		fmt.Printf("Validation error %s", err.Error())
	}

	// Scenario 3 (Authentication Failure)
	conn2, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, token string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(RequestIDAndAuthClientInterceptor("bad token")),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	req = &OrderRequest{OrderID: "order-1", Amount: 110, CustomerID: "cust-123"}
	err = conn2.Invoke(ctx, "/OrderService/CreateOrder", req, &resp)
	if err != nil {
		fmt.Printf("Validation error %s", err.Error())
	}

	// Scenario 4 (Deadline Exceeded)
	req = &OrderRequest{OrderID: "slow", Amount: 110, CustomerID: "cust-123"}
	err = conn.Invoke(ctx, "/OrderService/CreateOrder", req, &resp)
	if err != nil {
		fmt.Printf("Validation error %s", err.Error())
	}

	conn.Close()
	conn2.Close()
	server.GracefulStop()

	// TODO 7: Run scenarios (Happy path, InvalidArgument, Unauthenticated, DeadlineExceeded, and GracefulStop)
	/*
		   - Run test scenarios:
		     a) Scenario 1 (Happy Path): Send a valid order (`OrderID: "ord-1", Amount: 99.99`) -> verify success.
		     b) Scenario 2 (Validation Error): Send invalid order (`Amount: -10`) -> verify client receives `codes.InvalidArgument` using `status.FromError(err)`.
		     c) Scenario 3 (Authentication Failure): Call without valid auth token -> verify client receives `codes.Unauthenticated`.
		     d) Scenario 4 (Deadline Exceeded): Call `OrderID: "slow"` with a short context timeout (e.g., `50ms`) -> verify client receives `codes.DeadlineExceeded`.
		   - Perform graceful shutdown: `grpcServer.GracefulStop()` and `conn.Close()`.
		   - Verify that the program finishes cleanly and passes `go run -race 12_grpc/main.go`.

				Good luck! Implement your solution below.
	*/
}
