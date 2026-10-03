package concept

import (
	"strings"
	"testing"
)

func machineFixture(t *testing.T) MachineModule {
	t.Helper()
	m, err := GenerateMachineIR(evt2Fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyMachineIR(m); err != nil {
		t.Fatal(err)
	}
	return m
}
func machineHas(f MachineFunction, op string) bool {
	for _, b := range f.Blocks {
		for _, in := range b.Instructions {
			if in.Op == op {
				return true
			}
		}
	}
	return false
}
func machineTerm(f MachineFunction, op, cond string) bool {
	for _, b := range f.Blocks {
		if b.Term.Op == op && (cond == "" || b.Term.Cond == cond) {
			return true
		}
	}
	return false
}

func TestEVT2cCoreStructure(t *testing.T) {
	m := machineFixture(t)
	if len(m.Functions) != 7 {
		t.Fatalf("functions=%d", len(m.Functions))
	}
	byName := map[string]MachineFunction{}
	for _, f := range m.Functions {
		byName[f.Name] = f
	}
	add := byName["Add"]
	if add.Args[0].Register != RCX || add.Args[1].Register != RDX || !machineHas(add, "ADD") || !machineTerm(add, "JCC", "O") || !machineTerm(add, "TRAP", "") {
		t.Fatal("Add lost ABI, overflow, or trap")
	}
	max := byName["Max"]
	if !machineHas(max, "CMP") || !machineTerm(max, "JCC", "G") || machineHas(max, "SETCC") {
		t.Fatal("Max did not branch directly on flags")
	}
	sum := byName["Sum4"]
	if !sum.Args[0].Indirect || !machineHas(sum, "LEA") || !machineHas(sum, "LOAD") || !machineHas(sum, "STORE") || !machineTerm(sum, "JCC", "AE") || !machineTerm(sum, "JMP", "") {
		t.Fatal("Sum4 lost loop, address, or guard")
	}
	for _, name := range []string{"CheckedIndex", "StoreIndex"} {
		f := byName[name]
		if !machineTerm(f, "JCC", "AE") || !machineHas(f, "LEA") || !machineTerm(f, "TRAP", "") {
			t.Fatalf("%s lost retained bounds failure", name)
		}
	}
	if !machineHas(byName["StoreIndex"], "STORE") {
		t.Fatal("indexed store missing")
	}
	if !machineTerm(byName["Early"], "JCC", "O") {
		t.Fatal("early return lost checked subtraction")
	}
	for _, f := range m.Functions {
		for _, b := range f.Blocks {
			if b.Term.Op == "RET" {
				last := b.Instructions[len(b.Instructions)-1]
				if last.Dst.Kind != "preg" || last.Dst.ID != int(RAX) {
					t.Fatalf("%s return not in RAX family", f.Name)
				}
			}
		}
	}
	loadRegion := false
	for _, block := range sum.Blocks {
		for _, in := range block.Instructions {
			if in.Op == "LOAD" && len(in.Src) == 1 && in.Src[0].Kind == "mem" && in.Src[0].Region == "s0" && in.Effects().ReadsMemory {
				loadRegion = true
			}
		}
	}
	if !loadRegion {
		t.Fatal("indexed load lost memory region or effect")
	}
}
func TestEVT2cABIAndAliases(t *testing.T) {
	regs := []MachineReg{RCX, RDX, R8, R9}
	for i := 0; i < 5; i++ {
		a, e := Win64Argument(i, "i32")
		if e != nil {
			t.Fatal(e)
		}
		if i < 4 {
			if a.OnStack || a.Register != regs[i] {
				t.Fatalf("arg %d: %+v", i, a)
			}
		} else if !a.OnStack || a.StackOffset != 40 {
			t.Fatalf("fifth arg: %+v", a)
		}
	}
	for _, typ := range []LIRType{"i32", "u32", "i64", "u64"} {
		r, w, e := Win64Return(typ)
		if e != nil || r != RAX || w != machineScalarWidth(typ) {
			t.Fatalf("return %s: %v %d %v", typ, r, w, e)
		}
	}
	if RAX.Alias(1) != "al" || RAX.Alias(2) != "ax" || RAX.Alias(4) != "eax" || RAX.Alias(8) != "rax" || R8.Alias(4) != "r8d" {
		t.Fatal("subregister aliases")
	}
	wide, e := RAX.WriteEffect(4)
	if e != nil || wide.Parent != RAX || !wide.ZeroExtendsParent {
		t.Fatal("32-bit write must zero parent high bits")
	}
	narrow, e := RAX.WriteEffect(2)
	if e != nil || narrow.ZeroExtendsParent {
		t.Fatal("16-bit write must preserve parent high bits")
	}
	if len(Win64CallerSaved) != 7 || len(Win64CalleeSaved) != 8 {
		t.Fatal("Win64 saved-set metadata")
	}
	a, e := Win64Argument(0, "[4]i32")
	if e != nil || !a.Indirect || a.Width != 8 {
		t.Fatalf("large aggregate %v %v", a, e)
	}
	for _, tc := range []struct {
		local int
		calls bool
		want  int
	}{{0, false, 0}, {12, false, 12}, {0, true, 40}, {8, true, 40}, {9, true, 56}} {
		got, e := Win64CallFrameBytes(tc.local, tc.calls)
		if e != nil || got != tc.want {
			t.Fatalf("frame %+v: %d %v", tc, got, e)
		}
	}
}
func TestEVT2cUnsignedAndUnsupported(t *testing.T) {
	src := "profile Core; uint U(uint a, uint b) { if (a < b) { return a + b; } return a - b; }"
	module, e := Parse("unsigned.concept", src)
	if e != nil {
		t.Fatal(e)
	}
	m, e := GenerateMachineIR(module)
	if e != nil {
		t.Fatal(e)
	}
	f := m.Functions[0]
	for _, cond := range []string{"B", "C"} {
		if !machineTerm(f, "JCC", cond) {
			t.Fatalf("missing unsigned %s", cond)
		}
	}
	float, e := Parse("float.concept", "profile Core; float F(float x) { return x; }")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = GenerateMachineIR(float); e == nil || !strings.Contains(e.Error(), "EVT2_UNSUPPORTED_TYPE") {
		t.Fatalf("float accepted: %v", e)
	}
}
func TestEVT2cVerifierRejectsMalformed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*MachineFunction)
	}{
		{"bad vreg", func(f *MachineFunction) { f.Blocks[0].Instructions[0].Dst.ID = 999 }},
		{"wrong class", func(f *MachineFunction) { f.Blocks[0].Instructions[0].Dst.Width = 8 }},
		{"operand kind", func(f *MachineFunction) { f.Blocks[0].Instructions[0].Src[0].Kind = "unknown" }},
		{"bad block", func(f *MachineFunction) { f.Blocks[0].Term.True = 999 }},
		{"missing flags", func(f *MachineFunction) { f.Blocks[0].Term.FlagsUse = -1 }},
		{"bad condition", func(f *MachineFunction) { f.Blocks[0].Term.Cond = "XYZ" }},
		{"missing return move", func(f *MachineFunction) { f.Blocks[2].Instructions = nil }},
		{"bad ABI", func(f *MachineFunction) { f.Args[0].Register = RAX }},
		{"width mismatch", func(f *MachineFunction) { f.Blocks[0].Instructions[2].Width = 8 }},
		{"bad terminator", func(f *MachineFunction) { f.Blocks[0].Term.Op = "" }},
		{"duplicate definition", func(f *MachineFunction) { f.Blocks[0].Instructions[2].Dst.ID = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := machineFixture(t).Functions[0]
			tc.mutate(&f)
			if e := VerifyMachineFunction(f); e == nil {
				t.Fatal("malformed MachineIR accepted")
			}
		})
	}
	f := machineFixture(t).Functions[2]
	found := false
	for bi := range f.Blocks {
		for ii := range f.Blocks[bi].Instructions {
			in := &f.Blocks[bi].Instructions[ii]
			if in.Op == "LEA" {
				in.Src[0].Scale = 3
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found || VerifyMachineFunction(f) == nil {
		t.Fatal("illegal memory scale accepted")
	}
	f = machineFixture(t).Functions[2]
	found = false
	for bi := range f.Blocks {
		for ii := range f.Blocks[bi].Instructions {
			in := &f.Blocks[bi].Instructions[ii]
			if in.Dst.Kind == "slot" {
				in.Dst.ID = 999
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found || VerifyMachineFunction(f) == nil {
		t.Fatal("bad slot accepted")
	}
	f = machineFixture(t).Functions[2]
	found = false
	for bi := range f.Blocks {
		for ii := range f.Blocks[bi].Instructions {
			in := &f.Blocks[bi].Instructions[ii]
			if len(in.Src) == 1 && in.Src[0].Kind == "imm" {
				in.Src[0].Literal = "999999999999999999999999"
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found || VerifyMachineFunction(f) == nil {
		t.Fatal("out-of-range immediate accepted")
	}
}
// The literal set is the frozen pre-R9c Go verifier decision. The production
// decision now comes from the checked Concept bridge enum's generated table.
func TestR9cConditionAuthorityShadowAgreement(t *testing.T) {
	legacy := map[string]bool{
		"E": true, "NE": true, "L": true, "LE": true,
		"G": true, "GE": true, "B": true, "BE": true,
		"A": true, "AE": true, "O": true, "C": true,
	}
	if len(bridgeTagsCondition) != len(legacy)+1 || bridgeTagsCondition[0] != "" {
		t.Fatalf("Concept schema condition table changed: %v", bridgeTagsCondition)
	}
	for _, condition := range bridgeTagsCondition {
		if got := machineConditionValid(condition); got != legacy[condition] {
			t.Fatalf("condition %q: schema=%t legacy=%t", condition, got, legacy[condition])
		}
	}
	for condition := range legacy {
		if !machineConditionValid(condition) {
			t.Fatalf("legacy condition %q missing from schema", condition)
		}
	}
	for _, invalid := range []string{"", "XYZ", "e", "None", "O|C"} {
		if machineConditionValid(invalid) {
			t.Fatalf("invalid condition %q accepted", invalid)
		}
	}
}
func TestEVT2cDeterminism100(t *testing.T) {
	module := evt2Fixture(t)
	first, e := GenerateMachineIR(module)
	if e != nil {
		t.Fatal(e)
	}
	want := first.String()
	for i := 1; i < 100; i++ {
		next, e := GenerateMachineIR(module)
		if e != nil {
			t.Fatal(e)
		}
		if next.String() != want {
			t.Fatalf("MachineIR changed on run %d", i+1)
		}
	}
}

func TestEVT2cAddPrinterGolden(t *testing.T) {
	add := machineFixture(t).Functions[0]
	const want = `fn Add [Add|Add(int, int)] amd64-windows win64 -> i32
  arg 0 i32 ecx -> v0
  arg 1 i32 edx -> v1
  frame locals=0 align=16 shadow=0
b0 (lir b0):
  MOV v0:32 ecx
  MOV v1:32 edx
  MOV v2:32 v0:32
  ADD v2:32 v1:32 flags=f0
  JCC O f0 b1 b2
b1 (lir b0):
  TRAP [overflow]
  TRAP
b2 (lir b0):
  MOV eax v2:32
  RET
`
	if got := (MachineModule{Functions: []MachineFunction{add}}).String(); got != want {
		t.Fatalf("Add MachineIR printer drift:\n%s", got)
	}
}

func TestEVT2cLIRBlockCorrespondence(t *testing.T) {
	lir, e := GenerateLIR(evt2Fixture(t))
	if e != nil {
		t.Fatal(e)
	}
	m, e := LowerLirToAmd64Machine(lir)
	if e != nil {
		t.Fatal(e)
	}
	for fi, lf := range lir.Functions {
		mf := m.Functions[fi]
		for _, lb := range lf.Blocks {
			seen := false
			returned := false
			for _, mb := range mf.Blocks {
				if mb.LIRBlock == lb.ID {
					seen = true
					if mb.Term.Op == "RET" {
						returned = true
					}
				}
			}
			if !seen || lb.Term.Op == "return" && !returned {
				t.Fatalf("%s LIR b%d lost machine path", lf.Name, lb.ID)
			}
		}
	}
}
