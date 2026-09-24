package concept

import (
	"strconv"
	"strings"
	"testing"
)

func TestR7h2NestedGeneratedCodecErrorsRetainPath(t *testing.T) {
	artifacts := map[string][]byte{}
	artifacts["Standard.Octagon.Core"] = buildSemanticArtifact(t,
		"Standard/Octagon/Core.concept", standardMemorySource(t, "Standard/Octagon/Core.concept"), artifacts)
	artifacts["Standard.Octagon.Derive"] = buildSemanticArtifact(t,
		"Standard/Octagon/Derive.concept", standardMemorySource(t, "Standard/Octagon/Derive.concept"), artifacts)
	const bad = "Envelope {\n    Items: [Mode.Enabled(1), Mode.Enabled(no)]\n}\n"
	items := make([]string, len(bad))
	for index := range bad {
		items[index] = strconv.Itoa(int(bad[index]))
	}
	source := strings.ReplaceAll(`module R7h2ErrorPath; profile Core;
import Standard.Octagon.Core; import Standard.Octagon.Derive;
enum Mode { Enabled(int level), }
record struct Envelope { Mode<array>[2] Items; }
derive DeriveOctagonEnumRead reflect<Mode>;
derive DeriveOctagonEnumWrite reflect<Mode>;
derive DeriveOctagonRead reflect<Envelope>;
derive DeriveOctagonWrite reflect<Envelope>;

bool HasExpectedPath(OctagonError error, int elementIndex, OctagonErrorKind expectedKind) {
    match (error) {
        OctagonError::At(kind, parts, depth) => {
            if (kind != expectedKind) { return false; }
            if (depth != 6) { return false; }
            if (not OctagonPathNameIs(parts[0], "Envelope")) { return false; }
            if (not OctagonPathNameIs(parts[1], "Items")) { return false; }
            if (not OctagonPathIndexIs(parts[2], elementIndex)) { return false; }
            if (not OctagonPathNameIs(parts[3], "Mode")) { return false; }
            if (not OctagonPathNameIs(parts[4], "Enabled")) { return false; }
            return OctagonPathNameIs(parts[5], "level");
        }
        OctagonError::EndOfInput => { return false; }
        OctagonError::UnexpectedData => { return false; }
        OctagonError::CapacityExceeded => { return false; }
    }
}

bool HasCasePath(OctagonError error) {
    match (error) {
        OctagonError::At(kind, parts, depth) => {
            return kind == OctagonErrorKind::CapacityExceeded and depth == 2 and
                OctagonPathNameIs(parts[0], "Mode") and OctagonPathNameIs(parts[1], "Enabled");
        }
        OctagonError::EndOfInput => { return false; }
        OctagonError::UnexpectedData => { return false; }
        OctagonError::CapacityExceeded => { return false; }
    }
}

int CheckReader() {
    byte<array>[{{LEN}}] bytes = [{{BYTES}}];
    OctagonReader reader = OctagonReader{Span(bytes), 0};
    Result<Envelope, OctagonError> attempt = ReadOctagon(ref reader, OctagonType<Envelope>{0});
    match (move attempt) {
        Result::Ok(value) => { return 0; }
        Result::Error(error) => { if (HasExpectedPath(error, 1, OctagonErrorKind::UnexpectedData)) { return 1; } return 0; }
    }
}

int CheckWriter() {
    Envelope value = Envelope{Items = [Mode::Enabled(1), Mode::Enabled(2)]};
    byte<array>[36] bytes = [0 ...];
    OctagonWriter writer = OctagonWriter{Span(bytes), 0};
    Result<void, OctagonError> attempt = WriteOctagon(ref const value, ref writer);
    match (move attempt) {
        Result::Ok() => { return 0; }
        Result::Error(error) => { if (HasExpectedPath(error, 0, OctagonErrorKind::CapacityExceeded)) { return 1; } return 0; }
    }
}

int CheckEnumLabelWriter() {
    Mode value = Mode::Enabled(3);
    byte<array>[2] bytes = [0 ...];
    OctagonWriter writer = OctagonWriter{Span(bytes), 0};
    Result<void, OctagonError> attempt = WriteOctagon(ref const value, ref writer);
    match (move attempt) {
        Result::Ok() => { return 0; }
        Result::Error(error) => { if (HasCasePath(error)) { return 1; } return 0; }
    }
}

int Main() { return CheckReader() + CheckWriter() + CheckEnumLabelWriter(); }
`, "{{LEN}}", strconv.Itoa(len(bad)))
	source = strings.ReplaceAll(source, "{{BYTES}}", strings.Join(items, ", "))
	module, err := ParseWithSemanticModules("R7h2ErrorPath.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r7h2_error_path_harness.c", "#include \"r7h2errorpath.generated.h\"\nint main(void) { return concept_r7h2error_path_main() == 3 ? 0 : 1; }\n")
}
