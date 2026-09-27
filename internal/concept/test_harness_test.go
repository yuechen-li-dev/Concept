package concept

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// defaultNativeCommandTimeout bounds every compiler invocation and generated
// executable a test launches. A deadlocked native binary (for example in the
// native-thread race specimens) then fails its own test with NATIVE_TIMEOUT
// instead of stalling the whole package until `go test -timeout` panics.
// Override with CONCEPT_TEST_NATIVE_TIMEOUT (a Go duration, e.g. "5m").
const defaultNativeCommandTimeout = 2 * time.Minute

// nativeTimeoutGrace is reserved before the test binary's own deadline so a
// hung child is killed and reported by the owning test first.
const nativeTimeoutGrace = 15 * time.Second

func nativeCommandTimeout(t testing.TB) time.Duration {
	timeout := defaultNativeCommandTimeout
	if raw := strings.TrimSpace(os.Getenv("CONCEPT_TEST_NATIVE_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			t.Fatalf("CONCEPT_TEST_NATIVE_TIMEOUT=%q is not a positive duration", raw)
		}
		timeout = parsed
	}
	if tt, ok := t.(interface{ Deadline() (time.Time, bool) }); ok {
		if deadline, ok := tt.Deadline(); ok {
			if remaining := time.Until(deadline) - nativeTimeoutGrace; remaining > 0 && remaining < timeout {
				timeout = remaining
			}
		}
	}
	return timeout
}

// nativeCommand is exec.Command with a per-command deadline tied to the test.
// On expiry the process is killed and the test log records which command hung.
func nativeCommand(t testing.TB, name string, args ...string) *exec.Cmd {
	t.Helper()
	timeout := nativeCommandTimeout(t)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error {
		if ctx.Err() == context.DeadlineExceeded {
			t.Logf("NATIVE_TIMEOUT: killed after %s: %s %s", timeout, name, strings.Join(args, " "))
		}
		return cmd.Process.Kill()
	}
	// Do not wait forever on pipes held open by grandchildren after a kill.
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

// determinismRuns is the repetition count for artifact/plan/output
// determinism gates. Full runs keep the 100-run gate; `go test -short`
// keeps enough repetitions to catch map-order drift in the inner loop.
func determinismRuns() int {
	if testing.Short() {
		return 10
	}
	return 100
}
