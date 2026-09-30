package middleware

import (
	"fmt"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

type (
	Request struct {
		resourceName string // name of resource associated with config
		configHash   string // sha256 hash of config
		configGen    uint64 // bumps on config/auth changes
	}
	Resource interface {
		GetName() string
		GetConfigGen() uint64
		GetConfigHash() string
	}
)

func NewRequest(resourceName, configHash string, configGen uint64) *Request {
	return &Request{
		resourceName: resourceName,
		configHash:   configHash,
		configGen:    configGen,
	}
}

func ResourceRequest(res Resource) *Request {
	return NewRequest(
		res.GetName(),
		res.GetConfigHash(),
		res.GetConfigGen(),
	)
}

func (r *Request) ResourceName() string { return r.resourceName }
func (r *Request) ConfigHash() string   { return r.configHash }
func (r *Request) ConfigGen() uint64    { return r.configGen }

func (r *Request) IsValid() error {
	if r == nil {
		return fmt.Errorf("invalid request: nil")
	} else if r.resourceName == "" {
		return fmt.Errorf("resource name required")
	} else if r.configHash == "" {
		return fmt.Errorf("config hash required")
	}
	return nil
}

func (r *Request) HashSHA256() string {
	return util.HashSHA256(r.String())
}

func (r *Request) PrevSHA256() string {
	configGen := r.configGen
	if configGen > 0 {
		configGen--
	}
	return util.HashSHA256(configGenSignature(r.resourceName, configGen))
}

func (r *Request) String() string {
	return configGenSignature(r.resourceName, r.configGen)
}

func (r *Request) SetHeader(req *http.Request) {
	for key, vals := range RequestToHeader(r) {
		for _, val := range vals {
			req.Header.Set(key, val)
		}
	}
}

func configGenSignature(resourceName string, configGen uint64) string {
	return fmt.Sprintf("resource=%q|gen=%d", resourceName, configGen)
}
