package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/lib/log"
	flowv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta1"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ shared.Env = (*Node)(nil)
var _ shared.GraphNode = (*Node)(nil)

type Node struct {
	*GraphNode
	proto *flowv1beta1.Env_Node

	send,
	recv chan any
	pipe *Pipe[any, any]

	prevValue,
	currValue any
	err error

	mut sync.RWMutex
}

func newNode(graphNode *GraphNode, proto *flowv1beta1.Env_Node) (_ *Node, err error) {
	if proto == nil {
		return nil, fmt.Errorf("node proto cannot be nil")
	}

	recv := make(chan any)
	send := make(chan any)

	var currValue any
	if proto.ResultCurr == nil {
		proto.ResultCurr = &flowv1beta1.Env_Result{}
	} else if proto.ResultCurr.GetValue() != nil {
		currValue, err = common.ProtoOptions{
			Resolver: graphNode.Env.Resolver(),
		}.UnwrapAny(proto.ResultCurr.GetValue())
		if err != nil {
			return nil, err
		}
	}

	var prevValue any
	if proto.ResultPrev != nil && proto.ResultPrev.GetValue() != nil {
		prevValue, err = common.ProtoOptions{
			Resolver: graphNode.Env.Resolver(),
		}.UnwrapAny(proto.ResultPrev.GetValue())
		if err != nil {
			return nil, err
		}
	}

	return &Node{
		GraphNode: graphNode,
		proto:     proto,

		send: send,
		recv: recv,
		pipe: NewPipe(recv, send),

		prevValue: prevValue,
		currValue: currValue,
	}, nil
}

func (n *Node) Vars() (cel.Activation, error) {
	base, err := n.Env.Vars()
	if err != nil {
		return nil, err
	}

	self, err := cel.NewActivation(map[string]any{
		"self": func() ref.Val {
			vars, err := n.vars()
			if err != nil {
				return types.WrapErr(err)
			}

			return n.Env.Types().NativeToValue(vars)
		},
	})
	if err != nil {
		return nil, err
	}

	return interpreter.NewHierarchicalActivation(base, self), nil
}

func (n *Node) Proto() *flowv1beta1.Env_Node {
	n.mut.RLock()
	defer n.mut.RUnlock()

	return proto.CloneOf(n.proto)
}

func (n *Node) Ident() string {
	return n.ident
}

func (n *Node) Pipe() shared.Pipe {
	return n.pipe
}

func (n *Node) Send() chan<- any {
	return n.send
}

func (n *Node) Recv() <-chan any {
	return n.recv
}

func (n *Node) Run(ctx context.Context) error {
	if n.Env == nil {
		return fmt.Errorf("node %q must be parsed before running", n.ident)
	} else if n.parsed == nil {
		return fmt.Errorf("node %q must be compiled before running", n.ident)
	}

	ctx = log.NewCtx(ctx, log.FromCtx(ctx).With(slog.String("ident", n.ident)))

	return n.parsed.Run(ctx, n, n.pipe)
}

func (n *Node) Value() any {
	n.mut.RLock()
	defer n.mut.RUnlock()

	return n.currValue
}

func (n *Node) vars() (vars map[string]any, _ error) {
	n.mut.RLock()
	defer n.mut.RUnlock()

	if n.err != nil {
		return nil, n.err
	}

	vars = n.parsed.Metadata()
	if vars == nil {
		vars = map[string]any{}
	}

	maps.Copy(vars, map[string]any{
		"prev":        n.prevValue,
		"value":       n.currValue,
		"count":       n.proto.GetResultCount(),
		"start_time":  proto.CloneOf(n.proto.GetResultCurr().GetStartTime()),
		"finish_time": proto.CloneOf(n.proto.GetResultCurr().GetFinishTime()),
	})

	return
}

func (n *Node) setPending() {
	n.mut.Lock()
	defer n.mut.Unlock()

	n.proto.ResultPrev = proto.CloneOf(n.proto.ResultCurr)
	n.prevValue = n.currValue

	proto.Reset(n.proto.ResultCurr)
	n.proto.ResultCurr.StartTime = timestamppb.Now()
}

func (n *Node) setValue(value any) (_ any, err error) {
	n.mut.Lock()
	defer n.mut.Unlock()

	if value == nil {
		value = types.NullValue
	}

	switch v := value.(type) {
	case *types.Err:
		_, err = v.ConvertToNative(nil)
		value, err = n.maybeSetError(err)
		if err != nil {
			return
		}
	case error:
		value, err = n.maybeSetError(err)
		if err != nil {
			return
		}
	case ref.Val:
		value, err = n.Env.Types().ValueToNative(v)
		if err != nil {
			value, err = n.maybeSetError(err)
			if err != nil {
				return
			}
		}
	}

	if skip, ok := IsSkip(value); ok {
		value = skip
	} else if done, ok := IsDone(value); ok {
		if done.proto.GetIdent() == "" {
			done.proto.Ident = &n.ident
		}
		value = done.proto
	}

	proto, err := common.ProtoOptions{
		Resolver: n.Env.Resolver(),
	}.WrapAny(value)
	if err != nil {
		return nil, err
	}

	n.currValue = value
	n.setResultValue(proto)

	return value, nil
}

func (n *Node) setResultValue(value *anypb.Any) {
	n.proto.ResultCurr.Type = &flowv1beta1.Env_Result_Value{
		Value: value,
	}
	n.proto.ResultCurr.FinishTime = timestamppb.Now()
	n.proto.ResultCount++
}

func (n *Node) maybeSetError(err error) (any, error) {
	done, ok := IsDone(err)
	if !ok {
		statusErr, ok := status.FromError(err)
		if !ok {
			statusErr = status.New(codes.Aborted, err.Error())
		}

		done = NewDone(statusErr)
	}

	if done.IsError() {
		done.proto.Ident = &n.ident
		n.err = done
		n.proto.ResultCurr.Type = &flowv1beta1.Env_Result_Error{
			Error: done.Status().Proto(),
		}
		n.proto.ResultCurr.FinishTime = timestamppb.Now()
		n.proto.ResultCount++

		return nil, n.err
	}

	return done.Proto(), nil
}

func mergeMessage(resolver shared.Resolver, dst, src proto.Message) error {
	if src.ProtoReflect().Descriptor() == dst.ProtoReflect().Descriptor() {
		proto.Merge(dst, src)
		return nil
	}

	b, err := proto.Marshal(src)
	if err != nil {
		return err
	}

	return proto.UnmarshalOptions{
		Resolver:       resolver,
		DiscardUnknown: true,
		Merge:          true,
	}.Unmarshal(b, dst)
}
