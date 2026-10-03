package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEVT2eDirectCallsReachVerifiedLIR(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "evt2e_calls.concept"))
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("evt2e_calls.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyLIR(lir); err != nil {
		t.Fatal(err)
	}
	text := lir.String()
	for _, want := range []string{"call i32 @FortyTwo", "args=[]", "call bool @IdentityBool", "call void @Touch", "call i32 @Add", "call i32 @Sum5", "cc win64", "args=[i32 i32 i32 i32 i32]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in LIR:\n%s", want, text)
		}
	}
	for i := 0; i < 100; i++ {
		next, err := GenerateLIR(module)
		if err != nil || next.String() != text {
			t.Fatalf("call LIR differs on run %d: %v", i+2, err)
		}
	}
	if _, err := LowerLirToAmd64Machine(lir); err != nil {
		t.Fatalf("call contract did not reach MachineIR: %v", err)
	}
}

func TestEVT2eVerifierRejectsCallContractMutation(t *testing.T) {
	source := `profile Core;
int Id(int x) { return x; }
int F(int x) { return Id(x); }`
	module, err := Parse("calls.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	call := &lir.Functions[1].Blocks[0].Instructions[0]
	call.CallABI = "unknown"
	if err := VerifyLIR(lir); err == nil || !strings.Contains(err.Error(), "LIR_BAD_CALL") {
		t.Fatalf("invalid call convention accepted: %v", err)
	}
}
