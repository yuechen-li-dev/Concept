package concept

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testingFixturePath(parts ...string) string {
	return filepath.Join(append([]string{"testdata", "testing"}, parts...)...)
}

func TestTestMetadataDiagnostics(t *testing.T) {
	cases := map[string]string{
		"fact_has_parameter.concept_test":        "TEST_FACT_REQUIRES_ZERO_PARAMETERS",
		"fact_nonvoid.concept_test":              "TEST_FACT_REQUIRES_VOID_RETURN",
		"theory_nonvoid.concept_test":            "TEST_THEORY_REQUIRES_VOID_RETURN",
		"foretold_without_prophecy.concept_test": "TEST_FORETOLD_REQUIRES_PROPHECY",
		"fact_and_theory.concept_test":           "TEST_KIND_CONFLICT",
		"benchmark_and_prophecy.concept_test":    "TEST_KIND_CONFLICT",
		"assert_true_nonbool.concept_test":       "TEST_ASSERT_BOOL_REQUIRED",
		"assert_near_non_numeric.concept_test":   "TEST_ASSERT_NEAR_NUMERIC_REQUIRED",
		"assert_error_non_result.concept_test":   "TEST_ASSERT_RESULT_REQUIRED",
		"assert_lgtm_non_result.concept_test":    "TEST_ASSERT_RESULT_REQUIRED",
		"assert_missing_reason.concept_test":     "TEST_ASSERT_REASON_REQUIRED",
		"assert_blank_reason.concept_test":       "TEST_ASSERT_REASON_NONEMPTY",
	}
	for file, code := range cases {
		t.Run(file, func(t *testing.T) {
			path := testingFixturePath("invalid", file)
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(filepath.ToSlash(path), string(body))
			if err == nil || !strings.Contains(err.Error(), code) {
				t.Fatalf("err=%v want %s", err, code)
			}
		})
	}
}

func TestDiscoveryManifestIsDeterministicAndCarriesMIRMetadata(t *testing.T) {
	root := filepath.Join("..", "..", "language", "evt1", "tooling", "tests")
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != TestManifestSchema || len(manifest.Tests) != 16 {
		t.Fatalf("manifest=%s tests=%d", manifest.Schema, len(manifest.Tests))
	}
	want, err := MarshalTestManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		again, err := DiscoverTests(root)
		if err != nil {
			t.Fatal(err)
		}
		got, err := MarshalTestManifest(again)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("manifest changed on discovery iteration %d", i)
		}
	}
	fact := manifest.Tests[0]
	outputs, err := Generate(fact.module, fact.sourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	mir := string(outputs[evt1OutputBase(fact.sourcePath)+".mir.json"])
	if !strings.Contains(mir, `"test_declarations"`) || !strings.Contains(mir, `"kind": "fact"`) {
		t.Fatalf("test metadata absent from MIR:\n%s", mir)
	}
}

func TestDiscoveryIgnoresHistoricalConTestAndRejectsMissingArtifact(t *testing.T) {
	dir := t.TempDir()
	ordinary := "profile Core;\n[[fact]] void Works() { Assert.True(true, \"reason is mandatory\"); }\n"
	if err := os.WriteFile(filepath.Join(dir, "old.con_test"), []byte(ordinary), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.concept_test"), []byte(ordinary), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tests) != 1 || manifest.Tests[0].Source != "new.concept_test" {
		t.Fatalf("discovered %#v", manifest.Tests)
	}
	_, err = DiscoverTests(testingFixturePath("invalid", "missing_artifact.concept_test"))
	if err == nil || !strings.Contains(err.Error(), "TEST_ARTIFACT_MISSING") {
		t.Fatalf("missing artifact err=%v", err)
	}
}

func TestAttributesRejectTyposAndArtifactsCannotEscapeRoot(t *testing.T) {
	_, err := Parse("typo.concept_test", "profile Core; [[faact]] void Typo() { }")
	if err == nil || !strings.Contains(err.Error(), "TEST_ATTRIBUTE_UNKNOWN") {
		t.Fatalf("unknown attribute err=%v", err)
	}
	parent := t.TempDir()
	root := filepath.Join(parent, "tests")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "outside.json"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := "profile Core; [[fact]] [[artifact(\"../outside.json\")]] void Escape() { }"
	if err := os.WriteFile(filepath.Join(root, "escape.concept_test"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = DiscoverTests(root)
	if err == nil || !strings.Contains(err.Error(), "TEST_ARTIFACT_OUTSIDE_ROOT") {
		t.Fatalf("escaping artifact err=%v", err)
	}
}

func TestResultAssertionsRetainPrintablePayloadDiagnostics(t *testing.T) {
	for _, tc := range []struct{ file, marker string }{
		{"assert_error.concept_test", "|success=%d"},
		{"assert_lgtm.concept_test", "|error=%d"},
	} {
		path := filepath.Join("..", "..", "language", "evt1", "tooling", "tests", tc.file)
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse(filepath.ToSlash(path), string(body))
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, body)
		if err != nil {
			t.Fatal(err)
		}
		generated := string(outputs[evt1OutputBase(path)+".generated.c"])
		if !strings.Contains(generated, tc.marker) {
			t.Fatalf("%s missing %s", tc.file, tc.marker)
		}
	}
}

func TestTestToolingGeneratedCHasNoHiddenRuntime(t *testing.T) {
	root := filepath.Join("..", "..", "language", "evt1", "tooling", "tests")
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range manifest.Tests {
		outputs, err := Generate(test.module, test.sourceBytes)
		if err != nil {
			t.Fatal(err)
		}
		generated := strings.ToLower(string(outputs[evt1OutputBase(test.sourcePath)+".generated.c"]))
		for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "setjmp(", "longjmp(", "reflection_registry", "test_registry", "task_runtime", "event_loop"} {
			if strings.Contains(generated, forbidden) {
				t.Fatalf("%s contains forbidden runtime %q", test.Source, forbidden)
			}
		}
	}
}

func TestRunnerExecutesFactsTheoryBenchmarkAsyncAndProphecy(t *testing.T) {
	root := filepath.Join("..", "..", "language", "evt1", "tooling", "tests")
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunTests(manifest, TestRunOptions{ResultsDir: t.TempDir(), BenchmarkWarmup: 1, BenchmarkIterations: 3})
	if err != nil {
		t.Fatal(err)
	}
	if run.Failed != 0 || run.Passed != 14 || run.Fulfilled != 3 || run.Benchmarks != 1 {
		t.Fatalf("run=%+v", run)
	}
	for _, result := range run.Results {
		if strings.Contains(result.TestID, "prophecy_foretold_panic") && (len(result.Artifacts) != 1 || len(result.LastCheckpoints) != 1 || result.BuildIdentity == "" || result.PanicReason == "") {
			t.Fatalf("canonical foretold evidence incomplete: %+v", result)
		}
	}
}

func TestRunnerClassifiesRuntimeNegativeCasesAndRetainsForetoldEvidence(t *testing.T) {
	manifest, err := DiscoverTests(testingFixturePath("runtime-negative"))
	if err != nil {
		t.Fatal(err)
	}
	resultsDir := t.TempDir()
	run, err := RunTests(manifest, TestRunOptions{ResultsDir: resultsDir})
	if err != nil {
		t.Fatal(err)
	}
	if run.Failed != 3 || run.Fulfilled != 1 {
		t.Fatalf("failed=%d fulfilled=%d", run.Failed, run.Fulfilled)
	}
	var assertion, foretold *TestResult
	for i := range run.Results {
		if strings.Contains(run.Results[i].TestID, "fact_assert_failure") {
			assertion = &run.Results[i]
		}
		if strings.Contains(run.Results[i].TestID, "prophecy_foretold_crash") {
			foretold = &run.Results[i]
		}
	}
	if assertion == nil {
		t.Fatal("assertion result missing")
	}
	if assertion.Failure == nil {
		t.Fatalf("assertion failure missing: %+v", *assertion)
	}
	if assertion.Failure.Actual != "4" || assertion.Failure.Expected != "5" || assertion.Failure.Message != "intentional structured assertion failure" {
		t.Fatalf("assertion result=%+v failure=%+v", *assertion, *assertion.Failure)
	}
	if foretold == nil || len(foretold.LastCheckpoints) != 1 {
		t.Fatalf("foretold=%+v", foretold)
	}
	if len(foretold.Artifacts) != 0 {
		// The runtime-negative specimen intentionally has no external artifact;
		// the canonical foretold corpus separately proves artifact retention.
		t.Fatalf("unexpected runtime-negative artifacts=%+v", foretold.Artifacts)
	}
	evidence := filepath.Join(resultsDir, "prophecy", sanitizeTestID(foretold.TestID), "latest", "result.json")
	if _, err := os.Stat(evidence); err != nil {
		t.Fatal(err)
	}
}
