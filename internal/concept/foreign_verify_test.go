package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVerifyForeignNonNullViolationHasDeclaredOrigin(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang unavailable")
	}
	if _, err := exec.LookPath("clang++"); err != nil {
		t.Skip("clang++ unavailable")
	}
	dir := t.TempDir()
	source := `module Fake.Tests;
profile Core;
extern "C" byte* FakeCreate();
foreign concept ReturnContract on FakeCreate {
    requires compiler.NonNull(result);
}
[[fact]]
[[verify_foreign("ReturnContract")]]
void Check() {
    FakeCreate();
}`
	if err := os.WriteFile(filepath.Join(dir, "native.concept_test"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.Join(dir, "native.concept_test"), source)
	if err != nil {
		t.Fatal(err)
	}
	normalC, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	verifyC, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), VerifyCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(normalC["native.generated.c"]), "concept_verify_foreign_nonnull") || !strings.Contains(string(verifyC["native.generated.c"]), "concept_verify_foreign_nonnull") {
		t.Fatal("foreign observation must appear only in Verify generated C")
	}
	fake := filepath.Join(dir, "fake.c")
	if err := os.WriteFile(fake, []byte("#include <stdint.h>\nuint8_t* FakeCreate(void) { return 0; }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	object := filepath.Join(dir, "fake.o")
	if output, err := exec.Command(clang, "-std=c11", "-c", fake, "-o", object).CombinedOutput(); err != nil {
		t.Fatalf("fake native fixture: %v\n%s", err, output)
	}
	manifest, err := DiscoverTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunTests(manifest, TestRunOptions{Verify: true, NativeLinker: "clang++", NativeLinkInputs: []string{object}, ResultsDir: filepath.Join(dir, ".test-results")})
	if err != nil {
		t.Fatal(err)
	}
	if run.Failed != 1 || len(run.Results) != 1 || run.Results[0].Failure == nil || run.Results[0].Failure.Kind != "verification" {
		t.Fatalf("false foreign contract did not fail: %+v", run)
	}
	result := run.Results[0]
	if len(result.Verifications) != 1 || result.Verifications[0].Passed || result.Verifications[0].Origin != string(FactOriginDeclaredForeign) || result.Verifications[0].Contract != "ReturnContract" || !strings.HasSuffix(result.Verifications[0].DeclarationSource, "native.concept_test") || result.Verifications[0].DeclarationLine != 4 || result.Verifications[0].CallLine != 10 || !strings.Contains(result.Stderr, "foreign contract violated in this execution") {
		t.Fatalf("foreign violation lost source provenance: %+v", result)
	}
	for repetition := 2; repetition <= 100; repetition++ {
		repeated, err := RunTests(manifest, TestRunOptions{Verify: true, NativeLinker: "clang++", NativeLinkInputs: []string{object}, ResultsDir: filepath.Join(dir, ".test-results")})
		if err != nil || len(repeated.Results) != 1 || !reflect.DeepEqual(result.Verifications, repeated.Results[0].Verifications) {
			t.Fatalf("foreign verification report changed on run %d: %v %+v", repetition, err, repeated.Results)
		}
	}
}

func TestVerifyForeignRejectsContractWithoutRuntimeChecker(t *testing.T) {
	dir := t.TempDir()
	source := `profile Core;
extern "C" byte* FakeCreate();
foreign concept AllocationClaim on FakeCreate {
    requires compiler.Allocates(FakeCreate);
}
[[fact]]
[[verify_foreign("AllocationClaim")]]
void Check() { FakeCreate(); }`
	if err := os.WriteFile(filepath.Join(dir, "native.concept_test"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := DiscoverTests(dir)
	if err == nil || !strings.Contains(err.Error(), "VERIFY_RUNTIME_UNSUPPORTED") {
		t.Fatalf("unsupported runtime contract was accepted: %v", err)
	}
}
