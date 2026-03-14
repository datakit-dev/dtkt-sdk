package cloud

import (
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
)

func HTTPTransport(parent http.RoundTripper) http.RoundTripper {
	return network.HTTPRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if req, ok := FromContext(r.Context()); ok {
			err := req.SetHeader(r)
			if err != nil {
				return nil, err
			}
		}
		return parent.RoundTrip(r)
	})
}
