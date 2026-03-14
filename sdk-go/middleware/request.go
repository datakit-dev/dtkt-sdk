package middleware

import (
	"fmt"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

type (
	Request struct {
		addrName   string // addressable resource name (Deployment or Connection)
		configHash string // sha256 hash of config
		configGen  uint64 // bumps on config/auth changes

		reqHash string // hash of request (server only)
	}
	Connection interface {
		GetName() string
		GetConfigGen() uint64
		GetConfigHash() string
	}
)

func NewRequest(addrName, configHash string, configGen uint64) *Request {
	return &Request{addrName: addrName, configHash: configHash, configGen: configGen}
}

func NewConnectionRequest(conn Connection) *Request {
	return NewRequest(
		conn.GetName(),
		conn.GetConfigHash(),
		conn.GetConfigGen(),
	)
}

func (r *Request) IsValid() error {
	if r == nil {
		return fmt.Errorf("invalid request: nil")
	}
	if r.addrName == "" {
		return fmt.Errorf("addressable name required")
	}
	if r.configHash == "" {
		return fmt.Errorf("config hash required")
	}
	return nil
}

func (r Request) Prev() string {
	prev := r
	prev.configGen -= 1
	return prev.String()
}

func (r *Request) String() string {
	if r.reqHash == "" {
		r.reqHash = util.HashSHA256(fmt.Sprintf("%s|gen=%d", r.addrName, r.configGen))
	}
	return r.reqHash
}

func (r *Request) AddrName() string   { return r.addrName }
func (r *Request) ConfigHash() string { return r.configHash }
func (r *Request) ConfigGen() uint64  { return r.configGen }

func (r *Request) SetHeader(req *http.Request) {
	for key, vals := range RequestToHeader(r) {
		for _, val := range vals {
			req.Header.Set(key, val)
		}
	}
}
