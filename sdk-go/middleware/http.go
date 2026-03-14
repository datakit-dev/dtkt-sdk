package middleware

import (
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
)

func HTTPTransport(parent http.RoundTripper) http.RoundTripper {
	return network.HTTPRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if req, _ := RequestFromContext(r.Context()); req != nil {
			req.SetHeader(r)
		}
		return parent.RoundTrip(r)
	})
}
