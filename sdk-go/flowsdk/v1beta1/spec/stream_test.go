package spec_test

// // stream_test.go — black-box tests for ServerStream, ClientStream, BidiStream.
// // Stream nodes are constructed via exported Test* helpers in export_test.go which
// // are only compiled during tests of this package.

// import (
// "context"
// "io"
// "testing"
// "time"

// "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec"
// "github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/v1beta1/spec/spectest"
// flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
// "github.com/google/cel-go/common/types/ref"
// "google.golang.org/protobuf/proto"
// )

// // ---- helpers ---------------------------------------------------------------

// // collectSends launches sendFn in a goroutine and returns channels for the
// // emitted values and the final error.
// func collectSends(sendFn func(run interface {
// Context() context.Context
// 	Env() (interface {
// 		TypeProvider() interface{}
// 		TypeAdapter() interface {
// 			NativeToValue(interface{}) ref.Val
// 		}
// 		Vars() interface{}
// 		Resolver() interface{}
// 	}, error)
// 	Connectors() interface{}
// 	GetNode(string) (interface{}, bool)
// 	GetValue(string) (any, error)
// 	GetSendCh(string) (chan<- ref.Val, error)
// 	GetRecvCh(string) (<-chan any, error)
// }, vch chan<- ref.Val) error, run interface{}, vch chan<- ref.Val) {}

// // Simpler: just call the typed send function directly.

// // ---- BidiStream tests ------------------------------------------------------

// // TestBidiStream_SendDoneUnblocksRecv verifies that when the response loop exits
// // on EOF the request loop also exits rather than blocking forever.
// func TestBidiStream_SendDoneUnblocksRecv(t *testing.T) {
// 	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
// 	defer cancel()

// 	mockStream := spectest.NewMockBidiStream(ctx)
// 	close(mockStream.Responses) // EOF immediately

// 	bs := spec.TestNewBidiStream(ctx, "test", mockStream,
// func(run spec.Runtime) (proto.Message, error) {
// return &flowv1beta1.Runtime_Done{Reason: "req"}, nil
// 		},
// 	)

// 	run := spectest.NewMockRuntime(ctx)
// 	recvFn, hasRecv := bs.Recv()
// 	sendFn, hasSend := bs.Send()
// 	if !hasRecv || !hasSend {
// 		t.Fatal("expected both Recv and Send")
// 	}

// 	recvCh := make(chan any, 4)
// 	recvErrCh := make(chan error, 1)
// 	sendErrCh := make(chan error, 1)

// 	go func() {
// 		valCh := make(chan ref.Val, 8)
// 		go func() { for range valCh {} }()
// 		sendErrCh <- sendFn(run, valCh)
// 	}()
// 	go func() { recvErrCh <- recvFn(run, recvCh) }()

// 	deadline := time.After(3 * time.Second)
// 	var recvDone, sDone bool
// 	for !recvDone || !sDone {
// 		select {
// 		case err := <-recvErrCh:
// 			if err != nil {
// 				t.Errorf("Recv() unexpected error: %v", err)
// 			}
// 			recvDone = true
// 		case err := <-sendErrCh:
// 			if err != nil && err != io.EOF {
// 				t.Errorf("Send() unexpected error: %v", err)
// 			}
// 			sDone = true
// 		case <-deadline:
// 			t.Fatal("timeout: BidiStream goroutines did not exit after Send EOF")
// 		}
// 	}
// }
