package concept

import (
	"errors"
	"strings"
	"testing"
)

const comptimeSubjectsSource = `module Subjects;
profile Core;

class Resource
{
public:
    int id;
};

void Drop(owned Resource resource)
{
}

class Holder
{
public:
    owned Resource kept;
    int count;
};

[[repr(C)]]
record struct Pair
{
    int left;
    int right;
};

template <typename T>
struct Box
{
    owned T value;
}

enum Verdict
{
    Holds,
    Refuted(declaration at, string message),
}

comptime Verdict DroppableFieldIsOwned(declaration field)
{
    typename t = compiler.TypeOf(field);
    if (compiler.Owned(field) or compiler.IsBorrowLike(t) or not compiler.HasDrop(t))
    {
        return Verdict::Holds;
    }
    return Verdict::Refuted(field, "field " + compiler.QualifiedName(field) + " holds " + compiler.TypeName(t));
}

comptime Verdict FieldsArePlain(declaration type)
{
    for (i in 0..compiler.FieldCount(type))
    {
        declaration field = compiler.Field(type, i);
        if (not compiler.IsScalar(compiler.TypeOf(field)))
        {
            return Verdict::Refuted(field, compiler.QualifiedName(field) + " is not plain data");
        }
    }
    return Verdict::Holds;
}

comptime string Describe(declaration type)
{
    string text = compiler.Name(type);
    if (compiler.IsClass(type)) { text = text + " class"; }
    if (compiler.IsRecord(type)) { text = text + " record"; }
    if (compiler.HasAttributeArgument(type, "repr", "C")) { text = text + " repr(C)"; }
    if (compiler.IsGenerated(type)) { text = text + " generated"; }
    return text;
}

comptime bool SameParent(declaration a, declaration b)
{
    return compiler.Parent(a) == compiler.Parent(b);
}

comptime bool OutOfRange(declaration type)
{
    return compiler.IsField(compiler.Field(type, 9));
}

int Use()
{
    owned Box<Resource> box = Box<Resource>{Resource{1}};
    return box.value.id;
}
`

func comptimeSubjectsEnv(t *testing.T) *semanticEnv {
	t.Helper()
	module, err := Parse("subjects.concept", comptimeSubjectsSource)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func subjectType(t *testing.T, env *semanticEnv, name string) Value {
	t.Helper()
	ref, ok := evt1TypeDeclarationRef(env, name)
	if !ok {
		t.Fatalf("no declaration for %s", name)
	}
	return evt1DeclarationValue(ref)
}

func subjectField(t *testing.T, env *semanticEnv, owner string, index int) Value {
	t.Helper()
	ref, ok := evt1FieldDeclarationRef(env, owner, index)
	if !ok {
		t.Fatalf("no field %d of %s", index, owner)
	}
	return evt1DeclarationValue(ref)
}

func TestComptimeSubjectPredicates(t *testing.T) {
	t.Parallel()
	env := comptimeSubjectsEnv(t)
	call := func(name string, args ...Value) Value {
		t.Helper()
		value, err := evt1InvokeComptimeFunction(env, name, args, Span{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return value
	}

	if got := call("DroppableFieldIsOwned", subjectField(t, env, "Holder", 0)); got.Variant != "Holds" {
		t.Fatalf("owned droppable field: %s", got.Render())
	}
	if got := call("DroppableFieldIsOwned", subjectField(t, env, "Holder", 1)); got.Variant != "Holds" {
		t.Fatalf("plain field: %s", got.Render())
	}
	if got := call("DroppableFieldIsOwned", subjectField(t, env, "Box<Resource>", 0)); got.Variant != "Holds" {
		t.Fatalf("generic instance field: %s", got.Render())
	}

	refuted := call("FieldsArePlain", subjectType(t, env, "Holder"))
	if refuted.Variant != "Refuted" || len(refuted.Payload) != 2 {
		t.Fatalf("Holder has a non-scalar field: %s", refuted.Render())
	}
	at := refuted.Payload[0].Declaration
	if at == nil || at.Kind != FieldDeclaration || at.qualifiedName() != "Holder.kept" {
		t.Fatalf("refutation site: %s", refuted.Payload[0].Render())
	}
	if refuted.Payload[1].StringValue != "Holder.kept is not plain data" {
		t.Fatalf("refutation message: %q", refuted.Payload[1].StringValue)
	}
	if got := call("FieldsArePlain", subjectType(t, env, "Pair")); got.Variant != "Holds" {
		t.Fatalf("Pair is plain: %s", got.Render())
	}

	for name, want := range map[string]string{
		"Holder":        "Holder class",
		"Pair":          "Pair record repr(C)",
		"Box<Resource>": "Box<Resource> generated",
	} {
		if got := call("Describe", subjectType(t, env, name)); got.StringValue != want {
			t.Fatalf("Describe(%s) = %q, want %q", name, got.StringValue, want)
		}
	}

	if got := call("SameParent", subjectField(t, env, "Holder", 0), subjectField(t, env, "Holder", 1)); !got.BoolValue {
		t.Fatal("fields of one type share a parent")
	}
	if got := call("SameParent", subjectField(t, env, "Holder", 0), subjectField(t, env, "Pair", 0)); got.BoolValue {
		t.Fatal("fields of different types do not share a parent")
	}

	_, err := evt1InvokeComptimeFunction(env, "OutOfRange", []Value{subjectType(t, env, "Pair")}, Span{})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "OBSERVATION_INVALID" {
		t.Fatalf("Field past the end must be OBSERVATION_INVALID, got %v", err)
	}
}

func TestComptimeSubjectsStayAtCompileTime(t *testing.T) {
	t.Parallel()
	module, err := Parse("subjects.concept", comptimeSubjectsSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(comptimeSubjectsSource))
	if err != nil {
		t.Fatalf("a module with comptime-only types must still generate: %v", err)
	}
	for name, body := range outputs {
		if strings.Contains(strings.ToLower(string(body)), "verdict") && strings.HasSuffix(name, ".c") {
			t.Fatalf("%s mentions the comptime-only Verdict enum", name)
		}
	}

	cases := []struct {
		name, body, code string
	}{
		{"runtime parameter", "int Bad(declaration d) { return 0; }\n", "COMPTIME_ONLY_TYPE"},
		{"runtime aggregate", "enum Wrapped { One(declaration d), }\nint Bad(Wrapped w) { return 0; }\n", "COMPTIME_ONLY_TYPE"},
		{"runtime observation", "class Thing { public: int id; };\nint Bad() { return compiler.FieldCount(Thing); }\n", "OBSERVATION_RUNTIME"},
		{"unknown observation", "comptime bool Bad(declaration d) { return compiler.IsAwesome(d); }\n", "OBSERVATION_UNKNOWN"},
		{"observation argument", "comptime bool Bad(declaration d) { return compiler.HasDrop(d); }\n", "OBSERVATION_ARGUMENTS"},
		{"observation arity", "comptime int Bad(declaration d) { return compiler.Field(d); }\n", "OBSERVATION_ARGUMENTS"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse("bad.concept", "module Bad;\nprofile Core;\n\n"+tc.body)
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}
