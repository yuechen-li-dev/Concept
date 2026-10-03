package concept

import (
	"fmt"
	"strings"
	"testing"
)

func TestR9b3GeometryObservationBounds(t *testing.T) {
	var source strings.Builder
	source.WriteString(`module Research.Geometry;
profile Core;
record struct Cycle
{
    Cycle next;
}
record struct Level0
{
    int value;
}
using Huge = uint64<ndarray>[1048575, 1048575, 1048575, 1048575];
comptime bool GeometryAvailable(typename subject)
{
    return compiler.HasFixedGeometry(subject);
}
`)
	for level := 1; level <= 24; level++ {
		fmt.Fprintf(&source, "record struct Level%d\n{\n    Level%d left;\n    Level%d right;\n}\n", level, level-1, level-1)
	}
	module, err := Parse("geometry.concept", source.String())
	if err == nil || !strings.Contains(err.Error(), "CV4129") {
		t.Fatalf("recursive by-value type must be rejected before geometry: %v", err)
	}
	acyclic := strings.Replace(source.String(), "record struct Cycle\n{\n    Cycle next;\n}\n", "", 1)
	module, err = Parse("geometry.concept", acyclic)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range []Type{env.typeAliases["Huge"]} {
		result, err := evt1InvokeComptimeFunction(env, "GeometryAvailable", []Value{evt1TypenameValue(subject)}, Span{})
		if err != nil || result.Kind != ValueBool || result.BoolValue {
			t.Fatalf("unavailable geometry: %+v %v", result, err)
		}
	}
	query := newEVT1GeometryQuery()
	size, align, err := evt1TypeGeometryWithin(env, Type{Name: "Level24", Kind: TypeStruct}, query)
	if err != nil || size != 4*(1<<24) || align != 4 || query.accesses > 55 {
		t.Fatalf("bounded graph query: size=%d align=%d accesses=%d %v", size, align, query.accesses, err)
	}
	t.Logf("24-level repeated-field graph: geometry accesses=%d (memoized within the query); overflow and recursive geometry unavailable", query.accesses)
}
