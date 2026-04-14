package memory_test

import (
	"testing"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/memory"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta2/outbox/outboxtest"
)

func TestConformance(t *testing.T) {
	outboxtest.Run(t, func(t *testing.T) outbox.Outbox {
		t.Helper()
		return memory.New()
	}, outboxtest.Options{
		EventReader: func(t *testing.T, o outbox.Outbox) outbox.EventReader {
			t.Helper()
			return o.(*memory.Store)
		},
	})
}
