package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/pubsub"
)

// Recoverer catches panics from the handler and converts them to errors.
func Recoverer(h pubsub.HandlerFunc) pubsub.HandlerFunc {
	return func(msg *pubsub.Message) (msgs []*pubsub.Message, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("handler panicked: %v\n%s", r, debug.Stack())
			}
		}()
		return h(msg)
	}
}
