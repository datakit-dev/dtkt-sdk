package cloud

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"google.golang.org/grpc/metadata"
)

const (
	DefaultApiUrl     = "https://api.datakit.cloud"
	DefaultAuthUrl    = "https://auth.datakit.cloud"
	DefaultSchemasUrl = "https://schemas.datakit.cloud"
)

type (
	Request struct {
		// Auth URL for DataKit Cloud.
		AuthUrl string `json:"authUrl,omitempty"`
		// API URL for DataKit Cloud.
		ApiUrl string `json:"apiUrl,omitempty"`
		// Bearer token of a DataKit Cloud User.
		Token string `json:"token,omitempty"`
		// DataKit Cloud Organization.
		Org string `json:"org,omitempty"`
		// DataKit Cloud Space.
		Space string `json:"space,omitempty"`
	}
	requestCtxKey struct{}
)

func NewRequest(opts ...RequestOpt) *Request {
	req := &Request{}
	req.SetOptions(opts...)

	if req.ApiUrl == "" {
		req.ApiUrl = DefaultApiUrl
	}

	if req.AuthUrl == "" {
		req.AuthUrl = DefaultAuthUrl
	}

	return req
}

func (r *Request) GetGraphUrl() string {
	return fmt.Sprintf("%s/graphql", r.ApiUrl)
}

func (r *Request) SetOptions(opts ...RequestOpt) {
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
}

func (r *Request) SetHeader(req *http.Request) error {
	if r.Token != "" {
		req.Header.Set(network.AuthHeader, fmt.Sprintf("Bearer %s", r.Token))
	} else {
		return fmt.Errorf("cloud bearer token required")
	}

	if r.Org != "" {
		req.Header.Set(network.CloudOrgHeader, r.Org)

		if r.Space != "" {
			req.Header.Set(network.CloudSpaceHeader, r.Space)
		}
	}
	return nil
}

func NewRequestContext(ctx context.Context, opts ...RequestOpt) context.Context {
	return AddToContext(ctx, NewRequest(opts...))
}

func AddToContext(ctx context.Context, req *Request) context.Context {
	if req == nil {
		return ctx
	}
	return metadata.AppendToOutgoingContext(context.WithValue(ctx, requestCtxKey{}, req), RequestToGRPCPairs(req)...)
}

func FromContext(ctx context.Context) (*Request, bool) {
	req, ok := ctx.Value(requestCtxKey{}).(*Request)
	return req, ok
}

func (r *Request) String() string {
	var parts []string
	if r.AuthUrl != "" {
		parts = append(parts, fmt.Sprintf("AuthUrl=%s", r.AuthUrl))
	}
	if r.ApiUrl != "" {
		parts = append(parts, fmt.Sprintf("ApiUrl=%s", r.ApiUrl))
	}
	if r.Token != "" {
		parts = append(parts, fmt.Sprintf("Token=%s", r.Token))
	}
	if r.Org != "" {
		parts = append(parts, fmt.Sprintf("Org=%s", r.Org))
	}
	if r.Space != "" {
		parts = append(parts, fmt.Sprintf("Space=%s", r.Space))
	}
	return strings.Join(parts, " ")
}
