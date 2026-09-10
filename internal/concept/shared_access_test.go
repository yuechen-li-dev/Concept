package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const sharedAccessProofSource = `profile Core;
struct AgentA { int id; }
struct SlotX { int value; }

requires sync.ExclusiveWriter<AgentA, SlotX>;
requires sync.SynchronizedAccess<SlotX>;

concept BlackboardWriter<TAgent, TSlot>
{
    requires sync.ExclusiveWriter<TAgent, TSlot>;
    requires sync.SynchronizedAccess<TSlot>;
}

requires BlackboardWriter<AgentA, SlotX>;

void VerifySharedAccess()
{
    Assert.Concept<sync.ExclusiveWriter>(AgentA, SlotX, "blackboard slot has one declared writer");
    Assert.Concept<BlackboardWriter>(AgentA, SlotX, "named contract composes synchronization facts");
}
`

func TestSharedAccessContractsAreExplicitComposableAndProven(t *testing.T) {
	module, err := Parse("shared_access.concept", sharedAccessProofSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(sharedAccessProofSource))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal([]byte(moduleOutput(t, outputs, ".mir.json")), &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.ProofGraphs) != 2 {
		t.Fatalf("expected two synchronization proof graphs, got %d", len(mir.ProofGraphs))
	}
	for _, graph := range mir.ProofGraphs {
		if graph.Outcome != FactProven {
			t.Fatalf("expected proven graph, got %s: %s", graph.Outcome, RenderProofVerbose(graph))
		}
	}
	rendered := RenderProofVerbose(mir.ProofGraphs[1])
	if !strings.Contains(rendered, "sync.ExclusiveWriter") || !strings.Contains(rendered, "sync.SynchronizedAccess") || !strings.Contains(rendered, string(FactOriginDeclared)) {
		t.Fatalf("named proof omitted synchronization requirements or declared origin:\n%s", rendered)
	}
	found := false
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactExclusiveWriter && fact.Origin == FactOriginDeclared && len(fact.Subjects) == 2 && fact.Subjects[0].Type == "AgentA" && fact.Subjects[1].Type == "SlotX" {
			found = true
		}
	}
	if !found {
		t.Fatal("MIR omitted ordered declared ExclusiveWriter fact")
	}
	graph, err := ExplainSource("shared_access.concept", sharedAccessProofSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Outcome != FactProven || !strings.Contains(RenderProofVerbose(graph), string(FactOriginDeclared)) {
		t.Fatalf("concept explain omitted declared synchronization proof:\n%s", RenderProofVerbose(graph))
	}
	want, err := SerializeProof(graph)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		got, err := SerializeProof(graph)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("synchronization proof changed on run %d: %v", run, err)
		}
	}
}

func TestSharedAccessCompilerOwnedVocabularyIsOrdinarySourceSyntax(t *testing.T) {
	source := `profile Core;
struct Agent { int id; } struct Slot { int value; }
struct Producer { int id; } struct Consumer { int id; } struct Queue { int size; }
struct Write { int id; } struct Publish { int id; } struct Obligation { int id; }
requires sync.ExclusiveWriter<Agent, Slot>;
requires sync.SingleProducer<Producer, Queue>;
requires sync.SingleConsumer<Consumer, Queue>;
requires sync.SynchronizedAccess<Slot>;
requires sync.PublishedBefore<Write, Publish>;
requires sync.ExactlyOnce<Obligation>;
void Check()
{
    Assert.Concept<sync.SingleProducer>(Producer, Queue, "one producer authority");
    Assert.Concept<sync.SingleConsumer>(Consumer, Queue, "one consumer authority");
    Assert.Concept<sync.PublishedBefore>(Write, Publish, "publication visibility edge");
    Assert.Concept<sync.ExactlyOnce>(Obligation, "one internal claim authority");
}`
	if _, err := Parse("sync_vocabulary.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestSharedAccessContractsDistinguishUnknownDisprovenAndContradiction(t *testing.T) {
	unknown := `profile Core; struct AgentA { int id; } struct SlotX { int value; }
void Check() { Assert.Concept<sync.ExclusiveWriter>(AgentA, SlotX, "authority must be established"); }`
	if _, err := Parse("unknown_writer.concept", unknown); diagnosticCode(err) != "CONCEPT_ASSERT_UNKNOWN" {
		t.Fatalf("expected unknown synchronization proof, got %v", err)
	}
	disproven := `profile Core;
struct AgentA { int id; } struct AgentB { int id; } struct SlotX { int value; }
requires sync.ExclusiveWriter<AgentA, SlotX>;
void Check() { Assert.Concept<sync.ExclusiveWriter>(AgentB, SlotX, "a different writer conflicts"); }`
	if _, err := Parse("disproven_writer.concept", disproven); diagnosticCode(err) != "CONCEPT_ASSERT_DISPROVEN" {
		t.Fatalf("expected disproven synchronization proof, got %v", err)
	}
	contradiction := `profile Core;
struct AgentA { int id; } struct AgentB { int id; } struct SlotX { int value; }
requires sync.ExclusiveWriter<AgentA, SlotX>;
requires sync.ExclusiveWriter<AgentB, SlotX>;`
	if _, err := Parse("contradictory_writer.concept", contradiction); diagnosticCode(err) != "SYNC_AUTHORITY_CONTRADICTION" {
		t.Fatalf("expected declaration contradiction, got %v", err)
	}
}

func TestSharedAccessFactsRoundTripThroughSemanticArtifacts(t *testing.T) {
	library := `module Standard.Access;
profile Core;
struct Producer { int id; }
struct Queue { int size; }
requires sync.SingleProducer<Producer, Queue>;
requires sync.ExactlyOnce<Queue>;`
	body := buildSemanticArtifact(t, "Standard/Access.concept", library, nil)
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "Standard/Access.concept", library, nil); !bytes.Equal(got, body) {
			t.Fatalf("synchronization artifact changed on run %d", run)
		}
	}
	artifact, _, err := LoadSemanticModuleArtifact(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.SharedAccessFacts) != 2 || artifact.SharedAccessFacts[0].Origin != FactOriginDeclared {
		t.Fatalf("artifact omitted inspectable declared synchronization facts: %#v", artifact.SharedAccessFacts)
	}
	consumer := `module App;
profile Core;
import Standard.Access;
concept SafeQueue<TProducer, TQueue>
{
    requires sync.SingleProducer<TProducer, TQueue>;
    requires sync.ExactlyOnce<TQueue>;
}
requires SafeQueue<Producer, Queue>;
void Check() { Assert.Concept<SafeQueue>(Producer, Queue, "dependency contract must close without source"); }`
	module, err := ParseWithSemanticModules("App.concept", consumer, map[string][]byte{"Standard.Access": body})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	rendered := moduleOutput(t, outputs, ".mir.json")
	if !strings.Contains(rendered, string(FactOriginModuleFactSummary)) || !strings.Contains(rendered, string(FactSingleProducer)) {
		t.Fatalf("consumer MIR omitted imported synchronization authority:\n%s", rendered)
	}
}

func TestSharedAccessNamedContractTargetsLaterGenericParameters(t *testing.T) {
	source := `profile Core;
struct Value { int value; } struct Agent { int id; } struct Slot { int value; }
requires sync.ExclusiveWriter<Agent, Slot>;
concept WriterFor<TAgent, TSlot> { requires sync.ExclusiveWriter<TAgent, TSlot>; }
template <typename TValue, typename TAgent, typename TSlot>
requires WriterFor<TAgent, TSlot>
class Lease { public: TValue value; };
template <typename TValue, typename TAgent, typename TSlot>
requires WriterFor<TAgent, TSlot>
TValue Identity(TValue value) { return value; }
Lease<Value, Agent, Slot> MakeLease() { return Lease<Value, Agent, Slot>{Value{4}}; }
Value Use() { return Identity<Value, Agent, Slot>(Value{7}); }`
	if _, err := Parse("sync_generic_positions.concept", source); err != nil {
		t.Fatal(err)
	}
}
