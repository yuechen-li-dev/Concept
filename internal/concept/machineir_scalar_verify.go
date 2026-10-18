package concept

import (
	"fmt"
	"strings"
)

// Reuse the existing checked virtual CFG/type verifier after validating the
// version-specific machine metadata. This is input verification, not lowering.
func verifyScalarMachine(m ScalarMachineModule) error {
	if len(m.Functions) == 0 || len(m.Functions) > 128 {
		return fmt.Errorf("MIR_CMIRAMD4_FUNCTION_COUNT")
	}
	lir := LIRModule{}
	for _, f := range m.Functions {
		bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD4_%s %s", s, f.Name) }
		if f.Target != "amd64-windows" || f.ABI != "win64" || len(f.VRegs) > 128 || len(f.Slots) > 64 || len(f.Blocks) > 128 {
			return bad("FUNCTION")
		}
		if !strings.HasPrefix(f.Identity, f.Name+"|"+f.Name+"(") || !strings.HasSuffix(f.Identity, ")") {
			return bad("IDENTITY")
		}
		definitions := make([]int, len(f.VRegs))
		if _, e := machineBridgeTag(scalarBridgeTagsScalarRepresentation, f.Result); e != nil || f.Result == "address" {
			return bad("RESULT_CLASS")
		}
		lf := LIRFunction{Identity: f.Identity, Name: f.Name, Result: LIRType(f.Result), Source: Span{Line: f.Source.Line, Column: f.Source.Column}}
		for i, v := range f.VRegs {
			if _, e := machineBridgeTag(scalarBridgeTagsScalarRepresentation, v.Representation); e != nil || v.Representation == "void" || v.ID != i || v.Width != scalarWidth(v.Representation) || v.Class != scalarClass(v.Representation) || v.Address != (v.Representation == "address") {
				return bad("VREG_CLASS")
			}
		}
		for i, p := range f.Params {
			if p.Index != i || p.Value < 0 || p.Value >= len(f.VRegs) || f.VRegs[p.Value].Representation != scalarRepresentation(LIRType(p.Type)) {
				return bad("PARAMETER")
			}
			definitions[p.Value]++
			if f.VRegs[p.Value].Address {
				matches := 0
				for _, s := range f.Slots {
					if s.ParamValue == p.Value && s.Extent > 0 && p.Type == fmt.Sprintf("[%d]%s", s.Extent, s.Representation) {
						matches++
					}
				}
				if matches != 1 {
					return bad("ARRAY_PARAMETER")
				}
			}
			lf.Params = append(lf.Params, LIRValue{ID: p.Value, Type: LIRType(p.Type)})
		}
		for i, s := range f.Slots {
			width := scalarWidth(s.Representation)
			if width == 0 || s.Representation == "address" || s.ID != i || s.Extent < 0 || s.Extent > 1048576 || s.Align != width || s.ParamValue < -1 {
				return bad("SLOT")
			}
			count := s.Extent
			if count == 0 {
				count = 1
			}
			if s.Size != width*count {
				return bad("SLOT_GEOMETRY")
			}
			typ := LIRType(s.Representation)
			var elem LIRType
			if s.Extent > 0 {
				elem = typ
				typ = LIRType(fmt.Sprintf("[%d]%s", s.Extent, s.Representation))
			} else if s.ParamValue != -1 {
				return bad("SLOT_PARAMETER")
			}
			lf.Slots = append(lf.Slots, LIRSlot{ID: s.ID, Name: s.Name, Type: typ, Element: elem, Extent: s.Extent, ParamValue: s.ParamValue})
		}
		instructionCount := 0
		for bi, b := range f.Blocks {
			lb := LIRBlock{ID: b.ID, Term: LIRTerminator{Op: b.Term.Op, Value: b.Term.Value, True: b.Term.True, False: b.Term.False}}
			if _, e := machineBridgeTag(scalarBridgeTagsScalarTerminator, b.Term.Op); e != nil {
				return bad("TERMINATOR")
			}
			for ii, in := range b.Instructions {
				instructionCount++
				if instructionCount > 512 {
					return bad("INSTRUCTION_COUNT")
				}
				if _, e := machineBridgeTag(scalarBridgeTagsScalarOpcode, in.Op); e != nil {
					return bad("OPCODE")
				}
				if _, e := machineBridgeTag(scalarBridgeTagsScalarRepresentation, in.Representation); e != nil {
					return bad("OPERATION_REPRESENTATION")
				}
				if in.LIRBlock != bi || in.LIRInstruction != ii || in.Source.Line <= 0 || in.Source.Column <= 0 {
					return bad("PROVENANCE")
				}
				if in.Result >= 0 && (in.Result >= len(f.VRegs) || f.VRegs[in.Result].Representation != in.Representation) {
					return bad("RESULT_REPRESENTATION")
				}
				if in.Result >= 0 {
					definitions[in.Result]++
				}
				if in.Op == "load_slot" || in.Op == "store_slot" {
					if in.Slot < 0 || in.Slot >= len(f.Slots) || f.Slots[in.Slot].Representation != in.Representation {
						return bad("SLOT_REPRESENTATION")
					}
				}
				li := LIRInstruction{Op: in.Op, Result: in.Result, Type: LIRType(in.Representation), Slot: in.Slot, Literal: in.Literal, Extent: in.Extent, Stride: in.Stride}
				for _, a := range in.Args {
					if a.Value < 0 || a.Value >= len(f.VRegs) {
						return bad("OPERAND")
					}
					li.Args = append(li.Args, a.Value)
				}
				if in.Op != "call" && len(in.Calls) != 0 {
					return bad("UNEXPECTED_CALL")
				}
				if in.Op != "const" && in.Op != "fconst" && in.Literal != "" {
					return bad("UNEXPECTED_LITERAL")
				}
				floatCompare := in.Op == "feq" || in.Op == "fne" || in.Op == "flt" || in.Op == "fle" || in.Op == "fgt" || in.Op == "fge"
				intCompare := in.Op == "eq" || in.Op == "ne" || in.Op == "lt" || in.Op == "le" || in.Op == "gt" || in.Op == "ge"
				if floatCompare || intCompare {
					if len(in.Args) != 2 || lirFloating(LIRType(f.VRegs[in.Args[0].Value].Representation)) != floatCompare {
						return bad("COMPARE_CLASS")
					}
					if floatCompare {
						li.Op = strings.TrimPrefix(in.Op, "f")
					}
				}
				if in.Op == "fconst" {
					if !lirFloating(li.Type) || !lirFloatLiteralValid(li.Type, in.Literal) {
						return bad("FLOAT_CONSTANT")
					}
					li.Op = "const"
				}
				if in.Op == "const" {
					if !lirInteger(li.Type) && li.Type != "bool" {
						return bad("INTEGER_CONSTANT")
					}
					if li.Type == "bool" && in.Literal != "true" && in.Literal != "false" {
						return bad("BOOL_CONSTANT")
					}
					if _, e := machineImmediate(in.Literal, scalarWidth(in.Representation), machineSigned(li.Type)); e != nil {
						return bad("INTEGER_CONSTANT")
					}
				}
				if in.Op == "index_address" {
					if in.Slot < 0 || in.Slot >= len(f.Slots) || in.Representation != "address" {
						return bad("INDEX_ADDRESS")
					}
					li.Type = LIRType("ptr<" + f.Slots[in.Slot].Representation + ">")
				}
				if in.Op == "call" {
					if len(in.Calls) != 1 {
						return bad("CALL_SHAPE")
					}
					c := in.Calls[0]
					if c.Convention != "win64" || c.Result != in.Representation || len(c.Arguments) != len(in.Args) {
						return bad("CALL_CLASS")
					}
					li.Callee = ""
					for _, target := range m.Functions {
						if target.Identity == c.Target {
							li.Callee = target.Name
							break
						}
					}
					li.CallTarget = c.Target
					li.CallABI = c.Convention
					for i, a := range c.Arguments {
						if a.Value != in.Args[i].Value || a.Type != f.VRegs[a.Value].Representation || !lirCallScalar(LIRType(a.Type)) {
							return bad("CALL_ARGUMENT")
						}
						li.ArgTypes = append(li.ArgTypes, LIRType(a.Type))
					}
				}
				lb.Instructions = append(lb.Instructions, li)
			}
			lf.Blocks = append(lf.Blocks, lb)
		}
		lir.Functions = append(lir.Functions, lf)
		for _, count := range definitions {
			if count != 1 {
				return bad("VREG_DEFINITION")
			}
		}
	}
	// Proof IDs remain opaque transported provenance, as in CMIRAMD3. They are
	// not converted into semantic authority by this structural checker.
	return VerifyLIR(lir)
}
