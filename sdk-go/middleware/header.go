package middleware

import (
	"net/http"
	"strconv"

	"google.golang.org/grpc/metadata"
)

const (
	ResourceNameHeader = "dtkt-resource-name"
	ConfigHashHeader   = "dtkt-config-hash"
	ConfigGenHeader    = "dtkt-config-gen"
)

func AllowedHeaders() []string {
	return []string{
		ResourceNameHeader,
		ConfigHashHeader,
		ConfigGenHeader,
	}
}

func HeaderFromGRPC(md metadata.MD, key string) (val string) {
	vals := md.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return
}

func RequestToGRPCPairs(req *Request) []string {
	return []string{
		ResourceNameHeader, req.resourceName,
		ConfigHashHeader, req.configHash,
		ConfigGenHeader, strconv.Itoa(int(req.configGen)),
	}
}

func RequestToGRPC(req *Request) metadata.MD {
	return metadata.Pairs(RequestToGRPCPairs(req)...)
}

func RequestFromGRPC(md metadata.MD) *Request {
	configGen, _ := strconv.ParseUint(HeaderFromGRPC(md, ConfigGenHeader), 10, 64)
	return NewRequest(
		HeaderFromGRPC(md, ResourceNameHeader),
		HeaderFromGRPC(md, ConfigHashHeader),
		configGen,
	)
}

func RequestToHeader(req *Request) (header http.Header) {
	header = http.Header{}
	header.Add(ResourceNameHeader, req.resourceName)
	header.Add(ConfigHashHeader, req.configHash)
	header.Add(ConfigGenHeader, strconv.Itoa(int(req.configGen)))
	return
}

func RequestFromHeader(header http.Header) *Request {
	configGen, _ := strconv.ParseUint(header.Get(ConfigGenHeader), 10, 64)
	return NewRequest(
		header.Get(ResourceNameHeader),
		header.Get(ConfigHashHeader),
		configGen,
	)
}
