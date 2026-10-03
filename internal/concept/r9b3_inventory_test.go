package concept

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestR9b3InventoryIdentityScope(t *testing.T) {
	body, err := os.ReadFile("../../language/evt1/machine-stack/valid/machine_multiple_nested_frames.concept")
	if err != nil {
		t.Fatal(err)
	}
	firstSource := "module Research.First;\n" + string(body)
	secondSource := "module Research.Second;\n" + string(body)
	plans := []ActivationStackLayout{}
	for _, source := range []string{firstSource, secondSource} {
		module, err := Parse("inventory.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		mir := buildMIR(module, env)
		plan, err := planActivationStack(mir.Automata[0], env)
		if err != nil {
			t.Fatal(err)
		}
		plans = append(plans, plan)
	}
	if plans[0].Identity != plans[1].Identity || len(plans[0].Machines) != 3 || len(plans[1].Machines) != 3 {
		t.Fatalf("existing identity coverage changed: first=%+v second=%+v", plans[0], plans[1])
	}
	path := "testdata/r9b3/Research/Closure.concept"
	probe, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(probe))
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	stringType, _ := evt1BuiltinType("string", Span{})
	args := []Value{
		{Kind: ValueString, Type: stringType, StringValue: plans[0].Identity},
		{Kind: ValueString, Type: stringType, StringValue: "Research.First"},
		{Kind: ValueString, Type: stringType, StringValue: plans[1].Identity},
		{Kind: ValueString, Type: stringType, StringValue: "Research.Second"},
	}
	var first []byte
	for run := 0; run < 100; run++ {
		verdict, err := evt1InvokePredicateOnMeasured(env, env, "InventoryIdentityPair", args, Span{}, nil)
		if err != nil || verdict.Outcome != FactDisproven || verdict.Refutation == nil || len(verdict.FactAuthority) != 0 {
			t.Fatalf("coverage probe: %+v %v", verdict, err)
		}
		encoded, err := json.Marshal(verdict)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first = encoded
		}
		if !bytes.Equal(first, encoded) {
			t.Fatalf("inventory refutation drift %d", run)
		}
	}
	t.Logf("actual activation topology identity=%s; two distinct module boundaries, 3 members each; 100 typed boundary-coverage refutations", plans[0].Identity)
}
