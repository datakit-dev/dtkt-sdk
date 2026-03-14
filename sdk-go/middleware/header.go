package middleware

import (
	"net/http"
	"strconv"

	"google.golang.org/grpc/metadata"
)

const (
	AddrNameHeader   = "dtkt-addr-name"
	ConfigHashHeader = "dtkt-config-hash"
	ConfigGenHeader  = "dtkt-config-gen"
)

func AllowedHeaders() []string {
	return []string{
		AddrNameHeader,
		ConfigHashHeader,
		ConfigGenHeader,
	}
}

func GetFromGRPC(md metadata.MD, key string) (val string) {
	vals := md.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return
}

func RequestToGRPCPairs(req *Request) []string {
	return []string{
		AddrNameHeader, req.addrName,
		ConfigHashHeader, req.configHash,
		ConfigGenHeader, strconv.Itoa(int(req.configGen)),
	}
}

func RequestToGRPC(req *Request) metadata.MD {
	return metadata.Pairs(RequestToGRPCPairs(req)...)
}

func RequestFromGRPC(md metadata.MD) *Request {
	configGen, _ := strconv.ParseUint(GetFromGRPC(md, ConfigGenHeader), 10, 64)
	return NewRequest(
		GetFromGRPC(md, AddrNameHeader),
		GetFromGRPC(md, ConfigHashHeader),
		configGen,
	)
}

func RequestToHeader(req *Request) http.Header {
	header := http.Header{}
	header.Add(AddrNameHeader, req.addrName)
	header.Add(ConfigHashHeader, req.configHash)
	header.Add(ConfigGenHeader, strconv.Itoa(int(req.configGen)))
	return header
}

func RequestFromHeader(header http.Header) *Request {
	configGen, _ := strconv.ParseUint(header.Get(ConfigGenHeader), 10, 64)
	return NewRequest(
		header.Get(AddrNameHeader),
		header.Get(ConfigHashHeader),
		configGen,
	)
}
