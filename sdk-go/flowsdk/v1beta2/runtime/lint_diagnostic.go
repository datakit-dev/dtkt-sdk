package runtime

import (
	"errors"
	"strings"
)

// Severity represents the severity level of a lint diagnostic.
type Severity int

const (
	// SeverityError indicates a hard failure that prevents the flow from running.
	SeverityError Severity = iota
	// SeverityWarning indicates a potential issue that does not prevent execution.
	SeverityWarning
)

// Lint diagnostic codes.
const (
	CodeInvalidCEL               = "invalid-cel"
	CodeConstantDefaultExclusive = "constant-default-exclusive"
	CodeOrphanedNode             = "orphaned-node"
	CodeNoUpstream               = "no-upstream"
	CodeUndeclaredConnection     = "undeclared-connection"
	CodeMissingField             = "missing-field"
	CodeUnknownField             = "unknown-field"
	CodeTypeMismatch             = "type-mismatch"
	CodeSchemaError              = "schema-error"
	CodeProtoConflict            = "proto-conflict"
)

// LintDiagnostic represents a single lint finding with structured metadata.
// Node contains the graph node ID (e.g. "vars.doubled", "actions.run") and
// Path contains the field path within the node (e.g. "value", "call.connection",
// "request.name"). Together they form a format-agnostic location that consumers
// (CLI, VS Code extension) can resolve to source coordinates.
type LintDiagnostic struct {
	Severity Severity
	Node     string
	Path     string
	Message  string
	Code     string
}

// Error formats the diagnostic as a human-readable string.
func (d *LintDiagnostic) Error() string {
	var b strings.Builder
	b.WriteString("node ")
	b.WriteString(d.Node)
	if d.Severity == SeverityWarning {
		b.WriteString(": warning")
	}
	if d.Path != "" {
		b.WriteString(": ")
		b.WriteString(d.Path)
	}
	b.WriteString(": ")
	b.WriteString(d.Message)
	return b.String()
}

// LintResult collects all diagnostics from a lint pass. It implements the error
// interface so callers that need an error (e.g. cobra RunE) can return it directly.
type LintResult struct {
	Diagnostics []LintDiagnostic
}

// Error joins all diagnostic messages with newlines.
func (r *LintResult) Error() string {
	if len(r.Diagnostics) == 0 {
		return ""
	}
	errs := make([]error, len(r.Diagnostics))
	for i := range r.Diagnostics {
		errs[i] = &r.Diagnostics[i]
	}
	return errors.Join(errs...).Error()
}

// HasErrors returns true if any diagnostic has error severity.
func (r *LintResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only error-severity diagnostics as a joined error, or nil.
func (r *LintResult) Errors() error {
	var errs []error
	for i := range r.Diagnostics {
		if r.Diagnostics[i].Severity == SeverityError {
			errs = append(errs, &r.Diagnostics[i])
		}
	}
	return errors.Join(errs...)
}

// Warnings returns only warning-severity diagnostics.
func (r *LintResult) Warnings() []LintDiagnostic {
	var out []LintDiagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityWarning {
			out = append(out, d)
		}
	}
	return out
}
