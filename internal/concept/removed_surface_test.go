package concept

import (
	"errors"
	"testing"
)

// Every retired M-era spelling is rejected with a diagnostic naming its
// Core replacement.
func TestRemovedSurfacesNameTheirReplacement(t *testing.T) {
	cases := []struct{ name, source, code string }{
		{"signal automata", `profile Core;
enum S { Go, }
automata A(S) { machine M { state X { } } }`, "REMOVED_SIGNAL_AUTOMATA"},
		{"initial machine", `profile Core;
automata A { initial machine M { state X { } } }`, "REMOVED_INITIAL"},
		{"initial state", `profile Core;
automata A { machine M { initial state X { } } }`, "REMOVED_INITIAL"},
		{"finish", `profile Core;
automata A { machine M { state X { finish; } } }`, "REMOVED_FINISH"},
		{"on goto", `profile Core;
enum S { Go, }
automata A with input S { machine M { state X { on S::Go goto X; } } }`, "REMOVED_ON_GOTO"},
		{"on => goto", `profile Core;
enum S { Go, }
automata A with input S { machine M { state X { on S::Go => goto X; } } }`, "REMOVED_ON_GOTO"},
		{"dispatch", `profile Core;
int F() { return dispatch(a, b); }`, "REMOVED_DISPATCH"},
		{"effect", `profile Vulkan;
effect Record(int id);`, "REMOVED_EFFECT"},
		{"effects batch", `profile Core;
void F() { effects A batch; }`, "REMOVED_EFFECT"},
		{"emit", `profile Core;
void F() { emit Record(1); }`, "REMOVED_EFFECT"},
		{"actuator", `profile Vulkan;
actuator Run(A, borrow M m, E) { }`, "REMOVED_ACTUATOR"},
		{"actuation", `profile Core;
void F() { actuation R r = actuate(b, e); }`, "REMOVED_ACTUATOR"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			module, err := Parse("removed.concept", tc.source)
			if err == nil {
				_, err = Generate(module, []byte(tc.source))
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("diagnostic = %v, want %s", err, tc.code)
			}
		})
	}
}
