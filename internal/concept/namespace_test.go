package concept

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestNamespacesQualifyCollidingSymbolsAcrossArtifacts(t *testing.T) {
	alpha := `module Alpha.Timer;
profile Core;
namespace Platform.Alpha
{
    int Timer() { return 20; }
}
`
	beta := `module Beta.Timer;
profile Core;
namespace Platform.Beta
{
    int Timer() { return 22; }
}
`
	artifacts := map[string][]byte{}
	artifacts["Alpha.Timer"] = buildSemanticArtifact(t, "Alpha/Timer.concept", alpha, nil)
	artifacts["Beta.Timer"] = buildSemanticArtifact(t, "Beta/Timer.concept", beta, nil)
	consumer := `module Namespace.Consumer;
profile Core;
import Alpha.Timer;
import Beta.Timer;
int Main()
{
    return Platform.Alpha.Timer() + Platform.Beta.Timer();
}
`
	module, err := ParseWithSemanticModules("Namespace/Consumer.concept", consumer, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	implementation := moduleOutput(t, outputs, ".generated.c")
	if !strings.Contains(implementation, "platform_alpha_timer") || !strings.Contains(implementation, "platform_beta_timer") {
		t.Fatalf("qualified namespace identities did not survive lowering:\n%s", implementation)
	}
	for run := 0; run < 100; run++ {
		again := buildSemanticArtifact(t, "Alpha/Timer.concept", alpha, nil)
		if !bytes.Equal(again, artifacts["Alpha.Timer"]) {
			t.Fatalf("namespace artifact changed on run %d", run)
		}
	}
}

func TestNamespaceUnqualifiedCollisionDiagnosesCandidates(t *testing.T) {
	a := buildSemanticArtifact(t, "One/Timer.concept", "module One.Timer; profile Core; namespace One { int Timer() { return 1; } }", nil)
	b := buildSemanticArtifact(t, "Two/Timer.concept", "module Two.Timer; profile Core; namespace Two { int Timer() { return 2; } }", nil)
	_, err := ParseWithSemanticModules("Bad.concept", "module Bad; profile Core; import One.Timer; import Two.Timer; int Main() { return Timer(); }", map[string][]byte{"One.Timer": a, "Two.Timer": b})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "NAMESPACE_SYMBOL_AMBIGUOUS" || !strings.Contains(diagnostic.Message, "One.Timer") || !strings.Contains(diagnostic.Message, "Two.Timer") {
		t.Fatalf("expected concise namespace ambiguity with candidates, got %v", err)
	}
}

func TestModuleDefaultNamespaceIsParentAndArtifactCarriesQualifiedIdentity(t *testing.T) {
	source := "module Standard.Memory.Bump; profile Core; struct BumpAllocator { int offset; }"
	body := buildSemanticArtifact(t, "Standard/Memory/Bump.concept", source, nil)
	artifact, _, err := LoadSemanticModuleArtifact(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifact.Exports.QualifiedSymbols) != 1 || artifact.Exports.QualifiedSymbols[0] != "Standard.Memory.BumpAllocator" {
		t.Fatalf("unexpected default namespace exports: %#v", artifact.Exports.QualifiedSymbols)
	}
}

func TestTwoModulesContributeToOneNamespace(t *testing.T) {
	start := buildSemanticArtifact(t, "Core/Start.concept", "module Core.Start; profile Core; namespace Shared.Core { int Start() { return 20; } }", nil)
	stop := buildSemanticArtifact(t, "Core/Stop.concept", "module Core.Stop; profile Core; namespace Shared.Core { int Stop() { return 22; } }", nil)
	consumer := "module Shared.Consumer; profile Core; import Core.Start; import Core.Stop; int Main() { return Shared.Core.Start() + Shared.Core.Stop(); }"
	module, err := ParseWithSemanticModules("Shared/Consumer.concept", consumer, map[string][]byte{"Core.Start": start, "Core.Stop": stop})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	implementation := moduleOutput(t, outputs, ".generated.c")
	if !strings.Contains(implementation, "start") || !strings.Contains(implementation, "stop") {
		t.Fatalf("shared namespace contributions were not both lowered:\n%s", implementation)
	}
}
