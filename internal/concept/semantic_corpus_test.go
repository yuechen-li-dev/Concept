package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type semanticCorpusManifest struct {
	Schema     string `json:"schema"`
	CompilerID string `json:"compiler_id"`
	Totals     struct {
		Valid              int `json:"valid"`
		StaticInvalid      int `json:"static_invalid"`
		RuntimeNegative    int `json:"runtime_negative"`
		Compatibility      int `json:"compatibility"`
		ExpectedDivergence int `json:"expected_divergence"`
	} `json:"totals"`
	Subsystems []struct {
		Subsystem           string   `json:"subsystem"`
		Path                string   `json:"path"`
		Valid               int      `json:"valid"`
		StaticInvalid       int      `json:"static_invalid"`
		RuntimeNegative     []string `json:"runtime_negative"`
		HistoricalMilestone []string `json:"historical_milestones"`
	} `json:"subsystems"`
	CompatibilityFixtures []string `json:"compatibility_fixtures"`
}

func loadSemanticCorpusManifest(t *testing.T) ([]byte, semanticCorpusManifest) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest semanticCorpusManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != "concept-evt1-semantic-corpus-v1" || manifest.CompilerID != CompilerID {
		t.Fatalf("semantic corpus authority mismatch: schema=%q compiler=%q", manifest.Schema, manifest.CompilerID)
	}
	return data, manifest
}

func compileSemanticCorpusFile(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		return err
	}
	_, err = Generate(module, source)
	return err
}

func TestSemanticCorpusManifest(t *testing.T) {
	_, manifest := loadSemanticCorpusManifest(t)
	root := filepath.Join("..", "..", "language", "evt1")
	totals := struct{ valid, staticInvalid, runtimeNegative int }{}
	seen := map[string]string{}

	for _, subsystem := range manifest.Subsystems {
		subsystem := subsystem
		t.Run(subsystem.Subsystem, func(t *testing.T) {
			runtimeNegative := map[string]bool{}
			for _, name := range subsystem.RuntimeNegative {
				runtimeNegative[name] = true
			}
			valid, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(subsystem.Path), "valid", "*.concept"))
			if err != nil {
				t.Fatal(err)
			}
			invalid, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(subsystem.Path), "invalid", "*.concept"))
			if err != nil {
				t.Fatal(err)
			}
			if len(valid) != subsystem.Valid || len(invalid)-len(runtimeNegative) != subsystem.StaticInvalid {
				t.Fatalf("manifest count drift: valid=%d static-invalid=%d runtime-negative=%d", len(valid), len(invalid)-len(runtimeNegative), len(runtimeNegative))
			}
			for _, path := range valid {
				if err := compileSemanticCorpusFile(path); err != nil {
					t.Fatalf("valid corpus rejected: %s: %v", path, err)
				}
				seen[filepath.ToSlash(path)] = subsystem.Subsystem
			}
			for _, path := range invalid {
				name := filepath.Base(path)
				err := compileSemanticCorpusFile(path)
				if runtimeNegative[name] {
					if err != nil {
						t.Fatalf("runtime-negative corpus rejected statically: %s: %v", path, err)
					}
				} else if err == nil {
					t.Fatalf("static-invalid corpus accepted: %s", path)
				}
				seen[filepath.ToSlash(path)] = subsystem.Subsystem
			}
		})
		totals.valid += subsystem.Valid
		totals.staticInvalid += subsystem.StaticInvalid
		totals.runtimeNegative += len(subsystem.RuntimeNegative)
	}

	if totals.valid != manifest.Totals.Valid || totals.staticInvalid != manifest.Totals.StaticInvalid || totals.runtimeNegative != manifest.Totals.RuntimeNegative {
		t.Fatalf("manifest totals drift: got %+v want valid=%d static-invalid=%d runtime-negative=%d", totals, manifest.Totals.Valid, manifest.Totals.StaticInvalid, manifest.Totals.RuntimeNegative)
	}
	all, err := filepath.Glob(filepath.Join(root, "**", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	_ = all // filepath.Glob is not recursive; Walk below is the authority.
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".concept" {
			return nil
		}
		if _, ok := seen[filepath.ToSlash(path)]; !ok {
			return fmt.Errorf("fixture is outside semantic manifest: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSemanticCorpusManifestDeterminism(t *testing.T) {
	data, manifest := loadSemanticCorpusManifest(t)
	baseline, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for iteration := 0; iteration < 100; iteration++ {
		var next semanticCorpusManifest
		if err := json.Unmarshal(data, &next); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(next)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, baseline) {
			t.Fatalf("semantic manifest changed at iteration %d", iteration)
		}
	}
}

func TestCrossFeatureFreezeCompositions(t *testing.T) {
	root := filepath.Join("..", "..", "language", "evt1")
	cases := []struct {
		name  string
		paths []string
	}{
		{"DragonGod-style automata policy", []string{"automata/state/valid/automata_with_state_owned.concept", "machine-stack/valid/machine_multiple_nested_frames.concept", "inference/valid/transition_infer_hardmax.concept", "callable/types/valid/callable_machine_field_owned_capture.concept"}},
		{"Prometheus-style storage and planning", []string{"semantic-facts/valid/fact_preservation_chain.concept", "tensor/semantics/valid/tensor_einstein_matmul.concept", "planner/valid/plan_tensor_contract.concept"}},
		{"Aetheris-style edge policy", []string{"automata/transitions/valid/transition_decide_float_scores.concept"}},
		{"OctetDB-style transaction workflow", []string{"machine-stack/valid/machine_child_failure.concept", "async/basic/valid/async_result_question.concept", "automata/transitions/valid/transition_match_result.concept"}},
		{"async dyn callable", []string{"callable/capture/valid/callback_async_interface_composition.concept"}},
		{"callable span lifetime async", []string{"callable/capture/valid/callback_span_capture.concept", "callable/capture/valid/callback_async_live_across_await.concept"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, relative := range tc.paths {
				if err := compileSemanticCorpusFile(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
					t.Fatalf("%s: %v", relative, err)
				}
			}
		})
	}
}

func TestFreezeArtifactsDeterministicAndRuntimeBounded(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "callable", "capture", "valid", "callback_async_interface_composition.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(baseline))
	for key := range baseline {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for iteration := 0; iteration < 100; iteration++ {
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if !bytes.Equal(outputs[key], baseline[key]) {
				t.Fatalf("%s changed at iteration %d", key, iteration)
			}
		}
	}
	forbidden := []string{"malloc(", "calloc(", "realloc(", "setjmp(", "longjmp(", "promise runtime", "future runtime", "executor runtime", "scheduler runtime", "event loop", "per-object vtable", "rtti", "closure box"}
	for name, output := range baseline {
		if !strings.HasSuffix(name, ".generated.c") {
			continue
		}
		lower := strings.ToLower(string(output))
		for _, marker := range forbidden {
			if strings.Contains(lower, marker) {
				t.Fatalf("forbidden runtime marker %q in %s", marker, name)
			}
		}
	}
}
