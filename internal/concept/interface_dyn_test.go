package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var interfaceDynConformanceCases = []struct {
	file       string
	accepted   bool
	diagnostic string
}{
	{"valid/class_basic.concept", true, ""},
	{"valid/class_private_field_method.concept", true, ""},
	{"valid/class_public_method_call.concept", true, ""},
	{"valid/class_copyable_value_semantics.concept", true, ""},
	{"valid/class_owned_field_move_semantics.concept", true, ""},
	{"valid/interface_method_struct_satisfies.concept", true, ""},
	{"valid/interface_method_class_satisfies.concept", true, ""},
	{"valid/interface_field_struct_satisfies.concept", true, ""},
	{"valid/interface_composition.concept", true, ""},
	{"valid/interface_static_template_use.concept", true, ""},
	{"valid/dyn_struct_method_dispatch.concept", true, ""},
	{"valid/dyn_class_method_dispatch.concept", true, ""},
	{"valid/dyn_field_get.concept", true, ""},
	{"valid/dyn_field_set.concept", true, ""},
	{"valid/dyn_immovable_ref.concept", true, ""},
	{"valid/dyn_scoped_provenance.concept", true, ""},
	{"invalid/class_private_field_access.concept", false, "CLASS_PRIVATE_MEMBER_ACCESS"},
	{"invalid/class_private_method_access.concept", false, "CLASS_PRIVATE_MEMBER_ACCESS"},
	{"invalid/class_duplicate_member.concept", false, "CLASS_DUPLICATE_MEMBER"},
	{"invalid/interface_missing_method.concept", false, "CV4153"},
	{"invalid/interface_wrong_method_signature.concept", false, "CV4156"},
	{"invalid/interface_missing_field.concept", false, "INTERFACE_REQUIREMENT_UNSATISFIED"},
	{"invalid/interface_private_member_not_satisfy.concept", false, "INTERFACE_PRIVATE_MEMBER_CANNOT_SATISFY"},
	{"invalid/interface_not_dyn_compatible.concept", false, "INTERFACE_NOT_DYN_COMPATIBLE"},
	{"invalid/dyn_noninterface.concept", false, "DYN_REQUIRES_INTERFACE"},
	{"invalid/dyn_unsatisfied_interface.concept", false, "DYN_CONCRETE_TYPE_DOES_NOT_SATISFY"},
	{"invalid/dyn_mutable_from_const.concept", false, "DYN_MUTABLE_FROM_CONST"},
	{"invalid/dyn_escape_local.concept", false, "CV4511"},
	{"invalid/dyn_unknown_method.concept", false, "DYN_METHOD_NOT_IN_INTERFACE"},
	{"invalid/dyn_readonly_field_mutation.concept", false, "DYN_READONLY_FIELD_MUTATION"},
}

func generateInterfaceDynFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "interface", class, file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}

func TestInterfaceDynConformance(t *testing.T) {
	for _, tc := range interfaceDynConformanceCases {
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "interface", filepath.FromSlash(tc.file))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			if tc.accepted {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.diagnostic {
				t.Fatalf("diagnostic = %v, want %s", err, tc.diagnostic)
			}
		})
	}
}

func TestInterfaceDynClassMembersDefaultPrivate(t *testing.T) {
	source := []byte("profile Core; class Secret { int value; } int Read(ref Secret secret) { return secret.value; }")
	module, err := Parse("class_default_private.concept", string(source))
	if err == nil {
		_, err = Generate(module, source)
	}
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CLASS_PRIVATE_MEMBER_ACCESS" {
		t.Fatalf("diagnostic = %v", err)
	}
}

func TestInterfaceDynWitnessMIRAndCShape(t *testing.T) {
	outputs := generateInterfaceDynFixture(t, "valid", "class_interface_dyn.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["class_interface_dyn.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Witnesses) != 5 {
		t.Fatalf("witness count = %d", len(mir.Witnesses))
	}
	c := string(outputs["class_interface_dyn.generated.c"])
	h := string(outputs["class_interface_dyn.generated.h"])
	for _, required := range []string{"static const concept_concept_drawable_witness", ".witness->Draw", ".witness->get_x", ".witness->set_x", ".object = (void*)&(circle)"} {
		if !strings.Contains(c, required) {
			t.Fatalf("generated C missing %q", required)
		}
	}
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "typeid", "dynamic_cast", "vtable"} {
		if strings.Contains(c+h, forbidden) {
			t.Fatalf("generated artifacts contain forbidden runtime %q", forbidden)
		}
	}
}

func TestInterfaceDynRejectsMalformedWitnessMIR(t *testing.T) {
	for name, witness := range map[string]MIRInterfaceWitness{
		"missing identity": {Interface: "Drawable", ConcreteType: "Circle", NoAllocation: true},
		"allocation law":   {ID: "Drawable__Circle", Interface: "Drawable", ConcreteType: "Circle"},
		"duplicate entry":  {ID: "Drawable__Circle", Interface: "Drawable", ConcreteType: "Circle", Methods: []string{"Draw", "Draw"}, NoAllocation: true},
	} {
		t.Run(name, func(t *testing.T) {
			var diagnostic Diagnostic
			err := evt1ValidateMIR(MIR{Witnesses: []MIRInterfaceWitness{witness}})
			if !errors.As(err, &diagnostic) || diagnostic.Code != "INTERFACE_WITNESS_INVALID" {
				t.Fatalf("diagnostic = %v", err)
			}
		})
	}
}

func TestInterfaceDynRejectsMalformedDynMIR(t *testing.T) {
	for _, kind := range []string{"dyn_make", "dyn_call", "dyn_field_get", "dyn_field_set"} {
		t.Run(kind, func(t *testing.T) {
			mir := MIR{Functions: []MIRFunction{{Name: "Broken", Operations: []MIROperation{{ID: "Broken.01", Kind: kind}}}}}
			var diagnostic Diagnostic
			if err := evt1ValidateMIR(mir); !errors.As(err, &diagnostic) || diagnostic.Code != "DYN_MIR_INVALID" {
				t.Fatalf("diagnostic = %v", err)
			}
		})
	}
}

func TestInterfaceDynNativeC11(t *testing.T) {
	outputs := generateInterfaceDynFixture(t, "valid", "class_interface_dyn.concept")
	harness := "#include \"class_interface_dyn.generated.h\"\nint main(void) { return concept_class_interface_dyn_exercise() == 5 ? 0 : 1; }\n"
	runFoundationNativeHarness(t, outputs, "interface_dyn_harness.c", harness)
	cases := []struct {
		file    string
		harness string
	}{
		{"class_private_field_method.concept", "#include \"class_private_field_method.generated.h\"\nint main(void) { concept_device d = { .id = 1, .state = 7 }; return concept_class_private_field_method_read_private_through_public_method(&d) == 7 ? 0 : 1; }\n"},
		{"dyn_immovable_ref.concept", "#include \"dyn_immovable_ref.generated.h\"\nint main(void) { concept_device d = { .state = 7 }; concept_dyn_immovable_ref_reset_immovable(&d); return d.state; }\n"},
		{"dyn_scoped_provenance.concept", "#include \"dyn_scoped_provenance.generated.h\"\nint main(void) { concept_item i = { .value = 9 }; return concept_dyn_scoped_provenance_inspect_scoped(&i) == 9 ? 0 : 1; }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			fixture := generateInterfaceDynFixture(t, "valid", tc.file)
			runFoundationNativeHarness(t, fixture, "interface_dyn_"+tc.file+".c", tc.harness)
		})
	}
}
