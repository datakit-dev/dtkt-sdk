package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

type (
	MutationError interface {
		GetError() *string
	}
	CreatedError interface {
		MutationError
		GetCreated() bool
	}
	UpdatedError interface {
		MutationError
		GetUpdated() bool
	}
	DeletedError interface {
		MutationError
		GetDeleted() bool
	}
)

func GetGraphClient(ctx context.Context) (graphql.Client, error) {
	if req, ok := FromContext(ctx); ok {
		if req.GetGraphUrl() == "" {
			return nil, fmt.Errorf("graph url missing in context")
		}
		return graphql.NewClient(req.GetGraphUrl(), &http.Client{
			Transport: HTTPTransport(http.DefaultTransport),
		}), nil
	}
	return nil, fmt.Errorf("request missing from context, call Auth first")
}

func IsMutationError(mut MutationError) bool {
	switch err := mut.(type) {
	case CreatedError:
		return !err.GetCreated()
	case UpdatedError:
		return !err.GetUpdated()
	case DeletedError:
		return !err.GetDeleted()
	}
	return false
}

func HasMutationError(err error, mut MutationError) error {
	if err != nil {
		return err
	} else if IsMutationError(mut) {
		if mut.GetError() == nil {
			return errors.New("unknown error")
		}
		return errors.New(*mut.GetError())
	}
	return nil
}
