package cloud

import (
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/network"
	"google.golang.org/grpc/metadata"
)

func GetFromGRPC(md metadata.MD, key string) (val string) {
	vals := md.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return
}

func RequestToGRPCPairs(req *Request) []string {
	return []string{
		network.AuthHeader, "Bearer: " + req.Token,
		network.CloudOrgHeader, req.Org,
		network.CloudSpaceHeader, req.Space,
	}
}

func RequestToGRPC(req *Request) metadata.MD {
	return metadata.Pairs(RequestToGRPCPairs(req)...)
}

func RequestFromGRPC(md metadata.MD) *Request {
	return NewRequest(
		WithToken(GetFromGRPC(md, network.AuthHeader)),
		WithOrg(GetFromGRPC(md, network.CloudOrgHeader)),
		WithSpace(GetFromGRPC(md, network.CloudSpaceHeader)),
	)
}

func RequestToHeader(req *Request) http.Header {
	header := http.Header{}
	header.Add(network.AuthHeader, "Bearer: "+req.Token)
	header.Add(network.CloudOrgHeader, req.Org)
	header.Add(network.CloudSpaceHeader, req.Space)
	return header
}

func RequestFromHeader(header http.Header) *Request {
	return NewRequest(
		WithToken(header.Get(network.AuthHeader)),
		WithOrg(header.Get(network.CloudOrgHeader)),
		WithSpace(header.Get(network.CloudSpaceHeader)),
	)
}
