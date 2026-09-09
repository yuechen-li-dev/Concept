package concept

import (
	"fmt"
	"strings"
)

const evt1MachineStackCapacity = 8

func evt1ValidateMachineControlFlow(block *Block) error {
	var terminal Statement
	for _, statement := range block.Statements {
		if terminal != nil {
			previous, previousCompletion := terminal.(*MachineCompleteStmt)
			current, currentCompletion := statement.(*MachineCompleteStmt)
			if previousCompletion && currentCompletion {
				if previous.Operation == "pop" && current.Operation == "pop" {
					return evt1Diagnostic("MACHINE_STACK_UNDERFLOW", "a second pop would underflow the active frame", current.Span)
				}
				return evt1Diagnostic("MACHINE_DOUBLE_COMPLETION", "a machine Step path cannot complete twice", current.Span)
			}
			return evt1Diagnostic("MACHINE_FRAME_INVALID", "machine Step control is terminal; no statement may follow it", statement.statementSpan())
		}
		switch statement.(type) {
		case *TransitionStmt, *TransitionMatchStmt, *TransitionDecideStmt, *TransitionInferStmt, *YieldStmt, *PushMachineStmt, *MachineCompleteStmt:
			terminal = statement
		}
	}
	return nil
}

func collectMachineControlMIR(block *Block, machine MachineDecl, out *MIRState) {
	for _, statement := range block.Statements {
		switch stmt := statement.(type) {
		case *PushMachineStmt:
			out.MachineControl = append(out.MachineControl, MIRMachineControl{Kind: "push_machine", Machine: stmt.Machine, ResumeState: stmt.ResumeState, CleanupEdge: "TransientBeforeFramePush", SourceSpan: stmt.Span})
		case *MachineCompleteStmt:
			payload := "void"
			if stmt.Kind == "success" {
				payload = machine.ResultType.String()
			}
			if stmt.Kind == "failure" {
				payload = machine.ErrorType.String()
			}
			out.MachineControl = append(out.MachineControl, MIRMachineControl{Kind: "pop_machine", Outcome: strings.Title(stmt.Kind), PayloadType: payload, CleanupEdge: "TransientThenFrameBeforePop", SourceSpan: stmt.Span})
		case *IfStmt:
			collectMachineControlMIR(&stmt.Then, machine, out)
			if stmt.Else != nil {
				collectMachineControlMIR(stmt.Else, machine, out)
			}
		case *Block:
			collectMachineControlMIR(stmt, machine, out)
		}
	}
}

func evt1MachineFramesCName(machineName string) string {
	return evt1PayloadFieldName(machineName) + "_frames"
}

func evt1MachineOutcomeCName(automataName, machineName string) string {
	return evt1CName(evt1MachineOutcomeTypeName(automataName, machineName))
}

func evt1MachineLastOutcomeField(machineName string) string {
	return "last_" + evt1PayloadFieldName(machineName) + "_outcome"
}

func evt1MachineExecuteCName(automataName, machineName string) string {
	return evt1AutomataStepCName(automataName, machineName) + "_frame"
}

func evt1MachineInitFrameCName(automataName, machineName string) string {
	return evt1CName(automataName) + "_init_" + evt1PayloadFieldName(machineName) + "_frame"
}

func evt1MachineDropFrameCName(automataName, machineName string) string {
	return evt1CName(automataName) + "_drop_" + evt1PayloadFieldName(machineName) + "_frame"
}

// canonicalAutomataStackRuntimeSupport is the R5e canonical runtime. It is a
// statically-specialized, fixed-capacity frame stack: no heap, scheduler, or
// saved instruction pointer is present.
func (l *lowering) canonicalAutomataStackRuntimeSupport(info *evt1AutomataInfo) string {
	var b strings.Builder
	automata := info.Decl.Name
	stateType := evt1AutomataStateEnvironmentCName(automata)
	instanceType := evt1AutomataRuntimeInstanceCName(automata)

	b.WriteString(fmt.Sprintf("typedef struct %s {\n", stateType))
	if len(info.Decl.StateFields) == 0 {
		b.WriteString("  unsigned char unused;\n")
	}
	for _, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(field.Type), field.Name))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", stateType))

	for _, machine := range info.Decl.Machines {
		mt := evt1AutomataMachineStorageCName(automata, machine.Name)
		b.WriteString("enum {\n")
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("  %s = %d,\n", evt1AutomataStateConstName(automata, machine.Name, state.Name), info.StateOrdinal[machine.Name][state.Name]))
		}
		b.WriteString("};\n")
		b.WriteString(fmt.Sprintf("typedef struct %s {\n  uint8_t current_state;\n", mt))
		for _, field := range machine.Fields {
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(field.Type), field.Name))
			if evt1TypeHasDrop(l.env, field.Type) {
				b.WriteString(fmt.Sprintf("  bool %s_moved;\n", field.Name))
			}
		}
		b.WriteString(fmt.Sprintf("} %s;\n\n", mt))
		ot := evt1MachineOutcomeCName(automata, machine.Name)
		b.WriteString(fmt.Sprintf("typedef struct %s {\n  uint8_t tag; /* 0 unavailable, 1 neutral, 2 success, 3 failure */\n", ot))
		if machine.ResultType.Name != "void" {
			b.WriteString(fmt.Sprintf("  %s success;\n", evt1CType(machine.ResultType)))
		}
		if machine.ErrorType.Name != "void" {
			b.WriteString(fmt.Sprintf("  %s failure;\n", evt1CType(machine.ErrorType)))
		}
		b.WriteString(fmt.Sprintf("} %s;\n\n", ot))
	}

	b.WriteString(fmt.Sprintf("typedef struct %s {\n  %s shared;\n  uint8_t depth;\n  uint8_t machine_tags[%d];\n  bool completed;\n", instanceType, stateType, evt1MachineStackCapacity))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  %s %s[%d];\n", evt1AutomataMachineStorageCName(automata, machine.Name), evt1MachineFramesCName(machine.Name), evt1MachineStackCapacity))
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1MachineOutcomeCName(automata, machine.Name), evt1MachineLastOutcomeField(machine.Name)))
	}
	b.WriteString(fmt.Sprintf("} %s;\n\n", instanceType))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("static %s %s_result_%s(%s* instance);\n", evt1MachineOutcomeCName(automata, machine.Name), evt1CName(automata), evt1PayloadFieldName(machine.Name), instanceType))
	}
	b.WriteString("\n")

	for _, machine := range info.Decl.Machines {
		frames := evt1MachineFramesCName(machine.Name)
		b.WriteString(fmt.Sprintf("static void %s(%s* instance, uint8_t slot) {\n", evt1MachineInitFrameCName(automata, machine.Name), instanceType))
		b.WriteString(fmt.Sprintf("  instance->%s[slot].current_state=%s; /* frame initialization */\n", frames, evt1AutomataStateConstName(automata, machine.Name, machine.States[0].Name)))
		initLower := newEVT1FunctionLowerer(l, FunctionDecl{Name: "machine_frame_init", ReturnType: Type{Name: "void", Kind: TypeBuiltin}}, "", true)
		for _, field := range machine.Fields {
			value := fmt.Sprintf("(%s){0}", evt1CType(field.Type))
			if field.Initializer != nil {
				_, value, _ = initLower.lowerExpr(field.Initializer, 1)
			}
			b.WriteString(fmt.Sprintf("  instance->%s[slot].%s = %s;\n", frames, field.Name, value))
			if evt1TypeHasDrop(l.env, field.Type) {
				b.WriteString(fmt.Sprintf("  instance->%s[slot].%s_moved = false;\n", frames, field.Name))
			}
		}
		b.WriteString("}\n\n")
		b.WriteString(fmt.Sprintf("static void %s(%s* instance, uint8_t slot) {\n", evt1MachineDropFrameCName(automata, machine.Name), instanceType))
		dropLower := newEVT1FunctionLowerer(l, FunctionDecl{Name: "machine_frame_drop", ReturnType: Type{Name: "void", Kind: TypeBuiltin}}, "", true)
		for fi := len(machine.Fields) - 1; fi >= 0; fi-- {
			field := machine.Fields[fi]
			value := fmt.Sprintf("instance->%s[slot].%s", frames, field.Name)
			if evt1TypeHasDrop(l.env, field.Type) {
				b.WriteString(fmt.Sprintf("  if (!instance->%s[slot].%s_moved) {\n", frames, field.Name))
				b.WriteString(dropLower.lowerDropValue(field.Type, value, 2))
				b.WriteString("  }\n")
			}
		}
		b.WriteString("}\n\n")
	}

	for _, machine := range info.Decl.Machines {
		frames := evt1MachineFramesCName(machine.Name)
		b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n  uint8_t frame_index = (uint8_t)(instance->depth - 1u);\n  switch (instance->%s[frame_index].current_state) {\n", evt1MachineExecuteCName(automata, machine.Name), instanceType, frames))
		for _, state := range machine.States {
			b.WriteString(fmt.Sprintf("    case %s:\n      {\n", evt1AutomataStateConstName(automata, machine.Name, state.Name)))
			lower := newEVT1FunctionLowerer(l, FunctionDecl{Name: evt1MachineExecuteCName(automata, machine.Name), ReturnType: Type{Name: "void", Kind: TypeBuiltin}, Body: state.Body}, "", true)
			lower.automataStepName, lower.machineStepName = automata, machine.Name
			lower.automataInfo, lower.machineDecl = info, &machine
			lower.frameIndexName = "frame_index"
			lower.scope[0]["state"] = evt1Binding{cName: "instance->shared", t: Type{Name: automata + "#state", Kind: TypeStruct}}
			for _, field := range info.Decl.StateFields {
				lower.scope[0][field.Name] = evt1Binding{cName: "instance->shared." + field.Name, t: field.Type}
			}
			frameBase := "instance->" + frames + "[frame_index]"
			lower.scope[0]["machine"] = evt1Binding{cName: frameBase, t: Type{Name: automata + "#" + machine.Name + "#machine", Kind: TypeStruct}}
			for _, field := range machine.Fields {
				lower.scope[0][field.Name] = evt1Binding{cName: frameBase + "." + field.Name, t: field.Type}
			}
			b.WriteString(lower.lowerBlock(*state.Body, 4))
			b.WriteString("        break;\n      }\n")
		}
		b.WriteString(fmt.Sprintf("    default: concept_abort_invalid_automata_state(\"%s\", %d, instance->%s[frame_index].current_state);\n  }\n}\n\n", automata, info.MachineOrdinal[machine.Name], frames))
	}

	topStep := evt1CName(automata) + "_step_top"
	b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n", topStep, instanceType))
	b.WriteString(fmt.Sprintf("  if (instance->completed || instance->depth == 0u) return;\n  switch (instance->machine_tags[instance->depth - 1u]) {\n"))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("    case %d: %s(instance); return;\n", info.MachineOrdinal[machine.Name], evt1MachineExecuteCName(automata, machine.Name)))
	}
	b.WriteString(fmt.Sprintf("    default: concept_abort_automata_stack(\"%s\", \"invalid machine frame\");\n  }\n}\n\n", automata))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("static void %s(%s* instance) { if (instance->depth == 1u) instance->machine_tags[0] = %d; %s(instance); }\n", evt1AutomataStepCName(automata, machine.Name), instanceType, info.MachineOrdinal[machine.Name], topStep))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("static void %s(%s* instance", evt1AutomataRuntimeInitName(automata), instanceType))
	for i, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf(", %s state_%d", evt1CType(field.Type), i))
	}
	b.WriteString(") {\n")
	for i, field := range info.Decl.StateFields {
		b.WriteString(fmt.Sprintf("  instance->shared.%s = state_%d;\n", field.Name, i))
	}
	b.WriteString("  instance->depth = 1u;\n  instance->completed = false;\n")
	b.WriteString(fmt.Sprintf("  instance->machine_tags[0] = %d;\n", info.MachineOrdinal[info.RootMachine]))
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  %s(instance, 0u);\n", evt1MachineInitFrameCName(automata, machine.Name)))
	}
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("  instance->%s.tag = 0u;\n", evt1MachineLastOutcomeField(machine.Name)))
	}
	b.WriteString("}\n\n")

	for _, machine := range info.Decl.Machines {
		ot := evt1MachineOutcomeCName(automata, machine.Name)
		name := evt1CName(automata) + "_result_" + evt1PayloadFieldName(machine.Name)
		field := evt1MachineLastOutcomeField(machine.Name)
		b.WriteString(fmt.Sprintf("static %s %s(%s* instance) {\n  if (instance->%s.tag == 0u) concept_abort_automata_stack(\"%s\", \"machine result cannot be read before completion\");\n  return instance->%s;\n}\n\n", ot, name, instanceType, field, automata, field))
	}

	b.WriteString(fmt.Sprintf("static void %s(%s* instance) {\n", evt1AutomataDropCName(automata), instanceType))
	b.WriteString("  while (instance->depth > 0u) {\n    uint8_t slot = (uint8_t)(instance->depth - 1u);\n    switch (instance->machine_tags[slot]) {\n")
	for _, machine := range info.Decl.Machines {
		b.WriteString(fmt.Sprintf("      case %d: %s(instance, slot); break;\n", info.MachineOrdinal[machine.Name], evt1MachineDropFrameCName(automata, machine.Name)))
	}
	b.WriteString("      default: break;\n    }\n    instance->depth = slot;\n  }\n")
	dropLower := newEVT1FunctionLowerer(l, FunctionDecl{Name: "automata_drop", ReturnType: Type{Name: "void", Kind: TypeBuiltin}}, "", true)
	for _, machine := range info.Decl.Machines {
		outcome := "instance->" + evt1MachineLastOutcomeField(machine.Name)
		if evt1TypeHasDrop(l.env, machine.ResultType) {
			b.WriteString(fmt.Sprintf("  if (%s.tag == 2u) {\n", outcome))
			b.WriteString(dropLower.lowerDropValue(machine.ResultType, outcome+".success", 2))
			b.WriteString("  }\n")
		}
		if evt1TypeHasDrop(l.env, machine.ErrorType) {
			b.WriteString(fmt.Sprintf("  if (%s.tag == 3u) {\n", outcome))
			b.WriteString(dropLower.lowerDropValue(machine.ErrorType, outcome+".failure", 2))
			b.WriteString("  }\n")
		}
	}
	for i := len(info.Decl.StateFields) - 1; i >= 0; i-- {
		field := info.Decl.StateFields[i]
		b.WriteString(dropLower.lowerDropValue(field.Type, "instance->shared."+field.Name, 1))
	}
	b.WriteString("}\n\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerMachinePush(stmt PushMachineStmt, indent int) string {
	if f.automataInfo == nil || f.frameIndexName == "" {
		return ind(indent) + "/* invalid machine push */\n"
	}
	var b strings.Builder
	b.WriteString(f.lowerAllScopeDrops(indent))
	b.WriteString(ind(indent) + fmt.Sprintf("if (instance->depth >= %d) concept_abort_automata_stack(\"%s\", \"automata machine stack capacity exceeded\");\n", evt1MachineStackCapacity, f.automataStepName))
	b.WriteString(ind(indent) + fmt.Sprintf("instance->%s[%s].current_state = %s;\n", evt1MachineFramesCName(f.machineStepName), f.frameIndexName, evt1AutomataStateConstName(f.automataStepName, f.machineStepName, stmt.ResumeState)))
	b.WriteString(ind(indent) + fmt.Sprintf("instance->machine_tags[instance->depth] = %d;\n", f.automataInfo.MachineOrdinal[stmt.Machine]))
	b.WriteString(ind(indent) + fmt.Sprintf("instance->%s.tag = 0u;\n", evt1MachineLastOutcomeField(stmt.Machine)))
	b.WriteString(ind(indent) + fmt.Sprintf("%s(instance, instance->depth);\n", evt1MachineInitFrameCName(f.automataStepName, stmt.Machine)))
	b.WriteString(ind(indent) + "instance->depth = (uint8_t)(instance->depth + 1u);\n")
	b.WriteString(ind(indent) + "return; /* CONCEPT_STEP_CHILD_PUSHED */\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerMachineCompletion(stmt MachineCompleteStmt, indent int) string {
	if f.automataInfo == nil || f.machineDecl == nil || f.frameIndexName == "" {
		return ind(indent) + "/* invalid machine completion */\n"
	}
	var b strings.Builder
	valueName := ""
	if stmt.Value != nil {
		prelude, value, valueType := f.lowerExpr(stmt.Value, indent)
		b.WriteString(prelude)
		valueName = f.nextTemp("machine_outcome")
		b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(valueType), valueName, value))
		if moved, ok := stmt.Value.(*MoveExpr); ok {
			if name, ok := moved.Value.(*NameExpr); ok {
				for _, field := range f.machineDecl.Fields {
					if field.Name == name.Name && evt1TypeHasDrop(f.l.env, field.Type) {
						b.WriteString(ind(indent) + fmt.Sprintf("instance->%s[%s].%s_moved = true;\n", evt1MachineFramesCName(f.machineStepName), f.frameIndexName, field.Name))
					}
				}
			}
		}
	}
	b.WriteString(f.lowerAllScopeDrops(indent))
	field := "instance->" + evt1MachineLastOutcomeField(f.machineStepName)
	tag := map[string]int{"neutral": 1, "success": 2, "failure": 3}[stmt.Kind]
	b.WriteString(ind(indent) + fmt.Sprintf("%s.tag = %d;\n", field, tag))
	if stmt.Kind == "success" {
		b.WriteString(ind(indent) + fmt.Sprintf("%s.success = %s;\n", field, valueName))
	}
	if stmt.Kind == "failure" {
		b.WriteString(ind(indent) + fmt.Sprintf("%s.failure = %s;\n", field, valueName))
	}
	b.WriteString(ind(indent) + fmt.Sprintf("%s(instance, %s);\n", evt1MachineDropFrameCName(f.automataStepName, f.machineStepName), f.frameIndexName))
	b.WriteString(ind(indent) + "instance->depth = (uint8_t)(instance->depth - 1u);\n")
	b.WriteString(ind(indent) + "if (instance->depth == 0u) instance->completed = true;\n")
	b.WriteString(ind(indent) + "return; /* CONCEPT_STEP_CHILD_OR_ROOT_COMPLETED */\n")
	return b.String()
}
