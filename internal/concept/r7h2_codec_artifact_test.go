package concept

import (
	"bytes"
	"testing"
)

func TestR7h2GeneratedCodecFamilyAcrossArtifacts(t *testing.T) {
	artifacts := map[string][]byte{}
	artifacts["Standard.Octagon.Core"] = buildSemanticArtifact(t,
		"Standard/Octagon/Core.concept", standardMemorySource(t, "Standard/Octagon/Core.concept"), artifacts)
	artifacts["Standard.Octagon.Derive"] = buildSemanticArtifact(t,
		"Standard/Octagon/Derive.concept", standardMemorySource(t, "Standard/Octagon/Derive.concept"), artifacts)
	const sourceB = `module CodecB; profile Core;
import Standard.Octagon.Core; import Standard.Octagon.Derive;
[[reflect]] record struct Packet { int<array>[4] Values; }
[[reflect]] enum Mode { Enabled(int level), }
[[reflect]] record table<2> Grid { int ID; bool Active; }
[[reflect]] record struct Positive { int Value; }
Result<Positive, OctagonError> AdmitOctagon(int raw, OctagonType<Positive> type) {
    if (raw <= 0) { return Result::Error(OctagonError::UnexpectedData); }
    return Result::Ok(Positive{Value = raw});
}
derive DeriveOctagonRead reflect<Packet>;
derive DeriveOctagonWrite reflect<Packet>;
derive DeriveOctagonEnumRead reflect<Mode>;
derive DeriveOctagonEnumWrite reflect<Mode>;
derive DeriveOctagonRead reflect<Grid>;
derive DeriveOctagonWrite reflect<Grid>;
derive DeriveOctagonRefinedRead reflect<Positive>;
derive DeriveOctagonRefinedWrite reflect<Positive>;
requires OctagonCodec<Packet>;
requires OctagonCodec<Mode>;
requires OctagonCodec<Grid>;
requires OctagonCodec<Positive>;
`
	b, err := CompileSemanticModule("CodecB.concept", sourceB, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["CodecB"] = b
	const sourceC = `module CodecC; profile Core;
import Standard.Octagon.Core; import CodecB;
int Main() {
    Packet input = Packet{Values = [1, 2, 3, 4]};
    byte<array>[80] bytes = [0 ...];
    OctagonWriter writer = OctagonWriter{Span(bytes), 0};
    WriteOctagon(ref const input, ref writer)!;
    FinishOctagon(ref writer)!;
    OctagonReader reader = OctagonReader{Span(bytes), 0};
    Packet loaded = ReadOctagon(ref reader, OctagonType<Packet>{0})!;
    Mode mode = Mode::Enabled(7);
    writer.offset = 0;
    WriteOctagon(ref const mode, ref writer)!;
    OctagonReader modeReader = OctagonReader{Span(bytes), 0};
    Mode modeLoaded = ReadOctagon(ref modeReader, OctagonType<Mode>{0})!;
    int level = match (modeLoaded) { Mode::Enabled(n) => n, };
    Grid grid = Grid{ID = [3, 4], Active = [true, false]};
    writer.offset = 0;
    WriteOctagon(ref const grid, ref writer)!;
    OctagonReader gridReader = OctagonReader{Span(bytes), 0};
    Grid gridLoaded = ReadOctagon(ref gridReader, OctagonType<Grid>{0})!;
    Positive positive = AdmitOctagon(9, OctagonType<Positive>{0})!;
    writer.offset = 0;
    WriteOctagon(ref const positive, ref writer)!;
    OctagonReader positiveReader = OctagonReader{Span(bytes), 0};
    Positive positiveLoaded = ReadOctagon(ref positiveReader, OctagonType<Positive>{0})!;
    return loaded.Values[0] + loaded.Values[3] + level + gridLoaded.ID[1] + positiveLoaded.Value;
}
`
	module, err := ParseWithSemanticModules("CodecC.concept", sourceC, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(sourceC))
	if err != nil {
		t.Fatal(err)
	}
	derived, err := ParseWithSemanticModules("CodecB.concept", sourceB, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := ExplainGeneratedConcept(derived, "OctagonCodec<Packet>")
	if err != nil {
		t.Fatal(err)
	}
	proofBytes, err := SerializeProof(proof)
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		currentB, err := CompileSemanticModule("CodecB.concept", sourceB, artifacts)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, currentB) {
			t.Fatalf("codec artifact changed on run %d", run+1)
		}
		currentArtifacts := map[string][]byte{
			"Standard.Octagon.Core":   artifacts["Standard.Octagon.Core"],
			"Standard.Octagon.Derive": artifacts["Standard.Octagon.Derive"],
			"CodecB":                  currentB,
		}
		currentModule, err := ParseWithSemanticModules("CodecC.concept", sourceC, currentArtifacts)
		if err != nil {
			t.Fatal(err)
		}
		currentOutputs, err := Generate(currentModule, []byte(sourceC))
		if err != nil {
			t.Fatal(err)
		}
		if len(currentOutputs) != len(outputs) {
			t.Fatalf("output count changed on run %d", run+1)
		}
		for path, initial := range outputs {
			if !bytes.Equal(initial, currentOutputs[path]) {
				t.Fatalf("output %s changed on run %d", path, run+1)
			}
		}
		currentDerived, err := ParseWithSemanticModules("CodecB.concept", sourceB, currentArtifacts)
		if err != nil {
			t.Fatal(err)
		}
		currentProof, err := ExplainGeneratedConcept(currentDerived, "OctagonCodec<Packet>")
		if err != nil {
			t.Fatal(err)
		}
		currentProofBytes, err := SerializeProof(currentProof)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(proofBytes, currentProofBytes) {
			t.Fatalf("concept explain changed on run %d", run+1)
		}
	}
	runFoundationNativeHarness(t, outputs, "r7h2_codec_artifact_harness.c", "#include \"codecc.generated.h\"\nint main(void) { return concept_codec_c_main() == 25 ? 0 : 1; }\n")
}
