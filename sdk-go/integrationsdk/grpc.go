package integrationsdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrRecoveredFromPanic = errors.New("recovered from panic")

type grpcServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (c *grpcServerStream) Context() context.Context {
	return c.ctx
}

func grpcRecover(ctx context.Context, method string, value any) error {
	log.Error(ctx, "recovered from panic",
		slog.String("method", method),
		slog.Any("recover", value),
	)

	switch value := value.(type) {
	case error:
		return grpcError(fmt.Errorf("method %q recovered from panic: %v", method, value))
	}

	return status.Error(codes.Internal, fmt.Sprintf("method %q recovered from panic: %v", method, value))
}

func grpcError(err error) error {
	if err == nil {
		return nil
	}

	_, ok := status.FromError(err)
	if ok {
		return err
	} else if strings.Contains(err.Error(), "not implemented") {
		return status.Error(codes.Unimplemented, err.Error())
	}

	if errors.Is(err, io.EOF) {
		return status.Error(codes.OK, err.Error())
	} else if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, err.Error())
	} else if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, err.Error())
	}

	return status.Errorf(codes.Unknown, "unknown error: %v", err)
}
