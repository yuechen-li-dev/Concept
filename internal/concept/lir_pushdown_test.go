package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pushdownFixture(t *testing.T) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt2", "machines", "pushdown.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	return module, source
}

// The C Step API is void. Its existing yield marker observes the boundary,
// while completed/depth/state/fields are read from the actual C instance.
// This instrumentation does not replace or emulate any machine operation.
func TestEVT2x6COracleBoundaries(t *testing.T) {
	for _, rootPop := range []bool{false, true} {
		name := "root-complete"
		if rootPop {
			name = "root-pop"
		}
		t.Run(name, func(t *testing.T) { runPushdownCOracle(t, rootPop) })
	}
}

func runPushdownCOracle(t *testing.T, rootPop bool) {
	_, source := pushdownFixture(t)
	if rootPop {
		source = []byte(strings.ReplaceAll(string(source), "machine.preserved; complete;", "machine.preserved; pop;"))
	}
	source = append(source, []byte(`
int Main() { instance Worker a(0); Step(a, Parent); return 0; }
`)...)
	module, err := Parse("pushdown.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["pushdown.generated.c"])
	body = "static int oracle_yielded;\n" + strings.ReplaceAll(body, "return; /* CONCEPT_STEP_YIELDED", "oracle_yielded = 1; return; /* CONCEPT_STEP_YIELDED")
	outputs["pushdown.generated.c"] = []byte(body)
	// Include the generated C in the harness so static runtime entry points
	// and typed frames are observed directly, without exporting a test API.
	outputs["oracle.generated.c"] = []byte(`#include "pushdown.generated.c"
#include <stdio.h>
int main(void) {
  concept_worker_instance a = {0};
  concept_worker_init(&a, 0);
  const int depths[] = {2,2,2,3,2,1,2,2,2,3,2,1,0,0};
  const int values[] = {0,0,0,0,10,16,16,16,16,16,26,32,39,39};
  for (int i=0; i<14; ++i) {
    oracle_yielded=0; concept_worker_step_top(&a);
    int top=a.depth ? a.machine_tags[a.depth-1] : -1;
    int state=-1, child=-1;
    if (a.depth) {
      if (top==0) state=a.parent_frames[a.depth-1].current_state;
      if (top==1) { state=a.child_frames[a.depth-1].current_state; child=a.child_frames[a.depth-1].count; }
      if (top==2) state=a.grand_child_frames[a.depth-1].current_state;
    }
    printf("%d %d %u %d %d %d %d %d\n", i, a.completed?2:oracle_yielded?1:0, a.depth,top,state,a.parent_frames[0].preserved,child,a.shared.value);
    if (a.depth!=depths[i] || a.shared.value!=values[i]) return 10+i;
    if ((i==1 || i==7) && (!oracle_yielded || child!=5 || state!=0 || a.parent_frames[0].current_state!=(i==1?1:2))) return 30+i;
    if ((i==0 || i==6) && child!=4) return 50+i;
  }
  return 0;
}
`)
	// runFoundationNativeHarness compiles all generated C; oracle includes it.
	delete(outputs, "pushdown.generated.c")
	outputs["oracle_runtime.inc"] = []byte(body)
	outputs["oracle.generated.c"] = []byte(strings.ReplaceAll(string(outputs["oracle.generated.c"]), "pushdown.generated.c", "oracle_runtime.inc"))
	runFoundationNativeHarness(t, outputs, "entry.c", "int main(void);\n")
}
