//go:build leakcheck

package runtime

// leakCheckEnabled is true when the test binary is built with
// `-tags=leakcheck`. In that mode, withAndWithoutOutbox runs subtests
// serially and verifies the process-wide goroutine count returns to
// baseline after each subtest. Use for periodic / pre-merge audits;
// normal day-to-day runs use the default (parallel, no leak check).
const leakCheckEnabled = true
