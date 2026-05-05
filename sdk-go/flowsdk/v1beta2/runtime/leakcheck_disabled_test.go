//go:build !leakcheck

package runtime

// leakCheckEnabled is false by default: tests run in parallel via t.Parallel()
// for speed. Goroutine leak detection requires serial execution and is enabled
// by building with `-tags=leakcheck` (see leakcheck_enabled_test.go).
const leakCheckEnabled = false
