package concept

import "testing"

func TestGeneratedReflectionProbePreservesExactCallableInference(t *testing.T) {
	source := `module GeneratedCallable; profile Core;
struct Item { int value; }
auto MakeCallback() { return callback() { return 3; }; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }
derive DeriveValue reflect<Item>;
int Main() { Item item = Item{2}; auto cb = MakeCallback(); return ReadValue(ref const item) + cb(); }
`
	module, err := Parse("generated_callable.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}
