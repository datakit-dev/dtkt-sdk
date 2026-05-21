package runtime

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	expr "cel.dev/expr"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/cel-go/cel"
	"google.golang.org/genproto/googleapis/rpc/status"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/flowsdk/shared"
	flowv1beta2 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/flow/v1beta2"
)

// compiledRetryStrategy holds pre-compiled CEL programs for each
// RetryStrategy expression plus the backoff configuration.
type compiledRetryStrategy struct {
	whenProg      cel.Program // nil = activate on every error
	backoff       *flowv1beta2.Backoff
	skipProg      cel.Program // nil = never skip
	suspendProg   cel.Program // nil = never suspend
	terminateProg cel.Program // nil = never terminate
	continueProg  cel.Program // nil = never continue (emit error as value)
}

// compileRetryStrategy compiles a RetryStrategy proto into executable programs.
// Returns nil if rs is nil.
func compileRetryStrategy(env shared.Env, rs *flowv1beta2.RetryStrategy) (*compiledRetryStrategy, error) {
	if rs == nil {
		return nil, nil
	}
	c := &compiledRetryStrategy{backoff: rs.GetBackoff()}
	var err error
	if w := rs.GetWhen(); w != "" {
		c.whenProg, err = compileCEL(env, w)
		if err != nil {
			return nil, fmt.Errorf("retry_strategy.when: %w", err)
		}
	}
	if s := rs.GetSkipWhen(); s != "" {
		c.skipProg, err = compileCEL(env, s)
		if err != nil {
			return nil, fmt.Errorf("retry_strategy.skip_when: %w", err)
		}
	}
	if s := rs.GetSuspendWhen(); s != "" {
		c.suspendProg, err = compileCEL(env, s)
		if err != nil {
			return nil, fmt.Errorf("retry_strategy.suspend_when: %w", err)
		}
	}
	if t := rs.GetTerminateWhen(); t != "" {
		c.terminateProg, err = compileCEL(env, t)
		if err != nil {
			return nil, fmt.Errorf("retry_strategy.terminate_when: %w", err)
		}
	}
	if cw := rs.GetContinueWhen(); cw != "" {
		c.continueProg, err = compileCEL(env, cw)
		if err != nil {
			return nil, fmt.Errorf("retry_strategy.continue_when: %w", err)
		}
	}
	return c, nil
}

// retryOutcome is the result of a single retry-strategy evaluation.
type retryOutcome int

const (
	retryOutcomeRetry     retryOutcome = iota // backoff and retry
	retryOutcomeSkip                          // skip this item, continue processing
	retryOutcomeSuspend                       // suspend the node
	retryOutcomeTerminate                     // terminate the flow
	retryOutcomeContinue                      // emit error-derived value as output
	retryOutcomeFail                          // no strategy matched -- propagate error
)

// retryableCall wraps an RPC invocation that may be retried.
// It is invoked repeatedly by executeWithRetry until it succeeds or the
// retry strategy decides to stop.
type retryableCall func(ctx context.Context) error

// errSkipped is a sentinel returned by executeWithRetry when skip_when matches.
// Handlers should check for this and continue to the next input.
var errSkipped = errors.New("retry: skipped")

// executeWithRetry runs fn, applying the retry strategy on errors.
// If retry is nil, fn is called once and errors propagate immediately.
// Returns errSkipped if skip_when matches (caller should continue to next item).
//
// The vars map provides the current CEL activation (globals). The strategy's
// CEL expressions receive an additional "this" binding with:
//   - this.error:    *google.rpc.Status proto (nil when no error)
//   - this.response: always nil for now (future: last successful response)
func executeWithRetry(ctx context.Context, retry *compiledRetryStrategy, vars map[string]any, fn retryableCall) error {
	if retry == nil {
		return fn(ctx)
	}

	// Backoff engine. NextBackOff returns Stop once the attempt budget
	// (max_attempts; 0/unset = run once) is spent. Delays + cap + the
	// optional jitter all come from the proto via newRetryBackOff.
	bo := newRetryBackOff(retry.backoff)

	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		// Convert the Go error to a google.rpc.Status proto for CEL evaluation.
		errStatus := grpcStatusProto(err)
		retryVars := buildRetryVars(vars, errStatus)

		// Check the "when" guard -- if it doesn't match, the error is not
		// handled by this strategy. Propagate immediately.
		if retry.whenProg != nil {
			result, evalErr := evalCEL(retry.whenProg, retryVars)
			if evalErr != nil {
				return fmt.Errorf("retry_strategy.when eval: %w", evalErr)
			}
			if result.Value() != true {
				return err
			}
		}

		// Evaluate escalation paths (continue/skip/suspend/terminate) on every error.
		outcome, continueVal := evaluateEscalation(retry, retryVars)
		switch outcome {
		case retryOutcomeContinue:
			return &ContinueError{Value: continueVal}
		case retryOutcomeSkip:
			return errSkipped
		case retryOutcomeSuspend:
			return &SuspendError{Status: errStatus}
		case retryOutcomeTerminate:
			return &TerminateError{Status: errStatus, Cause: err}
		}

		// Attempt backoff retry. Stop => the attempt budget is spent.
		delay := bo.NextBackOff()
		if delay == backoff.Stop {
			return err // retries exhausted
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// evaluateEscalation checks continue_when, skip_when, suspend_when,
// terminate_when in order. Returns retryOutcomeRetry if none match.
// For retryOutcomeContinue, the second return value carries the result as *expr.Value.
func evaluateEscalation(retry *compiledRetryStrategy, vars map[string]any) (retryOutcome, *expr.Value) {
	if retry.continueProg != nil {
		result, err := evalCEL(retry.continueProg, vars)
		if err == nil && result.Value() != nil && result.Value() != false {
			val, convErr := refValToExpr(result)
			if convErr == nil {
				return retryOutcomeContinue, val
			}
		}
	}
	if retry.skipProg != nil {
		result, err := evalCEL(retry.skipProg, vars)
		if err == nil && result.Value() == true {
			return retryOutcomeSkip, nil
		}
	}
	if retry.suspendProg != nil {
		result, err := evalCEL(retry.suspendProg, vars)
		if err == nil && result.Value() == true {
			return retryOutcomeSuspend, nil
		}
	}
	if retry.terminateProg != nil {
		result, err := evalCEL(retry.terminateProg, vars)
		if err == nil && result.Value() == true {
			return retryOutcomeTerminate, nil
		}
	}
	return retryOutcomeRetry, nil
}

// SuspendError signals that a node should be suspended.
type SuspendError struct {
	Status *status.Status
}

func (e *SuspendError) Error() string {
	if e.Status != nil {
		return fmt.Sprintf("node suspended: %s", e.Status.GetMessage())
	}
	return "node suspended"
}

// ContinueError signals that the error should be converted to an output value.
// The Value field carries the result as *expr.Value from continue_when.
type ContinueError struct {
	Value *expr.Value
}

func (e *ContinueError) Error() string {
	return fmt.Sprintf("retry: continue with value: %v", e.Value)
}

// TerminateError signals that the entire flow should be terminated.
type TerminateError struct {
	Status *status.Status
	Cause  error
}

func (e *TerminateError) Error() string {
	if e.Status != nil {
		return fmt.Sprintf("flow terminated: %s", e.Status.GetMessage())
	}
	return "flow terminated"
}

func (e *TerminateError) Unwrap() error {
	return e.Cause
}

// buildRetryVars augments the CEL activation with retry-specific "this" context.
func buildRetryVars(base map[string]any, errStatus *status.Status) map[string]any {
	retryVars := make(map[string]any, len(base)+1)
	for k, v := range base {
		retryVars[k] = v
	}
	// Convert *status.Status proto to a map for consistent CEL access (same
	// pattern as nodeToMap). Users write `this.error.code`, `this.error.message`.
	var errMap map[string]any
	if errStatus != nil {
		errMap = map[string]any{
			"code":    int64(errStatus.GetCode()),
			"message": errStatus.GetMessage(),
		}
	}
	retryVars["this"] = map[string]any{
		"error":    errMap,
		"response": nil,
	}
	return retryVars
}

// grpcStatusProto extracts a google.rpc.Status proto from a Go error.
// Returns a generic UNKNOWN status if the error is not a gRPC status error.
func grpcStatusProto(err error) *status.Status {
	s, ok := grpcstatus.FromError(err)
	if !ok {
		// Not a gRPC error -- wrap as UNKNOWN.
		s = grpcstatus.New(2, err.Error()) // codes.Unknown = 2
	}
	return s.Proto()
}

// newRetryBackOff builds the backoff used to pace retries. The attempt
// budget is enforced by WithMaxRetries: flow's max_attempts is the
// *total* attempts (0/unset = 1, run once), while cenkalti counts
// retries *after* the first, so the cap is max_attempts-1. Once that
// many NextBackOff calls have been made, NextBackOff returns Stop and
// executeWithRetry propagates the last error.
func newRetryBackOff(b *flowv1beta2.Backoff) backoff.BackOff {
	maxAttempts := uint32(1)
	if b != nil && b.GetMaxAttempts() > 0 {
		maxAttempts = b.GetMaxAttempts()
	}
	return backoff.WithMaxRetries(newExponential(b), uint64(maxAttempts-1))
}

// newExponential builds the exponential schedule from the proto,
// reproducing the prior hand-rolled backoffDelay exactly when jitter is
// unset: with RandomizationFactor 0, cenkalti returns currentInterval
// verbatim, so the sequence is initial, initial*mult, ... capped at
// max_backoff - the same values the old math.Pow form produced. The
// deviations from cenkalti's own defaults are deliberate:
//   - RandomizationFactor defaults to 0 here (cenkalti's default is 0.5);
//     jitter is opt-in via the proto field, keeping existing flows
//     deterministic.
//   - Multiplier falls back to 2 (not cenkalti's 1.5) to match the
//     prior default-when-<1 behavior.
//   - MaxInterval is left effectively unbounded when max_backoff is
//     unset (cenkalti's default caps at 60s, which the old code did not).
//   - An invalid/unset initial_backoff yields a 0 interval (all delays
//     0), matching the old backoffDelay(nil-ish) == 0.
func newExponential(b *flowv1beta2.Backoff) *backoff.ExponentialBackOff {
	e := backoff.NewExponentialBackOff()
	e.MaxElapsedTime = 0      // bounded by max_attempts, never by wall time
	e.RandomizationFactor = 0 // deterministic unless jitter is set below
	if b != nil {
		if init := b.GetInitialBackoff(); init.IsValid() {
			e.InitialInterval = init.AsDuration()
		} else {
			e.InitialInterval = 0
		}
		if mult := b.GetBackoffMultiplier(); mult >= 1 {
			e.Multiplier = mult
		} else {
			e.Multiplier = 2
		}
		if mb := b.GetMaxBackoff(); mb.IsValid() {
			e.MaxInterval = mb.AsDuration()
		} else {
			e.MaxInterval = math.MaxInt64
		}
		e.RandomizationFactor = b.GetJitter()
	}
	e.Reset() // start currentInterval at InitialInterval
	return e
}

// lintRetryStrategy validates CEL expressions in a RetryStrategy without
// producing executable programs. Returns structured diagnostics.
func lintRetryStrategy(rs *flowv1beta2.RetryStrategy) []LintDiagnostic {
	if rs == nil {
		return nil
	}
	var diags []LintDiagnostic
	celDiag := func(path string, err error) {
		diags = append(diags, LintDiagnostic{
			Severity: SeverityError,
			Path:     path,
			Message:  err.Error(),
			Code:     CodeInvalidCEL,
		})
	}
	if w := rs.GetWhen(); w != "" {
		if _, err := parseCEL(w); err != nil {
			celDiag("retry_strategy.when", err)
		}
	}
	if s := rs.GetSkipWhen(); s != "" {
		if _, err := parseCEL(s); err != nil {
			celDiag("retry_strategy.skip_when", err)
		}
	}
	if s := rs.GetSuspendWhen(); s != "" {
		if _, err := parseCEL(s); err != nil {
			celDiag("retry_strategy.suspend_when", err)
		}
	}
	if t := rs.GetTerminateWhen(); t != "" {
		if _, err := parseCEL(t); err != nil {
			celDiag("retry_strategy.terminate_when", err)
		}
	}
	if cw := rs.GetContinueWhen(); cw != "" {
		if _, err := parseCEL(cw); err != nil {
			celDiag("retry_strategy.continue_when", err)
		}
	}
	return diags
}
