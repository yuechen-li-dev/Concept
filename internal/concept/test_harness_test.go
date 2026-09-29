package concept

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// A native determinism loop reprobes the same installed toolchain 100 times.
// On Windows, LookPath otherwise scans every PATHEXT spelling in every PATH
// directory for each probe. Resolve the actual tools once and keep those same
// installations visible throughout the loop; probe execution is unchanged.
func narrowNativeDeterminismPath(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	seen := map[string]bool{}
	var dirs []string
	for _, tool := range []string{"clang++", "clang", "llvm-ar", "gcc", "g++", "ar"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		dir := filepath.Dir(path)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
		dirs = append(dirs, filepath.Join(systemRoot, "System32"), systemRoot)
	}
	if len(dirs) == 0 {
		t.Fatal("native determinism fixture has no resolved toolchain")
	}
	t.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
	t.Setenv("PATHEXT", ".COM;.EXE")
}

// nativeHostLinkArgs are the host libraries a linked native harness may need.
// Generated C can call <math.h> functions (inference lowers softmax to expf).
// Unix toolchains use -lm; an MSVC-target clang on Windows interprets -lm as
// a request for m.lib, which is not supplied by the Windows SDK. Windows C
// runtime math symbols are resolved by the ordinary host link instead.
// Threaded harnesses use POSIX threads off Windows.
func nativeHostLinkArgs() []string {
	return nativeHostLinkArgsForOS(runtime.GOOS)
}

func nativeHostLinkArgsForOS(goos string) []string {
	if goos == "windows" {
		return nil
	}
	return []string{"-lm", "-pthread"}
}

func TestNativeHostLinkArgsForOS(t *testing.T) {
	for _, tc := range []struct {
		goos string
		want string
	}{
		{"windows", ""},
		{"linux", "-lm -pthread"},
		{"darwin", "-lm -pthread"},
	} {
		if got := strings.Join(nativeHostLinkArgsForOS(tc.goos), " "); got != tc.want {
			t.Errorf("%s host link args = %q, want %q", tc.goos, got, tc.want)
		}
	}
}

func TestWindowsNativeMathLinksWithoutMlib(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows C runtime link probe")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "math.c")
	const code = "#include <math.h>\nvolatile float x = 1.0f;\nint main(void) { return expf(x) > 2.0f ? 0 : 1; }\n"
	if err := os.WriteFile(source, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"clang", "gcc"} {
		compiler, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			executable := filepath.Join(dir, name+"-math.exe")
			if output, err := nativeCommand(t, compiler, withHostLinkArgs("-std=c11", source, "-o", executable)...).CombinedOutput(); err != nil {
				t.Fatalf("link math without m.lib: %v\n%s", err, output)
			}
			if output, err := nativeCommand(t, executable).CombinedOutput(); err != nil {
				t.Fatalf("run math probe: %v\n%s", err, output)
			}
		})
	}
}

// withHostLinkArgs inserts nativeHostLinkArgs after the inputs of a gcc/clang
// link command (before a trailing "-o <output>" when present), since libraries
// must follow the objects that reference them.
func withHostLinkArgs(args ...string) []string {
	out := make([]string, 0, len(args)+3)
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" && i+2 == len(args) {
			out = append(out, nativeHostLinkArgs()...)
			return append(out, args[i:]...)
		}
		out = append(out, args[i])
	}
	return append(out, nativeHostLinkArgs()...)
}

// nativeThreadShim is prepended to threaded native harnesses so the same
// specimen runs on Win32 threads and POSIX threads. It must precede every
// other include so the POSIX feature macro applies. POSIX has no timed join;
// hung joins are bounded by nativeCommand's deadline instead.
const nativeThreadShim = `#if !defined(_WIN32) && !defined(_POSIX_C_SOURCE)
#define _POSIX_C_SOURCE 200809L
#endif
#if defined(_WIN32)
#include <windows.h>
typedef HANDLE cpt_thread;
#define CPT_THREAD_FN(name, arg) static DWORD WINAPI name(LPVOID arg)
static int cpt_thread_start(cpt_thread* thread, LPTHREAD_START_ROUTINE fn, void* arg) {
  *thread = CreateThread(NULL, 0, fn, arg, 0, NULL);
  return *thread == NULL ? -1 : 0;
}
static int cpt_thread_join(cpt_thread thread, unsigned timeout_ms) {
  DWORD waited = WaitForSingleObject(thread, timeout_ms == 0 ? INFINITE : (DWORD)timeout_ms);
  CloseHandle(thread);
  return waited == WAIT_OBJECT_0 ? 0 : -1;
}
static void cpt_thread_yield(void) { Sleep(0); }
static double cpt_seconds_now(void) {
  LARGE_INTEGER frequency, now;
  QueryPerformanceFrequency(&frequency);
  QueryPerformanceCounter(&now);
  return (double)now.QuadPart / (double)frequency.QuadPart;
}
#else
#include <pthread.h>
#include <sched.h>
#include <time.h>
typedef pthread_t cpt_thread;
#define CPT_THREAD_FN(name, arg) static void* name(void* arg)
static int cpt_thread_start(cpt_thread* thread, void* (*fn)(void*), void* arg) {
  return pthread_create(thread, NULL, fn, arg) == 0 ? 0 : -1;
}
static int cpt_thread_join(cpt_thread thread, unsigned timeout_ms) {
  (void)timeout_ms;
  return pthread_join(thread, NULL) == 0 ? 0 : -1;
}
static void cpt_thread_yield(void) { sched_yield(); }
static double cpt_seconds_now(void) {
  struct timespec now;
  clock_gettime(CLOCK_MONOTONIC, &now);
  return (double)now.tv_sec + (double)now.tv_nsec / 1e9;
}
#endif
`

// requireCFloat16 skips tests for the _Float16 extension lane when the host
// C compiler lacks it (for example GCC < 12 on x86-64), which is a toolchain
// capability limit rather than a Concept defect.
// An empty compiler name selects gcc, then clang, like runFoundationNativeHarness.
func requireCFloat16(t *testing.T, compilerName string) {
	t.Helper()
	var compiler string
	var err error
	if compilerName != "" {
		compiler, err = exec.LookPath(compilerName)
	} else if compiler, err = exec.LookPath("gcc"); err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Skipf("C compiler unavailable for the _Float16 lane: %v", err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "float16_probe.c")
	if err := os.WriteFile(probe, []byte("_Float16 cpt_probe(_Float16 value) { return value; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, compiler, "-std=c11", "-fsyntax-only", probe).CombinedOutput(); err != nil {
		t.Skipf("%s does not support _Float16 on this target: %s", filepath.Base(compiler), strings.TrimSpace(string(out)))
	}
}
