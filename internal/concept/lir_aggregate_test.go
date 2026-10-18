package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func evt2gObjectFixture(t *testing.T) (Module, []byte) {
	t.Helper()
	path := "../../language/evt2g/valid/objects.concept"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	return module, source
}

func TestEVT2gCheckedAggregateLIR(t *testing.T) {
	module, _ := evt2gObjectFixture(t)
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	text := lir.String()
	for _, want := range []string{"Position identity=", "Particle identity=", "kind=struct repr=native size=24 align=8", "Small identity=", "kind=record repr=C size=4 align=4", "member 1 value: float64 offset=8 size=8 align=8", "make_aggregate aggregate:", "slot_address ptr<aggregate:", "field_address ptr<float64>", "member=1 offset=16", "make_array [4]float64", "index_address ptr<float64>", "stride=8", "load_slot aggregate:", "store_slot aggregate:", "@ByValue", "call aggregate:", "PerAccessRuntime"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in LIR:\n%s", want, text)
		}
	}
	if len(lir.Aggregates) != 8 {
		t.Fatalf("layouts=%d", len(lir.Aggregates))
	}
	// Source named-initializer evaluation order survives, while the complete
	// aggregate's operands are in checked declaration order.
	for _, f := range lir.Functions {
		if f.Name == "Named" {
			in := f.Blocks[0].Instructions
			if len(in) != 2 || in[0].Op != "const" || in[1].Op != "make_aggregate" || in[1].Args[0] != f.Params[0].ID || in[1].Args[1] != in[0].Result {
				t.Fatalf("named initializer order: %+v", in)
			}
		}
	}
	start := time.Now()
	for run := 0; run < 100; run++ {
		next, err := GenerateLIR(module)
		if err != nil || next.String() != text {
			t.Fatalf("run %d changed aggregate LIR: %v", run+1, err)
		}
	}
	t.Logf("100 checked aggregate LIR generations: %s; inspection bytes=%d layouts=%d functions=%d", time.Since(start), len(text), len(lir.Aggregates), len(lir.Functions))
	if _, err := LowerLirToAmd64Machine(lir); err == nil || !strings.Contains(err.Error(), "MIR_AGGREGATE_REQUIRES_CMIRAMD5") {
		t.Fatalf("aggregate evidence erased into scalar bridge: %v", err)
	}
	// An otherwise unused integer-only aggregate also requires the new bridge.
	small, err := Parse("small.concept", `profile Core; record struct R { int x; } int F() { R r = R{1}; return 2; }`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateMachineIR(small); err == nil || !strings.Contains(err.Error(), "MIR_AGGREGATE_REQUIRES_CMIRAMD5") {
		t.Fatalf("unused aggregate bypassed version gate: %v", err)
	}
}

func TestEVT2gLayoutIdentityAndMalformedInput(t *testing.T) {
	module, _ := evt2gObjectFixture(t)
	base, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	find := func(m *LIRModule, function, op string) *LIRInstruction {
		for fi := range m.Functions {
			f := &m.Functions[fi]
			if f.Name != function {
				continue
			}
			for bi := range f.Blocks {
				for ii := range f.Blocks[bi].Instructions {
					in := &f.Blocks[bi].Instructions[ii]
					if in.Op == op {
						return in
					}
				}
			}
		}
		t.Fatalf("missing %s/%s", function, op)
		return nil
	}
	cases := []struct {
		name, want string
		mutate     func(*LIRModule)
	}{
		{"stale offset", "LIR_STALE_AGGREGATE_LAYOUT", func(m *LIRModule) { m.Aggregates[0].Fields[0].Offset++ }},
		{"stale representation", "LIR_STALE_AGGREGATE_LAYOUT", func(m *LIRModule) { m.Aggregates[0].Representation = "C" }},
		{"stale identity", "LIR_STALE_AGGREGATE_LAYOUT", func(m *LIRModule) { m.Aggregates[0].Identity += "changed" }},
		{"bad representation", "LIR_BAD_AGGREGATE_LAYOUT", func(m *LIRModule) {
			l := &m.Aggregates[0]
			l.Representation = "packed"
			l.Type = lirAggregateIdentity(*l)
		}},
		{"out of object", "LIR_BAD_AGGREGATE_FIELD", func(m *LIRModule) {
			l := &m.Aggregates[0]
			l.Fields[1].Offset = l.Size
			l.Type = lirAggregateIdentity(*l)
		}},
		{"wrong field width", "LIR_BAD_AGGREGATE_FIELD_TYPE", func(m *LIRModule) { l := &m.Aggregates[0]; l.Fields[0].Type = "i32"; l.Type = lirAggregateIdentity(*l) }},
		{"wrong field offset", "LIR_BAD_FIELD_OFFSET", func(m *LIRModule) { find(m, "Read", "field_address").FrameOffset++ }},
		{"wrong field type", "LIR_BAD_FIELD_OFFSET", func(m *LIRModule) { find(m, "Read", "field_address").Type = "ptr<i32>" }},
		{"invalid member", "LIR_BAD_FIELD_ADDRESS", func(m *LIRModule) { find(m, "Read", "field_address").Member = 99 }},
		{"incomplete construction", "LIR_BAD_RECORD_INIT", func(m *LIRModule) { find(m, "Read", "make_aggregate").Args = nil }},
		{"wrong initializer", "LIR_BAD_RECORD_MEMBER", func(m *LIRModule) { in := find(m, "Padding", "make_aggregate"); in.Args[1] = in.Args[0] }},
		{"not live", "LIR_OBJECT_NOT_LIVE", func(m *LIRModule) {
			for fi := range m.Functions {
				f := &m.Functions[fi]
				if f.Name == "Read" {
					in := f.Blocks[0].Instructions
					for i, x := range in {
						if x.Op == "initialize_slot" {
							f.Blocks[0].Instructions = append(in[:i], in[i+1:]...)
							return
						}
					}
				}
			}
		}},
		{"wrong stride", "LIR_BAD_OBJECT_INDEX", func(m *LIRModule) { find(m, "ArrayField", "index_address").Stride = 4 }},
		{"missing bounds", "LIR_MISSING_INDEX_GUARD", func(m *LIRModule) { find(m, "ArrayField", "check_index").Extent++ }},
		{"record mutation", "LIR_OBJECT_READONLY", func(m *LIRModule) {
			for i := range m.Aggregates {
				l := &m.Aggregates[i]
				if l.Name == "Position" {
					old := l.Type
					l.Kind = "record"
					l.Type = lirAggregateIdentity(*l)
					replaceLIRAggregateType(m, old, l.Type)
					return
				}
			}
		}},
		{"const place mutation", "LIR_OBJECT_READONLY", func(m *LIRModule) {
			for i := range m.Functions {
				if m.Functions[i].Name == "Nested" {
					m.Functions[i].Slots[0].Readonly = true
				}
			}
		}},
		{"with duplicate member", "LIR_BAD_RECORD_WITH_MEMBER", func(m *LIRModule) { in := find(m, "WithCalls", "record_with"); in.Members[1] = in.Members[0] }},
		{"with invalid member", "LIR_BAD_RECORD_WITH_MEMBER", func(m *LIRModule) { find(m, "With", "record_with").Members[0] = 99 }},
		{"with wrong replacement type", "LIR_BAD_RECORD_WITH_MEMBER", func(m *LIRModule) { in := find(m, "With", "record_with"); in.Args[1] = in.Args[0] }},
		{"with missing base", "LIR_BAD_RECORD_WITH", func(m *LIRModule) { find(m, "With", "record_with").Args = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var m LIRModule
			if err := json.Unmarshal(encoded, &m); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&m)
			if err := VerifyLIR(m); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %s; got %v", tc.want, err)
			}
		})
	}
	for _, mutate := range []func(*LIRAggregateLayout){
		func(l *LIRAggregateLayout) { l.Identity += "another declaration" },
		func(l *LIRAggregateLayout) { l.Representation = "C" },
		func(l *LIRAggregateLayout) {
			if l.Kind == "record" {
				l.Kind = "struct"
			} else {
				l.Kind = "record"
			}
		},
		func(l *LIRAggregateLayout) {
			l.Fields = append([]LIRAggregateField(nil), l.Fields...)
			l.Fields[0].Offset++
		},
	} {
		l := base.Aggregates[0]
		mutate(&l)
		if lirAggregateIdentity(l) == base.Aggregates[0].Type {
			t.Fatal("layout-relevant change omitted from identity")
		}
	}
}

func TestEVT2gRecordWithEvaluationOrder(t *testing.T) {
	m, _ := evt2gObjectFixture(t)
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range lir.Functions {
		if f.Name != "WithCalls" {
			continue
		}
		var calls []string
		var values []int
		var update *LIRInstruction
		for i := range f.Blocks[0].Instructions {
			in := &f.Blocks[0].Instructions[i]
			if in.Op == "call" {
				calls = append(calls, in.Callee)
				values = append(values, in.Result)
			}
			if in.Op == "record_with" {
				update = in
			}
		}
		if strings.Join(calls, ",") != "WithBase,WithFirst,WithSecond" || update == nil || fmt.Sprint(update.Members) != "[1 0]" || fmt.Sprint(update.Args) != fmt.Sprint(values) {
			t.Fatalf("with base/replacement order changed: %v %v %+v", calls, values, update)
		}
		return
	}
	t.Fatal("missing WithCalls fixture")
}

func TestEVT2gWithSourceSideEffects(t *testing.T) {
	const source = `module WithOrder; profile Core;
record struct R { int x; int y; }
R Base(ref int trace){ trace=trace*10+1; return R{1,2}; }
int First(ref int trace){ trace=trace*10+2; return 3; }
int Second(ref int trace){ trace=trace*10+3; return 4; }
int Main(){ int trace=0; R updated=Base(ref trace) with { y=First(ref trace); x=Second(ref trace); }; return trace*100+updated.x*10+updated.y; }
`
	m, err := Parse("withorder.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		name, policy := "Normal", ConservativeCompilationPolicy()
		if verify {
			name, policy = "Verify", VerifyCompilationPolicy()
		}
		t.Run(name, func(t *testing.T) {
			out, err := GenerateForTargetWithPolicy(m, []byte(source), GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			runFoundationNativeHarnessOutput(t, out, "withorder_host.c", "#include \"withorder.generated.h\"\nint main(void){return concept_with_order_main()==12343 ? 0 : 1;}\n", "-pedantic-errors", "-O2")
		})
	}
}

// Retype every use after a test changes a checked descriptor. This is test-only
// mutation machinery, not a production layout repair path.
func replaceLIRAggregateType(m *LIRModule, old, next LIRType) {
	replace := func(t LIRType) LIRType { return LIRType(strings.ReplaceAll(string(t), string(old), string(next))) }
	for i := range m.Aggregates {
		l := &m.Aggregates[i]
		changed := false
		for j := range l.Fields {
			t := replace(l.Fields[j].Type)
			changed = changed || t != l.Fields[j].Type
			l.Fields[j].Type = t
		}
		if changed {
			prior := l.Type
			l.Type = lirAggregateIdentity(*l)
			replaceLIRAggregateType(m, prior, l.Type)
		}
	}
	for i := range m.Functions {
		f := &m.Functions[i]
		f.Result = replace(f.Result)
		for j := range f.Params {
			f.Params[j].Type = replace(f.Params[j].Type)
		}
		for j := range f.Slots {
			f.Slots[j].Type = replace(f.Slots[j].Type)
			f.Slots[j].Element = replace(f.Slots[j].Element)
		}
		for j := range f.Blocks {
			for k := range f.Blocks[j].Instructions {
				in := &f.Blocks[j].Instructions[k]
				in.Type = replace(in.Type)
				in.Aggregate = replace(in.Aggregate)
				for a := range in.ArgTypes {
					in.ArgTypes[a] = replace(in.ArgTypes[a])
				}
			}
		}
	}
}

func TestEVT2gSourceStrictC11(t *testing.T) {
	module, source := evt2gObjectFixture(t)
	const harness = `#include "objects.generated.h"
#include <stddef.h>
#include <stdio.h>
typedef struct RefPosition { double x,y; } RefPosition;
typedef struct RefParticle { RefPosition position; double mass; } RefParticle;
typedef struct RefPadded { _Bool enabled; double value; } RefPadded;
_Static_assert(sizeof(concept_position)==sizeof(RefPosition),"position size");
_Static_assert(_Alignof(concept_particle)==_Alignof(RefParticle),"particle alignment");
_Static_assert(offsetof(concept_particle,mass)==offsetof(RefParticle,mass),"mass offset");
_Static_assert(offsetof(concept_padded,value)==offsetof(RefPadded,value),"padding offset");
#define F(n) concept_evt2g__objects_##n
#define REQUIRE(c) do { if(!(c)){fprintf(stderr,"line %d: %s\n",__LINE__,#c);return __LINE__;} } while(0)
int main(void) {
 for(int i=0;i<40;++i){
  double x=(double)i/2.0;
  REQUIRE(F(read)(x,2.0)==x+2.0);
  REQUIRE(F(nested)(x)==x+13.0);
  REQUIRE(F(copy)(x)==102.0);
  REQUIRE(F(padding)(1,x)==x && F(padding)(0,x)==0.0);
  REQUIRE(F(calls)(x)==2.0*x+10.0);
  REQUIRE(F(early)(1,x)==x && F(early)(0,x)==7.0);
  REQUIRE(F(record_copy)(x)==x+2.0);
  REQUIRE(F(with)(x)==x+11.0);
  REQUIRE(F(nested_with)(x)==x+10.0);
  concept_record_pair updated=F(with_calls)(x); REQUIRE(updated.x==x+2.0 && updated.y==x+1.0);
  REQUIRE(F(small_value)(i)==i);
  concept_position named=F(named)(x); REQUIRE(named.x==x && named.y==2.0);
 }
 for(int i=0;i<4;++i){ REQUIRE(F(array_field)(i)==9.0); REQUIRE(F(repeat_field)(i)==2.0); REQUIRE(F(array_with)(i)==i+3.0); }
 puts("evt2g source C11 object oracle passed"); return 0;
}`
	for _, verify := range []bool{false, true} {
		name, policy := "Normal", ConservativeCompilationPolicy()
		if verify {
			name, policy = "Verify", VerifyCompilationPolicy()
		}
		t.Run(name, func(t *testing.T) {
			outputs, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), policy)
			if err != nil {
				t.Fatal(err)
			}
			runFoundationNativeHarnessOutput(t, outputs, "objects_host.c", harness, "-pedantic-errors", "-O2")
		})
	}
}

func TestEVT2gSourceAndNativeBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"record immutability", `record struct R { int x; } int F() { R r=R{1}; r.x=2; return r.x; }`, "CV4142"},
		{"record immutability through ref", `record struct R { int x; } void F(ref R r) { r.x=2; }`, "CV4142"},
		{"const borrow", `struct R { int x; } void F(ref const R r) { r.x=2; }`, "CV4513"},
		{"local escape", `struct R { int x; } ref R F() { R r=R{1}; return ref r; }`, "CV4511"},
		{"field escape", `struct R { int x; } ref int F() { R r=R{1}; return ref r.x; }`, "CV4511"},
		{"uninitialized", `struct R { int x; } int F() { R r; return r.x; }`, "CV4560"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("boundary.concept", "profile Core; "+tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %s; got %v", tc.want, err)
			}
		})
	}
	for _, source := range []string{
		`struct R { int x; } int F(ref R r) { return r.x; }`,
		`struct R { int x; } int F(ref const R r) { return r.x; }`,
		`int F(ref int x) { return x; }`,
	} {
		m, err := Parse("reference.concept", "profile Core; "+source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := GenerateLIR(m); err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_OBJECT_AUTHORITY") {
			t.Fatalf("reference collapsed to scalar: %v", err)
		}
	}
	for _, tc := range []struct{ source, want string }{
		{`immovable struct R { int x; } int F(){ R r=R{1}; return r.x; }`, "EVT2_UNSUPPORTED_AGGREGATE_REPRESENTATION"},
		{`struct Resource { int x; } struct R { owned Resource value; } R Make(){ return R{Resource{1}}; } void Drop(owned Resource r) {}`, "EVT2_UNSUPPORTED_AGGREGATE_REPRESENTATION"},
	} {
		m, err := Parse("unsupported.concept", "profile Core; "+tc.source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := GenerateLIR(m); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("wanted %s; got %v", tc.want, err)
		}
	}
}

func TestEVT2gObjectInitializationCFG(t *testing.T) {
	module, _ := evt2gObjectFixture(t)
	base, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	object := base.Aggregates[0].Type
	m := LIRModule{Aggregates: base.Aggregates, Functions: []LIRFunction{{Identity: "F|F(bool,object)", Name: "F", Result: object,
		Params: []LIRValue{{ID: 0, Type: "bool"}, {ID: 1, Type: object}}, Slots: []LIRSlot{{ID: 0, Name: "r", Type: object, ParamValue: -1}},
		Blocks: []LIRBlock{
			{ID: 0, Term: LIRTerminator{Op: "branch", Value: 0, True: 1, False: 2}},
			{ID: 1, Instructions: []LIRInstruction{{Op: "initialize_slot", Type: object, Args: []int{1}, Result: -1, Slot: 0}}, Term: LIRTerminator{Op: "jump", True: 3}},
			{ID: 2, Instructions: []LIRInstruction{{Op: "initialize_slot", Type: object, Args: []int{1}, Result: -1, Slot: 0}}, Term: LIRTerminator{Op: "jump", True: 3}},
			{ID: 3, Instructions: []LIRInstruction{{Op: "load_slot", Type: object, Result: 2, Slot: 0}}, Term: LIRTerminator{Op: "return", Value: 2}},
		},
	}}}
	if err := VerifyLIR(m); err != nil {
		t.Fatal(err)
	}
	m.Functions[0].Blocks[2].Instructions = nil
	if err := VerifyLIR(m); err == nil || !strings.Contains(err.Error(), "LIR_OBJECT_NOT_LIVE") {
		t.Fatalf("one initialized predecessor proved both: %v", err)
	}
	// A declaration inside a loop begins a new local object's lifetime on
	// each iteration; const initialization is distinct from reassignment.
	source := `profile Core; record struct R { int x; } int F(int n){ int sum=0; for(int i in 0 .. n) { const R r=R{i}; sum=sum+r.x; } return sum; }`
	loop, err := Parse("loop.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateLIR(loop); err != nil {
		t.Fatal(err)
	}
}

func TestEVT2gImportedAndClosedGenericLayouts(t *testing.T) {
	const declaration = `module ObjectTypes; profile Core;
record struct Position { double x; double y; }
template <typename T> record struct Box { T value; }
`
	artifact, err := CompileSemanticModule("ObjectTypes.concept", declaration, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module ObjectUse; profile Core; import ObjectTypes;
double Read(double x){ Position p=Position{x,2.0}; return p.x; }
int Closed(){ Box<int> b=Box<int>{7}; Box<int> updated=b with { value=9; }; return updated.value; }
`
	m, err := ParseWithSemanticModules("ObjectUse.concept", consumer, map[string][]byte{"ObjectTypes": artifact})
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(lir.Aggregates) != 2 {
		t.Fatalf("imported/closed layout count=%d", len(lir.Aggregates))
	}
	if lir.Aggregates[0].Identity != "ObjectTypes|Position" || lir.Aggregates[1].Size != 4 || !strings.Contains(lir.Aggregates[1].Identity, `"module":"ObjectTypes"`) {
		t.Fatalf("lost structured defining identity: %+v", lir.Aggregates)
	}
	// This proves source-free semantic declaration transport into LIR. It is
	// deliberately not an aggregate MachineIR artifact/native qualification.
}

func TestEVT2gAggregateBounds(t *testing.T) {
	var manyFields strings.Builder
	manyFields.WriteString("profile Core; struct R {")
	for i := 0; i < 65; i++ {
		fmt.Fprintf(&manyFields, "int field%d;", i)
	}
	manyFields.WriteString("} int F(R r){return 1;}")
	var deep strings.Builder
	deep.WriteString("profile Core; struct R0 {int x;}")
	for i := 1; i < 18; i++ {
		fmt.Fprintf(&deep, "struct R%d {R%d child;}", i, i-1)
	}
	deep.WriteString("int F(R17 r){return 1;}")
	for _, tc := range []struct{ source, want string }{
		{manyFields.String(), "EVT2_AGGREGATE_FIELD_CAPACITY"},
		{deep.String(), "EVT2_AGGREGATE_DEPTH_EXCEEDED"},
		{`profile Core; struct R { double<array>[257] values; } int F(R r){return 1;}`, "EVT2_UNSUPPORTED_OBJECT_ARRAY"},
	} {
		module, err := Parse("capacity.concept", tc.source)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := GenerateLIR(module); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("wanted %s; got %v", tc.want, err)
		}
	}
	// A freshly rehashed but excessively nested descriptor graph must not
	// evade the producer's bounded nesting rule at the verifier boundary.
	var list []LIRAggregateLayout
	for i := 0; i < 18; i++ {
		field := LIRAggregateField{Name: "child", Type: "i32", TypeIdentity: "checked-int", Size: 4, Alignment: 4}
		if i != 0 {
			field.Type = list[i-1].Type
		}
		l := LIRAggregateLayout{Identity: fmt.Sprintf("T%d", i), Name: fmt.Sprintf("T%d", i), Kind: "struct", Representation: "native", Size: 4, Alignment: 4, Fields: []LIRAggregateField{field}}
		l.Type = lirAggregateIdentity(l)
		list = append(list, l)
	}
	if _, err := verifyLIRAggregateLayouts(list); err == nil || !strings.Contains(err.Error(), "LIR_AGGREGATE_DEPTH_EXCEEDED") {
		t.Fatalf("unbounded descriptor depth: %v", err)
	}
}

func TestEVT2gConstFieldAuthority(t *testing.T) {
	const source = `profile Core; struct R { const int x; } int F(){ R r=R{1}; return r.x; }`
	m, err := Parse("constfield.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(m)
	if err != nil {
		t.Fatal(err)
	}
	if !lir.Aggregates[0].Fields[0].Readonly {
		t.Fatal("const field lost checked authority")
	}
	instructions := &lir.Functions[0].Blocks[0].Instructions
	var address, value int
	for _, in := range *instructions {
		if in.Op == "field_address" {
			address = in.Result
		}
		if in.Op == "load" {
			value = in.Result
		}
	}
	*instructions = append(*instructions, LIRInstruction{Op: "store", Result: -1, Type: "i32", Args: []int{address, value}, Slot: -1})
	if err := VerifyLIR(lir); err == nil || !strings.Contains(err.Error(), "LIR_OBJECT_READONLY") {
		t.Fatalf("forged const-field write: %v", err)
	}
}

func TestEVT2gFieldSemanticIdentity(t *testing.T) {
	const source = `profile Core; record struct R { double<m> x; } R F(double<m> x){ return R{x}; }`
	var identities []LIRType
	for _, text := range []string{source, strings.ReplaceAll(source, "<m>", "<s>")} {
		module, err := Parse("units.concept", text)
		if err != nil {
			t.Fatal(err)
		}
		lir, err := GenerateLIR(module)
		if err != nil {
			t.Fatal(err)
		}
		if lir.Aggregates[0].Size != 8 || lir.Aggregates[0].Fields[0].Type != "float64" {
			t.Fatal("unit changed physical representation")
		}
		identities = append(identities, lir.Aggregates[0].Type)
	}
	if identities[0] == identities[1] {
		t.Fatal("same-width field meaning change omitted from layout identity")
	}
}

func TestEVT2gPlainDataAndCABIStayDistinct(t *testing.T) {
	artifacts := r9b3VocabularyArtifacts(t)
	const prefix = `module EVT2g.Plain; profile Core; import Standard.Semantic.Vocabulary;
record struct Flags { bool enabled; }
concept CAbiValue<T> { requires compiler.CAbiLayout<T>(); }
int F(){ Assert.Concept<PlainData>(Flags,"semantic object"); return 1; }
`
	if _, err := ParseWithSemanticModules("plain.concept", prefix, artifacts); err != nil {
		t.Fatal(err)
	}
	negative := prefix + `int G(){ Assert.Concept<CAbiValue>(Flags,"separate foreign contract"); return 1; }`
	if _, err := ParseWithSemanticModules("plain.concept", negative, artifacts); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") || !strings.Contains(err.Error(), "lacks [[repr(C)]]") {
		t.Fatalf("PlainData authorized C ABI: %v", err)
	}
	refuted := prefix + `struct Resource { int x; } void Drop(owned Resource r) {} record struct Owner { owned Resource value; }
int G(){ Assert.Concept<PlainData>(Owner,"resource is not plain"); return 1; }`
	if _, err := ParseWithSemanticModules("resource.concept", refuted, artifacts); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") || !strings.Contains(err.Error(), "FieldProblem") {
		t.Fatalf("missing typed PlainData refutation: %v", err)
	}
}
