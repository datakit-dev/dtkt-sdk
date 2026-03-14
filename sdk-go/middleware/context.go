package middleware

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
)

type requestCtxKey struct{}

func NewRequestContext(ctx context.Context, req *Request) context.Context {
	return AddRequestToContext(ctx, req)
}

func AddRequestToContext(ctx context.Context, req *Request) context.Context {
	return metadata.AppendToOutgoingContext(
		context.WithValue(ctx, requestCtxKey{}, req),
		RequestToGRPCPairs(req)...,
	)
}

func RequestFromContext(ctx context.Context) (*Request, error) {
	req, ok := ctx.Value(requestCtxKey{}).(*Request)
	if !ok || req == nil {
		return nil, fmt.Errorf("request not found in context")
	}
	return req, req.IsValid()
}
