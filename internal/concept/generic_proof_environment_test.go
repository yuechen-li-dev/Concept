package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// This fixture deliberately uses no library-specific names. The imported
// generic's access contract must survive artifact transport and close at the
// consumer's concrete application.
func TestR7d6ImportedGenericAccessProof(t *testing.T) {
	library := `module Proof.Channel;
profile Core;
record struct AgentA {}
template <typename TQueue>
class Channel {
public:
    [[execution_context(AgentA)]]
    [[semantic_access("Publish", self)]]
    void Send(ref Channel self) {}
};`
	body := buildSemanticArtifact(t, "Proof/Channel.concept", library, nil)
	artifact, _, loadErr := LoadSemanticModuleArtifact(body)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(artifact.AccessSummaries) == 0 || artifact.AccessSummaries[0].OpenGenericOwner != "Channel" {
		t.Fatalf("missing symbolic generic access summary: %#v", artifact.AccessSummaries)
	}
	consumer := `module Proof.Consumer;
profile Core;
import Proof.Channel;
record struct QueueX {}
struct Use { Channel<QueueX> channel; }
`
	module, err := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	proof := evt1DerivedSharedAccess(env, FactSingleProducer, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Channel<QueueX>", Kind: TypeStruct}})
	if proof.Outcome != FactProven {
		t.Fatalf("closed imported producer proof: %#v", proof)
	}
	if proof.Origin != FactOriginDerivedAccessSummary {
		t.Fatalf("substitution changed derived proof provenance: %#v", proof)
	}
	first, err := json.Marshal(env.accessSummaries)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		again, parseErr := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		againEnv, analyzeErr := analyzeModule(again)
		if analyzeErr != nil {
			t.Fatal(analyzeErr)
		}
		encoded, marshalErr := json.Marshal(againEnv.accessSummaries)
		if marshalErr != nil || !bytes.Equal(first, encoded) {
			t.Fatalf("closed proof environment changed on run %d: %v", run, marshalErr)
		}
	}
}

func TestR7d6GenericInstancesKeepIndependentProducerSets(t *testing.T) {
	library := `module Proof.Channel;
profile Core;
record struct AgentA {}
record struct AgentB {}
template <typename TQueue>
class Channel {
public:
    [[execution_context(AgentA)]]
    [[semantic_access("Publish", self)]]
    void Send(ref Channel self) {}
};`
	body := buildSemanticArtifact(t, "Proof/Channel.concept", library, nil)
	consumer := `module Proof.Consumer;
profile Core;
import Proof.Channel;
record struct QueueX {}
record struct QueueY {}
struct Use { Channel<QueueX> first; Channel<QueueY> second; }
[[execution_context(AgentB)]]
[[semantic_access("Publish", other)]]
void Compete(ref Channel<QueueY> other) {}
`
	module, err := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		subject string
		outcome SemanticFactCertainty
	}{{"Channel<QueueX>", FactProven}, {"Channel<QueueY>", FactDisproven}} {
		proof := evt1DerivedSharedAccess(env, FactSingleProducer, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: check.subject, Kind: TypeStruct}})
		if proof.Outcome != check.outcome {
			t.Fatalf("%s: got %#v, want %s", check.subject, proof, check.outcome)
		}
	}
}

func TestR7d6GenericValueArgumentAndOpaqueProducer(t *testing.T) {
	library := `module Proof.Channel;
profile Core;
record struct AgentA {}
template <typename TQueue, usize Capacity>
class Channel {
public:
    [[execution_context(AgentA)]]
    [[semantic_access("Publish", self)]]
    void Send(ref Channel self) {}
};`
	body := buildSemanticArtifact(t, "Proof/Channel.concept", library, nil)
	consumer := `module Proof.Consumer;
profile Core;
import Proof.Channel;
record struct QueueX {}
struct Use { Channel<QueueX, 16> channel; }
[[semantic_access("Publish", queue)]]
void Opaque(ref Channel<QueueX, 16> queue) {}
`
	module, err := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	proof := evt1DerivedSharedAccess(env, FactSingleProducer, []Type{{Name: "AgentA", Kind: TypeStruct}, {Name: "Channel<QueueX, 16>", Kind: TypeStruct}})
	if proof.Outcome != FactUnknown {
		t.Fatalf("opaque concrete producer must stay Unknown: %#v", proof)
	}
	closed := false
	for _, entry := range env.accessSummaries {
		if entry.Instance == "Channel<QueueX, 16>" && entry.Subject.Type.String() == "Channel<QueueX, 16>" {
			closed = true
		}
	}
	if !closed {
		t.Fatalf("non-type argument did not close in imported access identity: %#v", env.accessSummaries)
	}
}

func TestR7d6GenericExecutionContextClosesFromArtifact(t *testing.T) {
	library := `module Proof.Channel;
profile Core;
record struct AgentA {}
record struct AgentB {}
template <typename TProducer, typename TConsumer, typename TQueue>
class Channel {
public:
    [[execution_context(TProducer)]]
    [[semantic_access("Publish", self)]]
    void Send(ref Channel self) {}
    [[execution_context(TConsumer)]]
    [[semantic_access("Consume", self)]]
    void Receive(ref Channel self) {}
};`
	body := buildSemanticArtifact(t, "Proof/Channel.concept", library, nil)
	consumer := `module Proof.Consumer;
profile Core;
import Proof.Channel;
record struct QueueX {}
struct Use { Channel<AgentA, AgentB, QueueX> channel; }
`
	module, err := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		kind    SemanticFactKind
		context string
	}{{FactSingleProducer, "AgentA"}, {FactSingleConsumer, "AgentB"}} {
		proof := evt1DerivedSharedAccess(env, check.kind, []Type{{Name: check.context, Kind: TypeStruct}, {Name: "Channel<AgentA, AgentB, QueueX>", Kind: TypeStruct}})
		if proof.Outcome != FactProven {
			t.Fatalf("generic execution context did not close %s: %#v", check.kind, proof)
		}
	}
	closed := 0
	for _, entry := range env.accessSummaries {
		if entry.Instance == "Channel<AgentA, AgentB, QueueX>" {
			closed++
			if entry.Context.Name == "TProducer" || entry.Context.Name == "TConsumer" || entry.Context.Type.String() != entry.Context.Name {
				t.Fatalf("closed instance retained symbolic execution context: %#v", entry.Context)
			}
		}
	}
	if closed != 2 {
		t.Fatalf("want two closed generic access entries, got %d", closed)
	}
}

func TestR7d6ConcreteContractContradiction(t *testing.T) {
	library := `module Proof.Channel;
profile Core;
record struct AgentA {}
record struct AgentB {}
template <typename TQueue>
class Channel {
public:
    [[execution_context(AgentA)]]
    [[semantic_access("Publish", self)]]
    void Send(ref Channel self) {}
};`
	body := buildSemanticArtifact(t, "Proof/Channel.concept", library, nil)
	consumer := `module Proof.Consumer;
profile Core;
import Proof.Channel;
record struct QueueY {}
requires sync.SingleProducer<AgentA, Channel<QueueY>>;
[[execution_context(AgentB)]]
[[semantic_access("Publish", other)]]
void Compete(ref Channel<QueueY> other) {}
`
	_, err := ParseWithSemanticModules("Proof/Consumer.concept", consumer, map[string][]byte{"Proof.Channel": body})
	if err == nil || !strings.Contains(err.Error(), "SYNC_AUTHORITY_CONTRADICTION") {
		t.Fatalf("declared concrete SPSC authority must reject MPSC access: %v", err)
	}
}

func TestR7d6PlannerScopesGenericGuardEvidence(t *testing.T) {
	entry := func(instance, context string, operation AccessKind) MIRAccessEntry {
		owner := Type{Name: instance, Kind: TypeStruct}
		return evt1FinalizeAccessEntry(MIRAccessEntry{
			Instance: instance, Operation: operation, Origin: FactOriginModuleAccessSummary,
			Subject:    AccessSubject{Root: AccessIdentity{Kind: "Parameter", Type: owner}, Type: owner, Resolution: AccessExact},
			Context:    AccessIdentity{Kind: "ExecutionContext", Name: context, Type: Type{Name: context, Kind: TypeStruct}},
			Resolution: AccessExact,
		})
	}
	accesses := []MIRAccessEntry{
		entry("Channel<QueueX>", "AgentA", AccessWrite),
		entry("Channel<QueueY>", "AgentA", AccessWrite),
		entry("Channel<QueueY>", "AgentB", AccessRead),
	}
	if !evt1GuardsSimplifiableForOwner("Channel<QueueX>", accesses) || evt1GuardsSimplifiableForOwner("Channel<QueueY>", accesses) || evt1GuardsSimplifiableForOwner("Channel<QueueZ>", accesses) {
		t.Fatal("concrete guard evidence leaked between generic instances")
	}
	functions := []MIRFunction{}
	for _, instance := range []string{"Channel<QueueX>", "Channel<QueueY>", "Channel<QueueZ>"} {
		functions = append(functions, MIRFunction{Name: "Send", MethodOf: instance, Operations: []MIROperation{{ID: "call-" + instance, Kind: "call", Synchronization: "Acquire"}}})
	}
	mir := MIR{Schema: "concept-evt1-mir.v1", Module: "Proof.Scope", Profile: "Core", Functions: functions, AccessSummaries: accesses}
	facts := NewSemanticFactSet(nil)
	plan, err := PlanModule(&mir, &facts, GenericC11Target(), ProfileDefinition{Name: "Core"}, OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for i, expected := range []string{"ElideSynchronization", "RetainSynchronization", "RetainSynchronization"} {
		if got := plan.Functions[i].Decisions[0].Strategy; got != expected {
			t.Fatalf("%s: got %s, want %s", functions[i].MethodOf, got, expected)
		}
	}
	verified, err := PlanModule(&mir, &facts, GenericC11Target(), ProfileDefinition{Name: "Core"}, ConservativeCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	for _, function := range verified.Functions {
		if function.Decisions[0].Strategy != "RetainSynchronization" {
			t.Fatalf("verification elided %s guard", function.Function)
		}
	}
}
