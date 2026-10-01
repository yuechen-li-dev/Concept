package concept

import (
	"errors"
	"testing"
)

func TestR9aStaticValidityAgreement(t *testing.T) {
	cases := []struct{ source, code string }{
		{"unsafe int F() { return 1; }", "TYPE_QUALIFIER_UNSUPPORTED"},
		{"imported int F() { return 1; }", "TYPE_QUALIFIER_UNSUPPORTED"},
		{"int F(owned ref int value) { return 1; }", "OWNERSHIP_QUALIFIER_CONFLICT"},
		{"int F(const const int value) { return value; }", "TYPE_QUALIFIER_DUPLICATE"},
		{"struct Item { int value; } void F(scoped Item item) {}", "CV4525"},
		{"struct Item { [[repr(C)]] int value; }", "ATTRIBUTE_POSITION_INVALID"},
		{"enum Item { [[must_use]] Ready }", "ATTRIBUTE_POSITION_INVALID"},
		{"template <typename T> struct Item { [[reflect]] T value; };", "ATTRIBUTE_POSITION_INVALID"},
		{"[[decorative]] struct Item { int value; }", "REFLECT_ATTRIBUTE_INVALID"},
		{"[[decorative]] void F() {}", "TEST_ATTRIBUTE_UNKNOWN"},
		{"[[reflect(1)]] struct Item { int value; }", "REFLECT_ATTRIBUTE_INVALID"},
		{"[[repr(Native)]] record struct Item { int value; }", "C_ABI_REPR_INVALID"},
		{"[[diagnostic(\"OWN\")]] concept Own<declaration D> { requires compiler.Authored(D); }", "CONCEPT_ATTRIBUTE_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			source := "module Closure.Audit; profile Core;\n" + tc.source
			check := func(err error) {
				var diagnostic Diagnostic
				if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
					t.Fatalf("expected %s, got %v", tc.code, err)
				}
			}
			_, err := Parse("audit.concept", source)
			check(err)
			_, err = CompileSemanticModule("audit.concept", source, nil)
			check(err)
			module, err := parseSyntaxModule("audit.concept", source)
			if err == nil {
				_, err = Generate(module, []byte(source))
			}
			check(err)
		})
	}
}
