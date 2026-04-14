package runtime

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"

	expr "cel.dev/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/rpc/mock"
)

// newMockRPCClient creates a mock.Client with sample RPC methods pre-registered
// for use in runtime tests.
func newMockRPCClient() *mock.Client {
	c := mock.NewClient()

	// Unary: Echo -- returns the request as-is.
	c.RegisterUnary("echo.Echo", func(_ context.Context, req proto.Message) (proto.Message, error) {
		return req, nil
	})

	// Server stream: RandomNumbers -- sends N random ints where N comes from the request.
	c.RegisterServerStream("random.Numbers", func(_ context.Context, req proto.Message, send func(proto.Message) error) error {
		n := int64(5)
		if v, ok := req.(*expr.Value); ok {
			if iv, ok := v.GetKind().(*expr.Value_Int64Value); ok {
				n = iv.Int64Value
			}
		}
		for range n {
			val, _ := nativeToExpr(rand.IntN(100))
			if err := send(val); err != nil {
				return err
			}
		}
		return nil
	})

	// Client stream: Log -- logs each message and returns the count.
	c.RegisterClientStream("log.Collect", func(_ context.Context, recv func() (proto.Message, error)) (proto.Message, error) {
		var count int
		for {
			msg, err := recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			log.Printf("[log.Collect] received: %v", msg)
			count++
		}
		resp, _ := nativeToExpr(fmt.Sprintf("logged %d messages", count))
		return resp, nil
	})

	// Bidi stream: Echo -- echoes each message back.
	c.RegisterBidiStream("echo.BidiEcho", func(_ context.Context, recv func() (proto.Message, error), send func(proto.Message) error) error {
		for {
			msg, err := recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if err := send(msg); err != nil {
				return err
			}
		}
	})

	// Bidi stream: RandomBatch -- for each incoming message, sends back a random batch.
	c.RegisterBidiStream("random.BidiBatch", func(_ context.Context, recv func() (proto.Message, error), send func(proto.Message) error) error {
		for {
			_, err := recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			batchSize := rand.IntN(3) + 1
			for range batchSize {
				val, _ := nativeToExpr(rand.IntN(100))
				if err := send(val); err != nil {
					return err
				}
			}
		}
	})

	// --- Error-producing methods ---

	// Unary: always returns NotFound.
	c.RegisterUnary("error.NotFound", func(_ context.Context, _ proto.Message) (proto.Message, error) {
		return nil, status.Error(codes.NotFound, "resource not found")
	})

	// Unary: always returns PermissionDenied.
	c.RegisterUnary("error.PermissionDenied", func(_ context.Context, _ proto.Message) (proto.Message, error) {
		return nil, status.Error(codes.PermissionDenied, "access denied")
	})

	// Unary: always returns Internal.
	c.RegisterUnary("error.Internal", func(_ context.Context, _ proto.Message) (proto.Message, error) {
		return nil, status.Error(codes.Internal, "internal server error")
	})

	// Unary: returns Unavailable for the first call, then succeeds (for retry tests).
	var unavailableCount int
	c.RegisterUnary("error.UnavailableThenOK", func(_ context.Context, req proto.Message) (proto.Message, error) {
		unavailableCount++
		if unavailableCount == 1 {
			return nil, status.Error(codes.Unavailable, "service temporarily unavailable")
		}
		return req, nil
	})

	// Server stream: sends 2 messages then fails with Aborted.
	c.RegisterServerStream("error.StreamAborted", func(_ context.Context, _ proto.Message, send func(proto.Message) error) error {
		for i := range 2 {
			val, _ := nativeToExpr(i)
			if err := send(val); err != nil {
				return err
			}
		}
		return status.Error(codes.Aborted, "stream aborted mid-flight")
	})

	// Server stream: fails with Unavailable on first call, succeeds on second (for retry tests).
	var streamUnavailableCount int
	c.RegisterServerStream("error.StreamUnavailableThenOK", func(_ context.Context, req proto.Message, send func(proto.Message) error) error {
		streamUnavailableCount++
		if streamUnavailableCount == 1 {
			return status.Error(codes.Unavailable, "stream temporarily unavailable")
		}
		return send(req)
	})

	// Server stream: closes immediately with no messages (EOF).
	c.RegisterServerStream("error.StreamEmpty", func(_ context.Context, _ proto.Message, _ func(proto.Message) error) error {
		return nil
	})

	// Unary: blocks forever until context is cancelled (simulates hung/unresponsive RPC).
	c.RegisterUnary("error.Hang", func(ctx context.Context, _ proto.Message) (proto.Message, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	// Server stream: opens successfully but never sends or closes (idle subscription).
	c.RegisterServerStream("stream.Idle", func(ctx context.Context, _ proto.Message, _ func(proto.Message) error) error {
		<-ctx.Done()
		return ctx.Err()
	})

	// Bidi stream: accepts one message then returns DeadlineExceeded.
	c.RegisterBidiStream("error.BidiDeadline", func(_ context.Context, recv func() (proto.Message, error), _ func(proto.Message) error) error {
		if _, err := recv(); err != nil {
			return err
		}
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	})

	// Client stream: accepts 2 messages then returns InvalidArgument (mid-stream error).
	c.RegisterClientStream("error.ClientStreamInvalid", func(_ context.Context, recv func() (proto.Message, error)) (proto.Message, error) {
		for range 2 {
			if _, err := recv(); err != nil {
				return nil, err
			}
		}
		return nil, status.Error(codes.InvalidArgument, "invalid payload")
	})

	return c
}

// mockRPCOptions returns a WithConnectors option backed by the mock client
// registered under every connection ID used in testdata flows.
func mockRPCOptions() []Option {
	c := newMockRPCClient()
	connectors := rpc.Connectors{}
	for _, id := range []string{"echo", "random", "log"} {
		connectors[id] = &rpc.Connector{Client: c, Resolver: c}
	}
	return []Option{WithConnectors(connectors)}
}
