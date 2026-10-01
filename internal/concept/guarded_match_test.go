package concept

import (
	"errors"
	"strings"
	"testing"
)

func guardedModule(body string) string {
	return "module Guarded;\nprofile Core;\n\n" + body
}

func TestGuardedMatchAtRuntimeAndCompileTime(t *testing.T) {
	t.Parallel()
	source := guardedModule(`comptime string Size(int n)
{
    return match
    {
        when n < 0 => "negative",
        when n == 0 => "zero",
        when n < 10 => "small",
        otherwise => "large",
    };
}

comptime int Classify(int n)
{
    int code = 0;
    match
    {
        when n % 2 == 0 => { code = 2; }
        when n % 3 == 0 => { code = 3; }
    }
    return code;
}

static_assert(Size(-4) == "negative", "first arm");
static_assert(Size(0) == "zero", "second arm");
static_assert(Size(7) == "small", "third arm");
static_assert(Size(70) == "large", "otherwise");
static_assert(Classify(4) == 2 and Classify(9) == 3 and Classify(7) == 0, "statement form, no otherwise");

int Sign(int n)
{
    return match
    {
        when n < 0 => -1,
        when n == 0 => 0,
        otherwise => 1,
    };
}

int Bucket(int n)
{
    int bucket = 0;
    match
    {
        when n < 10 => { bucket = 1; }
        when n < 100 => { bucket = 2; }
        otherwise => { bucket = 3; }
    }
    return bucket;
}
`)
	module, err := Parse("guarded.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatalf("guarded matches lower to C: %v", err)
	}
}

func TestGuardedMatchArmsAreTriedInOrder(t *testing.T) {
	t.Parallel()
	// The second arm would fail if evaluated: Field(type, 99) is out of
	// range. Arms after the first true guard are never evaluated.
	source := guardedModule(`comptime int First(int n)
{
    return match
    {
        when n > 0 => 1,
        when Fails(n) => 2,
        otherwise => 3,
    };
}

comptime bool Fails(int n)
{
    int<array>[1] one = [0];
    return one[n] == 0;
}

static_assert(First(5) == 1, "later guards are not evaluated");
`)
	if _, err := Parse("order.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestGuardedMatchShape(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, body, code string }{
		{"expression needs otherwise", "int F(int n) { return match { when n > 0 => 1, }; }\n", "MATCH_GUARD_OTHERWISE"},
		{"otherwise is last", "int F(int n) { return match { otherwise => 0, when n > 0 => 1, }; }\n", "MATCH_GUARD_ORDER"},
		{"needs a when arm", "int F(int n) { return match { otherwise => 0, }; }\n", "MATCH_GUARD_ARM"},
		{"arms are guards", "int F(int n) { return match { n > 0 => 1, otherwise => 0, }; }\n", "MATCH_GUARD_ARM"},
		{"guard is bool", "int F(int n) { return match { when n => 1, otherwise => 0, }; }\n", "CV4186"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse("shape.concept", guardedModule(tc.body))
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

const typeShapeSource = `extern "C" handle Device;

class Resource
{
public:
    int id;
};

record struct Pair
{
    int left;
    int right;
};

enum Mode { A, B, }

struct Sample
{
    int scalar;
    Device device;
    int<array>[4] fixed;
    owned Resource resource;
    Pair pair;
    Mode mode;
}

comptime string Describe(typename value)
{
    return match (compiler.Shape(value))
    {
        TypeShape::Scalar(name) => "scalar " + name,
        TypeShape::Handle(name) => "handle " + name,
        TypeShape::FixedArray(element) => "array of " + compiler.TypeName(element),
        TypeShape::RuntimeArray(element) => "runtime array",
        TypeShape::Record(record) => "record " + compiler.Name(record),
        TypeShape::Struct(aggregate) => "struct " + compiler.Name(aggregate),
        TypeShape::Enum(decl) => "enum " + compiler.Name(decl),
        TypeShape::Owned(target) => "owned " + Describe(target),
        TypeShape::Reference(target) => "reference",
        TypeShape::Pointer(target) => "pointer",
        TypeShape::Dyn(spelling) => "dyn",
        TypeShape::Callable(spelling) => "callable",
        TypeShape::Async(spelling) => "async",
        TypeShape::Generic(applied) => "generic",
        TypeShape::Other(spelling) => "other " + spelling,
    };
}
`

func TestTypeShapeClassifiesTypes(t *testing.T) {
	t.Parallel()
	source := guardedModule(strings.Replace(typeShapeSource, "comptime string Describe(typename value)", "comptime string Describe(typename value) bounded(4)", 1))
	module, err := Parse("shape.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"scalar int", "handle Device", "array of int", "owned class Resource", "record Pair", "enum Mode"}
	want[3] = "owned struct Resource"
	for i, expected := range want {
		ref, ok := evt1FieldDeclarationRef(env, "Sample", i)
		if !ok {
			t.Fatalf("field %d", i)
		}
		decl := env.structs["Sample"]
		value, err := evt1InvokeComptimeFunction(env, "Describe", []Value{evt1TypenameValue(decl.Fields[ref.Index].Type)}, Span{})
		if err != nil {
			t.Fatal(err)
		}
		if value.StringValue != expected {
			t.Fatalf("field %s: got %q, want %q", ref.Name, value.StringValue, expected)
		}
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatalf("TypeShape stays at compile time: %v", err)
	}
}

func TestTypeShapeMatchMustBeExhaustive(t *testing.T) {
	t.Parallel()
	source := guardedModule(strings.Replace(typeShapeSource, "        TypeShape::Other(spelling) => \"other \" + spelling,\n", "", 1))
	_, err := Parse("shape.concept", source)
	if err == nil || !strings.Contains(err.Error(), "Other") {
		t.Fatalf("a match on TypeShape that omits a shape must not compile, got %v", err)
	}
}

func TestTypeShapeIsCompileTimeOnly(t *testing.T) {
	t.Parallel()
	_, err := Parse("shape.concept", guardedModule("int Bad(TypeShape shape) { return 0; }\n"))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "COMPTIME_ONLY_TYPE" {
		t.Fatalf("expected COMPTIME_ONLY_TYPE, got %v", err)
	}
}
