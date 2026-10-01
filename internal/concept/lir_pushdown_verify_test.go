package concept

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEVT2x6VerifierRejectsMalformedTransfers(t *testing.T) {
	module, _ := pushdownFixture(t)
	base, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(*LIRFunction)
	}{
		{"missing-live-depth-guard", func(f *LIRFunction) {
			f.Blocks[f.Blocks[0].Term.False].Instructions[len(f.Blocks[f.Blocks[0].Term.False].Instructions)-1].Op = "ge"
		}},
		{"depth-before-init", func(f *LIRFunction) {
			b := pushBlock(f)
			v := activationView(*b)
			for i, in := range b.Instructions {
				if in.Op == "store" && v.headerAddress(in.Args[0], 0) {
					b.Instructions = append([]LIRInstruction{in}, append(b.Instructions[:i], b.Instructions[i+1:]...)...)
					return
				}
			}
		}},
		{"wrong-child-tag", func(f *LIRFunction) { mutatePushLiteral(f, 0, "99") }},
		{"missing-persistent-initializer", func(f *LIRFunction) {
			b := pushBlock(f)
			v := activationView(*b)
			l := f.Activation.Layout
			for i, in := range b.Instructions {
				if in.Op == "store" {
					a := v.defs[in.Args[0]]
					if a.Op == "index_address" && v.depth(a.Args[1]) && a.FrameOffset == l.SlotsOffset+l.SlotDataOffset+4 {
						b.Instructions = append(b.Instructions[:i], b.Instructions[i+1:]...)
						return
					}
				}
			}
		}},
		{"wrong-initial-state", func(f *LIRFunction) { mutatePushLiteral(f, 4, "99") }},
		{"child-write-outside-slot", func(f *LIRFunction) {
			b := pushBlock(f)
			v := activationView(*b)
			for i, in := range b.Instructions {
				if in.Op == "index_address" && v.depth(in.Args[1]) {
					b.Instructions[i].FrameOffset += f.Activation.Layout.SlotSize
					return
				}
			}
		}},
		{"missing-continuation", func(f *LIRFunction) {
			b := pushBlock(f)
			v := activationView(*b)
			for i, in := range b.Instructions {
				if in.Op == "store" {
					a := v.defs[in.Args[0]]
					if a.Op == "index_address" && v.top(a.Args[1]) {
						b.Instructions = append(b.Instructions[:i], b.Instructions[i+1:]...)
						return
					}
				}
			}
		}},
		{"missing-capacity-guard", func(f *LIRFunction) {
			b := pushBlock(f)
			v := activationView(*b)
			for i, in := range b.Instructions {
				if in.Op == "check_index" && v.depth(in.Args[0]) {
					b.Instructions = append(b.Instructions[:i], b.Instructions[i+1:]...)
					return
				}
			}
		}},
		{"depth-before-destroy", func(f *LIRFunction) {
			p := popTransfer(f)
			f.Blocks[p.DestroyBlocks[0]].Instructions = append([]LIRInstruction(nil), f.Blocks[p.PublishBlocks[0]].Instructions...)
		}},
		{"destroy-wrong-type", func(f *LIRFunction) {
			p := popTransfer(f)
			f.Blocks[p.DestroyBlocks[0]].Label = "destroy." + f.Activation.Layout.Machines[1].Identity
		}},
		{"missing-destroy-tag-arm", func(f *LIRFunction) {
			p := popTransfer(f)
			arm := f.Blocks[p.Block].Term.True
			f.Blocks[arm].Term.False = f.Blocks[f.Blocks[arm].Term.False].Term.False
		}},
		{"pop-without-depth-guard", func(f *LIRFunction) {
			p := popTransfer(f)
			arm := f.Blocks[p.Block].Term.True
			b := &f.Blocks[arm]
			for i, in := range b.Instructions {
				if in.Op == "check_index" {
					b.Instructions = append(b.Instructions[:i], b.Instructions[i+1:]...)
					return
				}
			}
		}},
		{"popped-frame-use-after-destroy", func(f *LIRFunction) {
			p := popTransfer(f)
			b := pushBlock(f)
			for _, in := range b.Instructions {
				if in.Op == "index_address" {
					q := p.PublishBlocks[0]
					f.Blocks[q].Instructions = append(f.Blocks[q].Instructions, in)
					return
				}
			}
		}},
		{"parent-access-with-child-layout", func(f *LIRFunction) {
			for bi, b := range f.Blocks {
				if b.Label == "state.Parent.Done" {
					for ii, in := range b.Instructions {
						if in.Op == "index_address" && in.Type == "ptr<i32>" {
							f.Blocks[bi].Instructions[ii].FrameOffset = f.Activation.Layout.SlotsOffset + f.Activation.Layout.SlotDataOffset + 8
							return
						}
					}
				}
			}
		}},
		{"missing-state-dispatch", func(f *LIRFunction) {
			for i, b := range f.Blocks {
				if b.Label == "dispatch.Child.Start" {
					f.Blocks[i].Term.True = f.Blocks[i].Term.False
					return
				}
			}
		}},
		{"child-frame-tag-incoherent", func(f *LIRFunction) {
			for i, t := range f.Activation.Transfers {
				if t.Kind == "push" {
					f.Activation.Transfers[i].ChildTag = 2
					return
				}
			}
		}},
		{"removed-push-lifetime-region", func(f *LIRFunction) {
			for i, t := range f.Activation.Transfers {
				if t.Kind == "push" {
					f.Activation.Transfers = append(f.Activation.Transfers[:i], f.Activation.Transfers[i+1:]...)
					return
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := json.Marshal(base)
			if err != nil {
				t.Fatal(err)
			}
			var malformed LIRModule
			if err := json.Unmarshal(wire, &malformed); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&malformed.Functions[1])
			if err := VerifyLIR(malformed); err == nil {
				t.Fatal("malformed transfer accepted")
			} else {
				t.Log(err)
			}
		})
	}
}

func pushBlock(f *LIRFunction) *LIRBlock {
	for _, t := range f.Activation.Transfers {
		if t.Kind == "push" {
			return &f.Blocks[t.Block]
		}
	}
	panic("fixture has no push")
}
func popTransfer(f *LIRFunction) LIRActivationTransfer {
	for _, t := range f.Activation.Transfers {
		if t.Kind != "push" {
			return t
		}
	}
	panic("fixture has no completion")
}
func mutatePushLiteral(f *LIRFunction, offset int, literal string) {
	b := pushBlock(f)
	v := activationView(*b)
	for _, in := range b.Instructions {
		if in.Op == "store" {
			a := v.defs[in.Args[0]]
			if a.Op == "index_address" && v.depth(a.Args[1]) && a.FrameOffset == f.Activation.Layout.SlotsOffset+offset {
				b.Instructions[v.positions[in.Args[1]]].Literal = literal
				return
			}
		}
	}
	panic("fixture store missing")
}

func TestEVT2x6InitializerScopeAndReconciledBoundaries(t *testing.T) {
	_, source := pushdownFixture(t)
	text := strings.ReplaceAll(string(source), "state Start { push Child goto Again; }", "state Start { int value = 99; push Child goto Again; }")
	text = strings.ReplaceAll(text, "int count = 4;", "int count = value + 4;")
	module, err := Parse("scope.concept", text)
	if err != nil {
		t.Fatal(err)
	}
	lir, err := GenerateLIR(module)
	if err != nil {
		t.Fatal(err)
	}
	b := pushBlock(&lir.Functions[1])
	seenShared := false
	v := activationView(*b)
	for _, in := range b.Instructions {
		if in.Op == "load_slot" {
			t.Fatal("child initializer read a parent local")
		}
		if in.Op == "load" && v.headerAddress(in.Args[0], 2) {
			seenShared = true
		}
	}
	if !seenShared {
		t.Fatal("child initializer did not read the shared environment")
	}
	terminal := strings.ReplaceAll(string(source), "state Done { state.value = state.value + machine.count; pop; }", "terminal state Done { }")
	module, err = Parse("terminal_pushdown.concept", terminal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateLIR(module); err == nil || !strings.Contains(err.Error(), "EVT2_UNSUPPORTED_PUSHDOWN_TERMINAL") {
		t.Fatalf("terminal settling silently changed: %v", err)
	}
	rootPop := `profile Core; automata Worker { machine Parent { state Start { pop; } } }`
	module, err = Parse("root_pop.concept", rootPop)
	if err != nil {
		t.Fatal(err)
	}
	if lir, err := GenerateLIR(module); err != nil || len(lir.Functions) != 2 || lir.Functions[1].Activation == nil {
		t.Fatalf("standalone root pop: %v", err)
	}
}
