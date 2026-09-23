package concept

import "testing"

func TestGeneratedPayloadEnumMatchUsesOrdinaryCheckingAndMIR(t *testing.T) {
	source := `module GeneratedEnum; profile Core;
enum Choice {
    None,
    One([[selected]] int value),
    Pair([[selected]] int left, int ignored, [[selected]] int right),
}
generator <typename T> DeriveSum
int Sum(ref const T item) {
    int result = 0;
    foreach (EnumCaseInfo option in Cases<T>()) {
        foreach (FieldInfo field in Payload<option>(selected)) {
            result = result + field;
        }
    }
    return result;
}
derive DeriveSum reflect<Choice>;
int Main() { Choice item = Choice::Pair(1, 99, 2); return Sum(ref const item); }
`
	module, err := Parse("generated_enum.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Functions) != 2 || module.Functions[1].Generated == nil || len(module.Functions[1].Generated.Inputs) != 3 {
		t.Fatalf("generated enum inputs: %+v", module.Functions)
	}
	if _, ok := module.Functions[1].Body.Statements[1].(*MatchStmt); !ok {
		t.Fatalf("generated enum body has no ordinary match: %+v", module.Functions[1].Body)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}
