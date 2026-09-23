package concept

import "testing"

func TestClosedRequiredOperationCanUseUnconstrainedGenericWitness(t *testing.T) {
	source := `profile Core;
template <typename T> struct Wrapped { T value; };
template <typename T> void Read(T context, Wrapped<T> wrapped) { }
concept Readable<W, T> { requires void Read(T context, W wrapped); }
requires Readable<Wrapped<int>, int>;
`
	module, err := Parse("closed_generic_requirement.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}
