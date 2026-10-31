package concept

import (
	"encoding/json"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// Opt-in, run-local observation only. Events never participate in build keys,
// admission, planning, output selection or test expectations.
var qualificationProfile struct {
	sync.Mutex
	once sync.Once
	file *os.File
}

func qualificationEvent(t testing.TB, kind string, start time.Time, fields map[string]any) {
	qualificationProfile.once.Do(func() {
		if path := os.Getenv("CONCEPT_QUALIFICATION_PROFILE"); path != "" {
			var err error
			qualificationProfile.file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
		}
	})
	if qualificationProfile.file == nil {
		return
	}
	event := map[string]any{"time": time.Now().UTC().Format(time.RFC3339Nano), "test": t.Name(), "kind": kind, "seconds": time.Since(start).Seconds(), "fields": fields}
	qualificationProfile.Lock()
	defer qualificationProfile.Unlock()
	if err := json.NewEncoder(qualificationProfile.file).Encode(event); err != nil {
		panic(err)
	}
}
func qualificationCommand(t testing.TB, cmd *exec.Cmd) {
	if os.Getenv("CONCEPT_QUALIFICATION_PROFILE") == "" {
		return
	}
	start := time.Now()
	qualificationEvent(t, "command-created", start, map[string]any{"path": cmd.Path, "args": cmd.Args})
	t.Cleanup(func() {
		fields := map[string]any{"path": cmd.Path, "args": cmd.Args, "started": cmd.Process != nil}
		if cmd.ProcessState != nil {
			fields["exit"] = cmd.ProcessState.ExitCode()
			fields["user_seconds"] = cmd.ProcessState.UserTime().Seconds()
			fields["system_seconds"] = cmd.ProcessState.SystemTime().Seconds()
		}
		qualificationEvent(t, "command-cleanup", start, fields)
	})
}
