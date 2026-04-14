package runtime

import (
	"context"
	"time"

	expr "cel.dev/expr"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// minInputThrottle is the built-in minimum throttle applied to inputs that
// have a cache or type-level default but no explicit throttle. Without a
// throttle the input handler blocks indefinitely and the default/cache
// fallback never fires.
const minInputThrottle = 10 * time.Millisecond

// inputHandler implements the input resolution chain: try fresh value within
// the throttle window, then fall back to cached value, then type default, then
// block. For inputs without throttle/cache/default it is not used -- the demux
// publishes directly to the input's topic.
type inputHandler struct {
	id         string
	raw        <-chan *expr.Value // values fed by the demux goroutine
	publish    func(val *expr.Value, eof bool) error
	throttle   time.Duration
	cache      bool
	defaultVal *expr.Value
}

func (h *inputHandler) Run(ctx context.Context) error {
	var cached *expr.Value

	for {
		val, eof := h.resolve(ctx, cached)
		if eof {
			return h.publish(newEOFValue(), true)
		}
		if h.cache {
			cached = val
		}
		if err := h.publish(val, false); err != nil {
			return err
		}
	}
}

func (h *inputHandler) resolve(ctx context.Context, cached *expr.Value) (val *expr.Value, eof bool) {
	if h.throttle > 0 {
		select {
		case <-ctx.Done():
			return nil, true
		case v, ok := <-h.raw:
			if !ok {
				return nil, true
			}
			return v, false
		case <-time.After(h.throttle):
			if cached != nil {
				return cached, false
			}
			if h.defaultVal != nil {
				return h.defaultVal, false
			}
			// No fallback available: block until a value arrives.
			select {
			case <-ctx.Done():
				return nil, true
			case v, ok := <-h.raw:
				if !ok {
					return nil, true
				}
				return v, false
			}
		}
	}

	// No throttle: block for value.
	select {
	case <-ctx.Done():
		return nil, true
	case v, ok := <-h.raw:
		if !ok {
			return nil, true
		}
		return v, false
	}
}

// inputTypeDefault extracts the default value from an Input's type-level
// default field. Returns nil if the type has no default set.
func inputTypeDefault(inp *flowv1beta2.Input) *expr.Value {
	switch inp.WhichType() {
	case flowv1beta2.Input_Bool_case:
		b := inp.GetBool()
		if b.HasDefault() {
			return &expr.Value{Kind: &expr.Value_BoolValue{BoolValue: b.GetDefault()}}
		}
	case flowv1beta2.Input_Bytes_case:
		bt := inp.GetBytes()
		if bt.HasDefault() {
			return &expr.Value{Kind: &expr.Value_BytesValue{BytesValue: bt.GetDefault()}}
		}
	case flowv1beta2.Input_Double_case:
		d := inp.GetDouble()
		if d.HasDefault() {
			return &expr.Value{Kind: &expr.Value_DoubleValue{DoubleValue: d.GetDefault()}}
		}
	case flowv1beta2.Input_Float_case:
		f := inp.GetFloat()
		if f.HasDefault() {
			return &expr.Value{Kind: &expr.Value_DoubleValue{DoubleValue: float64(f.GetDefault())}}
		}
	case flowv1beta2.Input_Int64_case:
		i := inp.GetInt64()
		if i.HasDefault() {
			return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: i.GetDefault()}}
		}
	case flowv1beta2.Input_Uint64_case:
		u := inp.GetUint64()
		if u.HasDefault() {
			return &expr.Value{Kind: &expr.Value_Uint64Value{Uint64Value: u.GetDefault()}}
		}
	case flowv1beta2.Input_Int32_case:
		i := inp.GetInt32()
		if i.HasDefault() {
			return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: int64(i.GetDefault())}}
		}
	case flowv1beta2.Input_Uint32_case:
		u := inp.GetUint32()
		if u.HasDefault() {
			return &expr.Value{Kind: &expr.Value_Uint64Value{Uint64Value: uint64(u.GetDefault())}}
		}
	case flowv1beta2.Input_String__case:
		s := inp.GetString()
		if s.HasDefault() {
			return &expr.Value{Kind: &expr.Value_StringValue{StringValue: s.GetDefault()}}
		}
	}
	return nil
}
