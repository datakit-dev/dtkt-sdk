package v1beta1

import (
	context "context"
	"fmt"
	"strings"
	"time"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/common"
	actionv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/action/v1beta1"
	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const ActionsPrefix = "actions"

var _ RegisteredAction = (*actionWrapper[any, any])(nil)

type (
	ActionRegistry struct {
		m util.SyncMap[string, RegisteredAction]
	}
	RegisteredAction interface {
		Execute(context.Context, *actionv1beta1.ExecuteActionRequest) (*actionv1beta1.ExecuteActionResponse, error)
		Proto() *actionv1beta1.Action
	}
	RegisterActionFunc[I InstanceType]             func(InstanceMux[I]) (RegisteredAction, error)
	ExecuteActionFunc[I InstanceType, In, Out any] func(context.Context, InstanceMux[I], In) (Out, error)
	actionWrapper[In, Out any]                     struct {
		inputSchema  *TypeSchema[In]
		outputSchema *TypeSchema[Out]
		action       *actionv1beta1.Action
		execFunc     func(context.Context, In) (Out, error)
	}
	actionRun struct {
		req         *actionv1beta1.ExecuteActionRequest
		id          uuid.UUID
		start, done time.Time
	}
)

func RegisterActions[I InstanceType](mux InstanceMux[I], newActions ...RegisterActionFunc[I]) error {
	for _, regFunc := range newActions {
		action, err := regFunc(mux)
		if err != nil {
			return err
		}
		mux.Actions().m.Store(action.Proto().GetName(), action)
	}
	return nil
}

func NewAction[I InstanceType, In, Out any, S ~string](
	displayName S,
	description string,
	execFunc ExecuteActionFunc[I, In, Out],
) RegisterActionFunc[I] {
	return func(mux InstanceMux[I]) (RegisteredAction, error) {
		if displayName == "" {
			return nil, fmt.Errorf("failed to register action: name is required")
		} else if execFunc == nil {
			return nil, fmt.Errorf("failed to register action: exec function is required")
		}

		name := util.ToPascalCase(string(displayName))
		inputSchema, err := NewTypeSchemaFor[In](mux.Types(), fmt.Sprintf("ActionInput.%s", name))
		if err != nil {
			return nil, err
		}

		outputSchema, err := NewTypeSchemaFor[Out](mux.Types(), fmt.Sprintf("ActionOutput.%s", name))
		if err != nil {
			return nil, err
		}

		return &actionWrapper[In, Out]{
			inputSchema:  inputSchema,
			outputSchema: outputSchema,
			execFunc: func(ctx context.Context, input In) (Out, error) {
				return execFunc(ctx, mux, input)
			},

			action: &actionv1beta1.Action{
				Name:        fmt.Sprintf("%s/%s", ActionsPrefix, util.Slugify(string(displayName))),
				DisplayName: string(displayName),
				Description: description,
				Input:       inputSchema.ToProto(),
				Output:      outputSchema.ToProto(),
			},
		}, nil
	}
}

func newActionRun(req *actionv1beta1.ExecuteActionRequest) *actionRun {
	return &actionRun{
		req:   req,
		id:    uuid.New(),
		start: time.Now(),
	}
}

func (r *actionRun) GetRunId() string {
	if r.id == uuid.Nil {
		r.id = uuid.New()
	}
	return r.id.String()
}

func (r *actionRun) String() string {
	return fmt.Sprintf("%s (id=%s) finished in %s", r.req.GetName(), r.GetRunId(), r.done.Sub(r.start))
}

func (a *ActionRegistry) Find(name string) (RegisteredAction, error) {
	if strings.HasPrefix(name, ActionsPrefix+"/") {
		action, ok := a.m.Load(name)
		if ok {
			return action, nil
		}
	} else {
		_, action, found := a.m.FindFunc(func(_ string, action RegisteredAction) bool {
			return name == action.Proto().GetDisplayName()
		})
		if found {
			return action, nil
		}
	}

	return nil, fmt.Errorf("action not found: %q", name)
}

func (a *actionWrapper[In, Out]) Execute(ctx context.Context, req *actionv1beta1.ExecuteActionRequest) (*actionv1beta1.ExecuteActionResponse, error) {
	var cancel context.CancelFunc
	if req.GetTimeout().GetSeconds() > 0 {
		ctx, cancel = context.WithTimeout(ctx, req.GetTimeout().AsDuration())
		defer cancel()
	}

	input, err := a.resolveInput(req)
	if err != nil {
		return nil, err
	}

	run := newActionRun(req)
	output, err := a.execFunc(ctx, input)
	if err != nil {
		return nil, err
	}
	run.done = time.Now()

	return a.resolveOutput(run, output)
}

func (a *ActionRegistry) Execute(ctx context.Context, req *actionv1beta1.ExecuteActionRequest) (*actionv1beta1.ExecuteActionResponse, error) {
	action, err := a.Find(req.GetName())
	if err != nil {
		return nil, err
	}
	return action.Execute(ctx, req)
}

func (a *ActionRegistry) Protos() *actionv1beta1.ListActionsResponse {
	return &actionv1beta1.ListActionsResponse{
		Actions: util.SliceMap(a.m.Values(), func(a RegisteredAction) *actionv1beta1.Action {
			return a.Proto()
		}),
	}
}

func (a *ActionRegistry) List(ctx context.Context, req *actionv1beta1.ListActionsRequest) (*actionv1beta1.ListActionsResponse, error) {
	return &actionv1beta1.ListActionsResponse{
		Actions: a.Protos().GetActions(),
	}, nil
}

func (a *ActionRegistry) Get(ctx context.Context, req *actionv1beta1.GetActionRequest) (*actionv1beta1.GetActionResponse, error) {
	action, ok := a.m.Load(req.GetName())
	if ok {
		return &actionv1beta1.GetActionResponse{
			Action: action.Proto(),
		}, nil
	}
	return nil, status.Errorf(codes.NotFound, "action not found: %s", req.Name)
}

func (a *actionWrapper[In, Out]) Proto() *actionv1beta1.Action {
	return proto.CloneOf(a.action)
}

func (a *actionWrapper[In, Out]) resolveInput(req *actionv1beta1.ExecuteActionRequest) (_ In, err error) {
	if a.inputSchema.IsEmpty() {
		return
	} else if req.GetInput() == nil {
		err = fmt.Errorf("action %q input required", a.action.GetName())
		return
	}

	input, err := req.GetInput().UnmarshalNew()
	if err != nil {
		err = fmt.Errorf("action %q input unmarshal error: %w", a.action.GetName(), err)
		return
	}

	return a.inputSchema.ValidateAny(input)
}

func (a *actionWrapper[In, Out]) resolveOutput(run *actionRun, output Out) (*actionv1beta1.ExecuteActionResponse, error) {
	outputAny, err := common.WrapProtoAny(output)
	if err != nil {
		return nil, err
	}

	return &actionv1beta1.ExecuteActionResponse{
		Name:    a.Proto().GetName(),
		RunId:   run.GetRunId(),
		Message: run.String(),
		Output:  outputAny,
	}, nil
}
