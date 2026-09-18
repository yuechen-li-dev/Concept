package concept

import (
	"encoding/json"
	"strings"
	"testing"
)

const r7d4AccessSource = `module Access.Proof;
profile Core;

record struct AgentA {}
record struct AgentB {}
struct Slot { int Value; }
record struct Queue {}

void Store(ref Slot slot, int value)
{
    slot.Value = value;
}

[[execution_context(AgentA)]]
void AgentAEntry(ref Slot slot)
{
    Store(ref slot, 1);
}

[[execution_context(AgentA)]]
[[semantic_access("Publish", queue)]]
void AgentAPublish(ref Queue queue)
{
}

void CheckDerivedAuthority()
{
    Assert.Concept<sync.ExclusiveWriter>(AgentA, Slot, "derived writer authority");
    Assert.Concept<sync.SingleProducer>(AgentA, Queue, "derived producer authority");
}
`

func r7d4Env(t *testing.T, source string) (*semanticEnv, Module) {
	t.Helper()
	module, err := parseSyntaxModule("access.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	return env, module
}

func TestR7d4LocalCallAndContextAccessSummary(t *testing.T) {
	env, module := r7d4Env(t, r7d4AccessSource)
	if mir := buildMIR(module, env); len(mir.AccessSummaries) == 0 {
		t.Fatal("demanded MIR omitted access summaries")
	}
	var write, publish *MIRAccessEntry
	for i := range env.accessSummaries {
		entry := &env.accessSummaries[i]
		if entry.Operation == AccessWrite && entry.Function.Name == "AgentAEntry" && entry.Subject.Root.Type.valueType().Name == "Slot" {
			write = entry
		}
		if entry.Operation == AccessPublish {
			publish = entry
		}
	}
	if write == nil || write.Context.Type.Name != "AgentA" || write.Resolution != AccessExact {
		t.Fatalf("missing propagated exact AgentA write: %#v", env.accessSummaries)
	}
	if publish == nil || publish.Subject.Root.Kind != "Parameter" || publish.Context.Type.Name != "AgentA" {
		t.Fatalf("explicit publish contract did not remain structural: %#v", publish)
	}
	result := evt1DerivedSharedAccess(env, FactExclusiveWriter, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Slot", Kind: TypeStruct}})
	if result.Outcome != FactProven || result.Origin != FactOriginDerivedAccessSummary {
		t.Fatalf("expected derived exclusive writer, got %#v", result)
	}
	producer := evt1DerivedSharedAccess(env, FactSingleProducer, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Queue", Kind: TypeStruct}})
	if producer.Outcome != FactProven {
		t.Fatalf("expected single producer, got %#v", producer)
	}
}

func TestR7d4TwoWritersDisproveAndContradictDeclaration(t *testing.T) {
	source := strings.Replace(r7d4AccessSource, `[[execution_context(AgentA)]]
[[semantic_access("Publish", queue)]]`, `[[execution_context(AgentB)]]
void AgentBEntry(ref Slot slot)
{
    slot.Value = 2;
}

[[execution_context(AgentA)]]
[[semantic_access("Publish", queue)]]`, 1)
	source = strings.Replace(source, `
void CheckDerivedAuthority()
{
    Assert.Concept<sync.ExclusiveWriter>(AgentA, Slot, "derived writer authority");
    Assert.Concept<sync.SingleProducer>(AgentA, Queue, "derived producer authority");
}
`, "", 1)
	env, _ := r7d4Env(t, source)
	result := evt1DerivedSharedAccess(env, FactExclusiveWriter, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Slot", Kind: TypeStruct}})
	if result.Outcome != FactDisproven || !strings.Contains(result.Evidence.Detail, "AgentA") || !strings.Contains(result.Evidence.Detail, "AgentB") {
		t.Fatalf("expected two-writer falsifier, got %#v", result)
	}
	declared := strings.Replace(source, "record struct Queue {}", "record struct Queue {}\nrequires sync.ExclusiveWriter<AgentA, Slot>;", 1)
	if _, err := Parse("contradiction.concept", declared); err == nil || !strings.Contains(err.Error(), "SYNC_AUTHORITY_CONTRADICTION") {
		t.Fatalf("expected declared-vs-derived contradiction, got %v", err)
	}
}

func TestR7d4DynamicIndexIsOpaqueAndAtomicKindsRemainDistinct(t *testing.T) {
	source := `module Access.Opaque;
profile Core;
record struct Agent {}
[[execution_context(Agent)]]
void Dynamic(ref int[4] values, int index)
{
    values[index] = 1;
}`
	env, _ := r7d4Env(t, source)
	foundOpaque := false
	for _, entry := range env.accessSummaries {
		if entry.Operation == AccessWrite && entry.Resolution == AccessOpaque {
			foundOpaque = true
		}
	}
	if !foundOpaque {
		t.Fatalf("dynamic index did not remain opaque: %#v", env.accessSummaries)
	}

	atomicSource := standardMemorySource(t, "Standard/Synchronization/Atomic.concept")
	atomicEnv, atomicModule := r7d4Env(t, atomicSource)
	if err := evt1DeriveAccessSummaries(atomicEnv, atomicModule); err != nil {
		t.Fatal(err)
	}
	read, write := false, false
	for _, entry := range atomicEnv.accessSummaries {
		read = read || entry.Operation == AccessAtomicRead
		write = write || entry.Operation == AccessAtomicWrite
	}
	if !read || !write {
		t.Fatalf("atomic RMW access kinds were collapsed: read=%v write=%v", read, write)
	}
}

func TestR7d4AccessSummaryArtifactTransportAndDeterminism(t *testing.T) {
	first, err := CompileSemanticModule("access.concept", r7d4AccessSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileSemanticModule("access.concept", r7d4AccessSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("access-summary artifact is nondeterministic")
	}
	graph, err := ExplainSource("access.concept", r7d4AccessSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	graphJSON, _ := json.Marshal(graph)
	parsed, err := Parse("access.concept", r7d4AccessSource)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := Generate(parsed, []byte(r7d4AccessSource))
	if err != nil {
		t.Fatal(err)
	}
	var mirName string
	for name := range generated {
		if strings.HasSuffix(name, ".mir.json") {
			mirName = name
		}
	}
	for i := 0; i < 100; i++ {
		again, compileErr := CompileSemanticModule("access.concept", r7d4AccessSource, nil)
		if compileErr != nil || string(again) != string(first) {
			t.Fatalf("artifact changed on run %d: %v", i+1, compileErr)
		}
		againGraph, explainErr := ExplainSource("access.concept", r7d4AccessSource, 0)
		againGraphJSON, _ := json.Marshal(againGraph)
		if explainErr != nil || string(againGraphJSON) != string(graphJSON) {
			t.Fatalf("proof changed on run %d: %v", i+1, explainErr)
		}
		againOutputs, generateErr := Generate(parsed, []byte(r7d4AccessSource))
		if generateErr != nil || string(againOutputs[mirName]) != string(generated[mirName]) {
			t.Fatalf("MIR changed on run %d: %v", i+1, generateErr)
		}
	}
	var artifact SemanticModuleArtifact
	if err := json.Unmarshal(first, &artifact); err != nil {
		t.Fatal(err)
	}
	if len(artifact.AccessSummaries) == 0 {
		t.Fatal("concept-module.v1 omitted access summaries")
	}
	for _, entry := range artifact.AccessSummaries {
		if entry.Origin != FactOriginDerivedAccessSummary {
			t.Fatalf("local artifact origin changed: %s", entry.Origin)
		}
	}
	consumer := `module Access.Consumer;
profile Core;
import Access.Proof;
void CheckImportedAccess()
{
    Assert.Concept<sync.ExclusiveWriter>(AgentA, Slot, "artifact-only writer proof");
}`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Access.Proof": first})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	foundImported := false
	for _, entry := range env.accessSummaries {
		foundImported = foundImported || entry.Origin == FactOriginModuleAccessSummary
	}
	if !foundImported {
		t.Fatal("artifact-only consumer omitted ModuleAccessSummary evidence")
	}
	importedProof := evt1DerivedSharedAccess(env, FactExclusiveWriter, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Slot", Kind: TypeStruct}})
	if importedProof.Outcome != FactProven {
		t.Fatalf("artifact-only access proof did not close: %#v", importedProof)
	}
}

func TestR7d4NoMethodNameRecognition(t *testing.T) {
	source := `module Access.Names;
profile Core;
record struct Agent {}
record struct Queue {}
[[execution_context(Agent)]]
void Publish(ref Queue queue) {}`
	env, _ := r7d4Env(t, source)
	for _, entry := range env.accessSummaries {
		if entry.Operation == AccessPublish {
			t.Fatalf("method spelling created publish semantics: %#v", entry)
		}
	}
}

func TestR7d4ForeignBoundaryAndRecursiveFixpoint(t *testing.T) {
	source := `module Access.Foreign;
profile Core;
record struct Agent {}
struct Slot { int Value; }
extern "C" void Opaque(byte* slot);
[[semantic_access("Consume", queue)]]
extern "C" void Receive(int* queue);
void Recursive(ref Slot slot)
{
    slot.Value = 1;
    Recursive(ref slot);
}
[[execution_context(Agent)]]
void Entry(byte* opaque, int* queue, ref Slot slot)
{
    Opaque(opaque);
    Receive(queue);
    Recursive(ref slot);
}`
	env, _ := r7d4Env(t, source)
	opaque, foreignConsume := false, false
	count := 0
	for _, entry := range env.accessSummaries {
		if entry.Function.Name == "Entry" && entry.Operation == AccessWrite && entry.Resolution == AccessOpaque && entry.Origin == FactOriginDeclaredForeign {
			opaque = true
		}
		if entry.Operation == AccessConsume && entry.Origin == FactOriginDeclaredForeign {
			foreignConsume = true
		}
		if entry.Function.Name == "Recursive" && entry.Operation == AccessWrite {
			count++
		}
	}
	if !opaque || !foreignConsume {
		t.Fatalf("foreign boundary lost conservative/declared origins: %#v", env.accessSummaries)
	}
	if count != 1 {
		t.Fatalf("recursive summary failed finite deduplication: %d entries", count)
	}
	queueType := env.functions["Entry"][0].Params[1].Type
	consumer := evt1DerivedSharedAccess(env, FactSingleConsumer, []Type{{Name: "Agent", Kind: TypeStruct}, queueType})
	if consumer.Outcome != FactProven {
		t.Fatalf("explicit foreign consume was not propagated: %#v", consumer)
	}
}

func TestR7d4FixedDisjointSlotsDoNotCreateFalseContradiction(t *testing.T) {
	source := `module Access.Disjoint;
profile Core;
record struct AgentA {}
record struct AgentB {}
[[execution_context(AgentA)]]
void WriteZero(ref int[2] slots)
{
    slots[0] = 1;
}
[[execution_context(AgentB)]]
void WriteOne(ref int[2] slots)
{
    slots[1] = 2;
}`
	env, _ := r7d4Env(t, source)
	arrayType := env.functions["WriteZero"][0].Params[0].Type
	result := evt1DerivedSharedAccess(env, FactExclusiveWriter, []Type{{Name: "AgentA", Kind: TypeStruct}, arrayType})
	if result.Outcome == FactDisproven || !strings.Contains(result.Evidence.Detail, "disjoint") {
		t.Fatalf("fixed disjoint indices falsely collided: %#v", result)
	}
}

func TestR7d4GenericParameterSubjectClosesAtCall(t *testing.T) {
	source := `module Access.Generic;
profile Core;
record struct Agent {}
struct Slot { int Value; }
template <typename T>
void CopyInto(ref T target, T source)
{
    target = source;
}
[[execution_context(Agent)]]
void Entry(ref Slot target, Slot source)
{
    CopyInto<Slot>(ref target, source);
}`
	env, _ := r7d4Env(t, source)
	result := evt1DerivedSharedAccess(env, FactExclusiveWriter, []Type{{Name: "Agent", Kind: TypeStruct}, {Name: "Slot", Kind: TypeStruct}})
	if result.Outcome != FactProven {
		t.Fatalf("generic subject did not close structurally: %#v", result)
	}
	for _, entry := range env.accessSummaries {
		if entry.Function.Name == "Entry" && strings.Contains(entry.Subject.Type.String(), "T") {
			t.Fatalf("closed caller summary retained template parameter: %#v", entry)
		}
	}
}
