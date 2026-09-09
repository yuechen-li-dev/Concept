package concept

import (
	"fmt"
	"sort"
	"strings"
)

const (
	evt1AsyncFrameBytes  = 1024
	evt1AsyncResultBytes = 512
)

func evt1ModuleHasAsync(module Module) bool {
	for _, fn := range module.Functions {
		if fn.Async {
			return true
		}
	}
	for _, template := range module.Templates {
		if template.Async {
			return true
		}
	}
	return false
}

func evt1AsyncRuntimeDeclarations() string {
	return fmt.Sprintf(`#include <stddef.h>
#define CONCEPT_ASYNC_STACK_CAPACITY %d
#define CONCEPT_ASYNC_FRAME_BYTES %d
#define CONCEPT_ASYNC_RESULT_BYTES %d
typedef union { max_align_t align; unsigned char bytes[CONCEPT_ASYNC_FRAME_BYTES]; } concept_async_frame_storage;
typedef union { max_align_t align; unsigned char bytes[CONCEPT_ASYNC_RESULT_BYTES]; } concept_async_result_storage;
struct concept_async_operation;
typedef void (*concept_async_step_fn)(struct concept_async_operation*, void*);
typedef struct { concept_async_step_fn step; concept_async_frame_storage storage; } concept_async_machine_frame;
typedef struct concept_async_operation {
  concept_async_machine_frame frames[CONCEPT_ASYNC_STACK_CAPACITY];
  unsigned int depth;
  bool completed;
  concept_async_result_storage result;
  size_t result_size;
  concept_async_result_storage child_outcome;
  size_t child_outcome_size;
} concept_async_operation;
void concept_async_step(concept_async_operation* operation);
bool concept_async_complete(const concept_async_operation* operation);
void concept_async_result(const concept_async_operation* operation, void* result, size_t size);

`, evt1MachineStackCapacity, evt1AsyncFrameBytes, evt1AsyncResultBytes)
}

func evt1AsyncRuntimeDefinitions() string {
	return `static void concept_async_abort(const char* reason) {
  fprintf(stderr, "invalid async operation: %s\n", reason);
  abort();
}

static void concept_async_push(concept_async_operation* operation, concept_async_step_fn step) {
  if (operation->depth >= CONCEPT_ASYNC_STACK_CAPACITY) concept_async_abort("machine stack capacity exceeded");
  concept_async_machine_frame* frame = &operation->frames[operation->depth++];
  memset(frame, 0, sizeof(*frame));
  frame->step = step;
}

static void concept_async_adopt(concept_async_operation* parent, concept_async_operation* child) {
  if (child->completed || child->depth == 0u) concept_async_abort("cannot push completed async operation");
  if (parent->depth + child->depth > CONCEPT_ASYNC_STACK_CAPACITY) concept_async_abort("machine stack capacity exceeded");
  memcpy(&parent->frames[parent->depth], child->frames, child->depth * sizeof(child->frames[0]));
  parent->depth += child->depth;
  child->depth = 0u;
}

static void concept_async_finish(concept_async_operation* operation, const void* value, size_t size) {
  if (operation->depth == 0u || size > CONCEPT_ASYNC_RESULT_BYTES) concept_async_abort("invalid completion");
  if (value != NULL && size != 0u) memcpy(operation->child_outcome.bytes, value, size);
  operation->child_outcome_size = size;
  operation->depth--;
  if (operation->depth == 0u) {
    if (value != NULL && size != 0u) memcpy(operation->result.bytes, value, size);
    operation->result_size = size;
    operation->completed = true;
  }
}

void concept_async_step(concept_async_operation* operation) {
  if (operation == NULL || operation->completed) return;
  if (operation->depth == 0u) concept_async_abort("empty active stack");
  concept_async_machine_frame* frame = &operation->frames[operation->depth - 1u];
  frame->step(operation, frame->storage.bytes);
}

bool concept_async_complete(const concept_async_operation* operation) {
  return operation != NULL && operation->completed;
}

void concept_async_result(const concept_async_operation* operation, void* result, size_t size) {
  if (operation == NULL || !operation->completed) concept_async_abort("result requested before completion");
  if (size != operation->result_size) concept_async_abort("result type or size mismatch");
  if (size != 0u) memcpy(result, operation->result.bytes, size);
}

`
}

func evt1AsyncStepName(base, fn string) string { return evt1FunctionSymbol(base, fn) + "__async_step" }
func evt1AsyncInitName(base, fn string) string { return evt1FunctionSymbol(base, fn) + "__async_init" }
func evt1AsyncFrameName(base, fn string) string {
	return evt1FunctionSymbol(base, fn) + "__async_frame"
}

func (l *lowering) asyncForwardDeclarations() string {
	var b strings.Builder
	for _, fn := range l.module.Functions {
		if !fn.Async {
			continue
		}
		b.WriteString(fmt.Sprintf("static void %s(concept_async_operation*, void*);\n", evt1AsyncStepName(l.outputBase, fn.Name)))
		b.WriteString(fmt.Sprintf("static void %s(void*", evt1AsyncInitName(l.outputBase, fn.Name)))
		for _, p := range fn.Params {
			b.WriteString(fmt.Sprintf(", %s %s", evt1CType(p.Type), p.Name))
		}
		b.WriteString(");\n")
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func (l *lowering) asyncFunctionSymbols(fn FunctionDecl) evt1FunctionSymbols {
	name := evt1FunctionSymbolForDecl(l.outputBase, l.env, fn)
	var proto strings.Builder
	proto.WriteString("concept_async_operation " + name + "(")
	for i, p := range fn.Params {
		if i > 0 {
			proto.WriteString(", ")
		}
		proto.WriteString(evt1CType(p.Type) + " " + p.Name)
	}
	proto.WriteString(");\n")
	if fn.Body == nil {
		return evt1FunctionSymbols{Prototype: proto.String()}
	}
	return evt1FunctionSymbols{Prototype: proto.String(), Body: l.lowerAsyncFunction(fn, name)}
}

func (l *lowering) lowerAsyncFunction(fn FunctionDecl, symbol string) string {
	a := evt1AnalyzeAsync(fn)
	cfg := evt1NormalizeAsyncCFG(fn, l.env)
	persistent := evt1AsyncPersistentMap(fn, a, cfg)
	frameName, stepName, initName := evt1AsyncFrameName(l.outputBase, fn.Name), evt1AsyncStepName(l.outputBase, fn.Name), evt1AsyncInitName(l.outputBase, fn.Name)
	names := evt1SortedPersistentNames(persistent)
	var b strings.Builder
	b.WriteString("typedef struct " + frameName + " {\n  unsigned int state;\n")
	for _, name := range names {
		b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(persistent[name]), name))
	}
	for i, await := range a.Awaits {
		if await.ResultType.Name != "void" {
			b.WriteString(fmt.Sprintf("  %s await_result_%d;\n", evt1CType(await.ResultType), i))
		}
	}
	for _, node := range cfg.Nodes {
		if node.Kind == "LoopHeader" && node.HasBound {
			b.WriteString(fmt.Sprintf("  int loop_iterations_%d;\n", node.Index))
		}
	}
	for _, each := range cfg.Foreaches {
		sourceType := each.Statement.SourceType
		if each.SourceByRef && sourceType.ArrayElem != nil {
			b.WriteString(fmt.Sprintf("  %s* %s;\n", evt1CType(sourceType), each.SourceField))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(sourceType), each.SourceField))
		}
		b.WriteString(fmt.Sprintf("  size_t %s;\n", each.IndexField))
		if each.Statement.SourceKind == "custom" {
			b.WriteString(fmt.Sprintf("  %s %s;\n", evt1CType(each.Statement.IteratorType), each.Iterator))
		}
	}
	b.WriteString("} " + frameName + ";\n")
	b.WriteString(fmt.Sprintf("_Static_assert(sizeof(%s) <= CONCEPT_ASYNC_FRAME_BYTES, \"Concept async frame exceeds inline capacity\");\n\n", frameName))
	if fn.ReturnType.Name != "void" {
		b.WriteString(fmt.Sprintf("_Static_assert(sizeof(%s) <= CONCEPT_ASYNC_RESULT_BYTES, \"Concept async result exceeds inline capacity\");\n\n", evt1CType(fn.ReturnType)))
	}
	b.WriteString(fmt.Sprintf("static void %s(void* storage", initName))
	for _, p := range fn.Params {
		b.WriteString(fmt.Sprintf(", %s %s", evt1CType(p.Type), p.Name))
	}
	b.WriteString(") {\n")
	b.WriteString(fmt.Sprintf("  %s* frame = (%s*)storage;\n  memset(frame, 0, sizeof(*frame));\n", frameName, frameName))
	for _, p := range fn.Params {
		if _, ok := persistent[p.Name]; ok {
			b.WriteString(fmt.Sprintf("  frame->%s = %s;\n", p.Name, p.Name))
		}
	}
	b.WriteString("}\n\n")
	b.WriteString("concept_async_operation " + symbol + "(")
	for i, p := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(evt1CType(p.Type) + " " + p.Name)
	}
	b.WriteString(") {\n  concept_async_operation operation;\n  memset(&operation, 0, sizeof(operation));\n")
	b.WriteString(fmt.Sprintf("  concept_async_push(&operation, %s);\n  %s(operation.frames[0].storage.bytes", stepName, initName))
	for _, p := range fn.Params {
		b.WriteString(", " + p.Name)
	}
	b.WriteString(");\n  return operation;\n}\n\n")
	b.WriteString(l.lowerAsyncCFG(fn, a, cfg, persistent, frameName, stepName))
	return b.String()
}

func (l *lowering) lowerAsyncStep(fn FunctionDecl, a evt1AsyncAnalysis, persistent map[string]Type, frameName, stepName string) string {
	f := newEVT1FunctionLowerer(l, fn, stepName, true)
	for _, p := range fn.Params {
		if _, ok := persistent[p.Name]; ok {
			f.scope[0][p.Name] = evt1Binding{cName: "frame->" + p.Name, t: p.Type}
		}
	}
	for name, t := range persistent {
		f.scope[0][name] = evt1Binding{cName: "frame->" + name, t: t}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static void %s(concept_async_operation* async_operation, void* storage) {\n  %s* frame = (%s*)storage;\n  switch (frame->state) {\n  case 0: {\n", stepName, frameName, frameName))
	state, awaitIndex := 0, 0
	for _, stmt := range fn.Body.Statements {
		if branch, ok := stmt.(*IfStmt); ok && evt1StatementContainsAwait(branch) {
			b.WriteString(l.lowerAsyncIf(f, branch, persistent, &state, &awaitIndex))
			continue
		}
		if loop, ok := stmt.(*WhileStmt); ok && evt1BlockContainsAwait(loop.Body) {
			b.WriteString(l.lowerAsyncWhile(f, loop, persistent, &state, &awaitIndex))
			continue
		}
		if each, ok := stmt.(*ForeachStmt); ok && evt1BlockContainsAwait(each.Body) {
			b.WriteString(l.lowerAsyncForeach(f, each, persistent, &state, &awaitIndex))
			continue
		}
		await := evt1DirectAwait(stmt)
		if await != nil {
			call, callOK := await.Value.(*CallExpr)
			var name *NameExpr
			if moved, ok := await.Value.(*MoveExpr); ok {
				name, _ = moved.Value.(*NameExpr)
			}
			nameOK := name != nil
			if nameOK {
				prelude, value, _ := f.lowerExpr(name, 2)
				b.WriteString(prelude)
				b.WriteString(f.lowerAllScopeDrops(2))
				b.WriteString(fmt.Sprintf("    frame->state = %d;\n    if (concept_async_complete(&%s)) {\n      memcpy(async_operation->child_outcome.bytes, %s.result.bytes, %s.result_size);\n      async_operation->child_outcome_size = %s.result_size;\n      goto async_resume_%d;\n    }\n    concept_async_adopt(async_operation, &%s);\n    return;\n", state+1, value, value, value, value, state+1, value))
			} else if callOK {
				var prelude strings.Builder
				var args []string
				var argTypes []Type
				for _, arg := range call.Args {
					p, value, t := f.lowerExpr(arg, 2)
					prelude.WriteString(p)
					args = append(args, value)
					argTypes = append(argTypes, t)
				}
				child, _ := evt1ResolveGeneratedCall(l.env, call.Callee, argTypes)
				for i := range args {
					if child.Params[i].Type.isBorrowLike() && !argTypes[i].isBorrowLike() {
						args[i] = "&" + args[i]
					}
				}
				b.WriteString(prelude.String())
				b.WriteString(f.lowerAllScopeDrops(2))
				b.WriteString(fmt.Sprintf("    frame->state = %d;\n    concept_async_push(async_operation, %s);\n    %s(async_operation->frames[async_operation->depth - 1u].storage.bytes", state+1, evt1AsyncStepName(l.outputBase, child.Name), evt1AsyncInitName(l.outputBase, child.Name)))
				for _, arg := range args {
					b.WriteString(", " + arg)
				}
				b.WriteString(");\n    return;\n")
			} else {
				b.WriteString("    concept_async_abort(\"R5f await operand lowering requires an async call or named operation\");\n    return;\n")
			}
			b.WriteString(fmt.Sprintf("  }\n  case %d: {\n", state+1))
			b.WriteString(fmt.Sprintf("async_resume_%d:\n", state+1))
			if await.ResultType.Name != "void" {
				b.WriteString(fmt.Sprintf("    if (async_operation->child_outcome_size != sizeof(frame->await_result_%d)) concept_async_abort(\"await result size mismatch\");\n    memcpy(&frame->await_result_%d, async_operation->child_outcome.bytes, sizeof(frame->await_result_%d));\n", awaitIndex, awaitIndex, awaitIndex))
				f.scope[0][fmt.Sprintf("#await%d", awaitIndex)] = evt1Binding{cName: fmt.Sprintf("frame->await_result_%d", awaitIndex), t: await.ResultType}
			}
			replaced := evt1ReplaceDirectAwait(stmt, awaitIndex)
			b.WriteString(l.lowerAsyncResumedStatement(f, replaced, persistent, awaitIndex, 2))
			state++
			awaitIndex++
			continue
		}
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
	}
	if fn.ReturnType.Name == "void" {
		b.WriteString("    concept_async_finish(async_operation, NULL, 0u);\n    return;\n")
	}
	b.WriteString("  }\n  default: concept_async_abort(\"invalid generated async state\");\n  }\n}\n")
	return b.String()
}

func (l *lowering) lowerAsyncIf(f *evt1FunctionLowerer, branch *IfStmt, persistent map[string]Type, state, awaitIndex *int) string {
	thenStart := *state + 1
	next := thenStart + 1
	thenHasAwait := evt1BlockContainsAwait(branch.Then)
	thenResume := thenStart
	if thenHasAwait {
		thenResume, next = next, next+1
	}
	elseStart := -1
	elseResume := -1
	elseHasAwait := branch.Else != nil && evt1BlockContainsAwait(*branch.Else)
	if branch.Else != nil {
		elseStart, next = next, next+1
		elseResume = elseStart
		if elseHasAwait {
			elseResume, next = next, next+1
		}
	}
	afterState := next
	prelude, condition, _ := f.lowerExpr(branch.Condition, 2)
	var b strings.Builder
	b.WriteString(prelude)
	falseState := afterState
	if elseStart >= 0 {
		falseState = elseStart
	}
	b.WriteString(fmt.Sprintf("    frame->state = (%s) ? %d : %d;\n    return;\n  }\n  case %d: {\n", condition, thenStart, falseState, thenStart))
	b.WriteString(l.lowerAsyncBranchBlock(f, branch.Then, persistent, thenResume, afterState, awaitIndex))
	if branch.Else != nil {
		b.WriteString(fmt.Sprintf("  case %d: {\n", elseStart))
		b.WriteString(l.lowerAsyncBranchBlock(f, *branch.Else, persistent, elseResume, afterState, awaitIndex))
	}
	b.WriteString(fmt.Sprintf("  case %d: {\n", afterState))
	*state = afterState
	return b.String()
}

func (l *lowering) lowerAsyncBranchBlock(f *evt1FunctionLowerer, block Block, persistent map[string]Type, resumeState, afterState int, awaitIndex *int) string {
	var b strings.Builder
	awaitAt := -1
	var await *AwaitExpr
	for i, stmt := range block.Statements {
		if candidate := evt1DirectAwait(stmt); candidate != nil {
			await, awaitAt = candidate, i
			break
		}
	}
	if await == nil {
		for _, stmt := range block.Statements {
			b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
		}
		b.WriteString(fmt.Sprintf("    frame->state = %d;\n    return;\n  }\n", afterState))
		return b.String()
	}
	for _, stmt := range block.Statements[:awaitAt] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
	}
	b.WriteString(l.lowerAsyncDirectCallPush(f, await, resumeState, 2))
	b.WriteString(fmt.Sprintf("  }\n  case %d: {\nasync_resume_%d:\n", resumeState, resumeState))
	idx := *awaitIndex
	if await.ResultType.Name != "void" {
		b.WriteString(fmt.Sprintf("    if (async_operation->child_outcome_size != sizeof(frame->await_result_%d)) concept_async_abort(\"await result size mismatch\");\n    memcpy(&frame->await_result_%d, async_operation->child_outcome.bytes, sizeof(frame->await_result_%d));\n", idx, idx, idx))
		f.scope[0][fmt.Sprintf("#await%d", idx)] = evt1Binding{cName: fmt.Sprintf("frame->await_result_%d", idx), t: await.ResultType}
	}
	b.WriteString(l.lowerAsyncResumedStatement(f, evt1ReplaceDirectAwait(block.Statements[awaitAt], idx), persistent, idx, 2))
	for _, stmt := range block.Statements[awaitAt+1:] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
	}
	b.WriteString(fmt.Sprintf("    frame->state = %d;\n    return;\n  }\n", afterState))
	*awaitIndex++
	return b.String()
}

func (l *lowering) lowerAsyncDirectCallPush(f *evt1FunctionLowerer, await *AwaitExpr, resumeState, indent int) string {
	call, ok := await.Value.(*CallExpr)
	if moved, movedOK := await.Value.(*MoveExpr); movedOK {
		if name, nameOK := moved.Value.(*NameExpr); nameOK {
			prelude, value, _ := f.lowerExpr(name, indent)
			var b strings.Builder
			b.WriteString(prelude)
			b.WriteString(f.lowerAllScopeDrops(indent))
			b.WriteString(ind(indent) + fmt.Sprintf("frame->state = %d;\n", resumeState))
			b.WriteString(ind(indent) + fmt.Sprintf("if (concept_async_complete(&%s)) {\n", value))
			b.WriteString(ind(indent+1) + fmt.Sprintf("memcpy(async_operation->child_outcome.bytes, %s.result.bytes, %s.result_size);\n", value, value))
			b.WriteString(ind(indent+1) + fmt.Sprintf("async_operation->child_outcome_size = %s.result_size;\n", value))
			b.WriteString(ind(indent+1) + "goto async_dispatch;\n")
			b.WriteString(ind(indent) + "}\n")
			b.WriteString(ind(indent) + fmt.Sprintf("concept_async_adopt(async_operation, &%s);\n", value))
			b.WriteString(ind(indent) + "return;\n")
			return b.String()
		}
	}
	if !ok {
		return ind(indent) + "concept_async_abort(\"control-flow await requires direct async call\");\n" + ind(indent) + "return;\n"
	}
	if call.Member {
		prelude, value, operationType := f.lowerExpr(call, indent)
		if operationType.Kind != TypeAsync {
			return ind(indent) + "concept_async_abort(\"member await did not construct Async<T>\");\n" + ind(indent) + "return;\n"
		}
		child := f.nextTemp("async_child")
		var b strings.Builder
		b.WriteString(prelude)
		b.WriteString(ind(indent) + fmt.Sprintf("concept_async_operation %s = %s;\n", child, value))
		b.WriteString(f.lowerAllScopeDrops(indent))
		b.WriteString(ind(indent) + fmt.Sprintf("frame->state = %d;\n", resumeState))
		b.WriteString(ind(indent) + fmt.Sprintf("if (concept_async_complete(&%s)) {\n", child))
		b.WriteString(ind(indent+1) + fmt.Sprintf("memcpy(async_operation->child_outcome.bytes, %s.result.bytes, %s.result_size);\n", child, child))
		b.WriteString(ind(indent+1) + fmt.Sprintf("async_operation->child_outcome_size = %s.result_size;\n", child))
		b.WriteString(ind(indent+1) + "goto async_dispatch;\n")
		b.WriteString(ind(indent) + "}\n")
		b.WriteString(ind(indent) + fmt.Sprintf("concept_async_adopt(async_operation, &%s);\n", child))
		b.WriteString(ind(indent) + "return;\n")
		return b.String()
	}
	var b strings.Builder
	var args []string
	var argTypes []Type
	for _, arg := range call.Args {
		prelude, value, t := f.lowerExpr(arg, indent)
		b.WriteString(prelude)
		args, argTypes = append(args, value), append(argTypes, t)
	}
	child, _ := evt1ResolveGeneratedCall(l.env, call.Callee, argTypes)
	for i := range args {
		if child.Params[i].Type.isBorrowLike() && !argTypes[i].isBorrowLike() {
			args[i] = "&" + args[i]
		}
	}
	b.WriteString(f.lowerAllScopeDrops(indent))
	b.WriteString(ind(indent) + fmt.Sprintf("frame->state = %d;\n", resumeState))
	b.WriteString(ind(indent) + fmt.Sprintf("concept_async_push(async_operation, %s);\n", evt1AsyncStepName(l.outputBase, child.Name)))
	b.WriteString(ind(indent) + fmt.Sprintf("%s(async_operation->frames[async_operation->depth - 1u].storage.bytes", evt1AsyncInitName(l.outputBase, child.Name)))
	for _, arg := range args {
		b.WriteString(", " + arg)
	}
	b.WriteString(");\n" + ind(indent) + "return;\n")
	return b.String()
}

func (l *lowering) lowerAsyncWhile(f *evt1FunctionLowerer, loop *WhileStmt, persistent map[string]Type, state, awaitIndex *int) string {
	loopState, resumeState, afterState := *state+1, *state+2, *state+3
	idx := *awaitIndex
	var b strings.Builder
	b.WriteString(fmt.Sprintf("    frame->foreach_index_%d = 0;\n    frame->state = %d;\n    return;\n  }\n  case %d: {\n", idx, loopState, loopState))
	if loop.Bound != nil {
		if value, err := evt1EvalExpr(newEVT1ComptimeState(l.env), f.evalScope(), loop.Bound); err == nil {
			b.WriteString(fmt.Sprintf("    if (frame->foreach_index_%d >= %d) { frame->state = %d; return; }\n", idx, value.IntValue, afterState))
		}
	}
	prelude, condition, _ := f.lowerExpr(loop.Condition, 2)
	b.WriteString(prelude)
	b.WriteString(fmt.Sprintf("    if (!(%s)) { frame->state = %d; return; }\n", condition, afterState))
	awaitAt := -1
	var await *AwaitExpr
	for i, stmt := range loop.Body.Statements {
		if candidate := evt1DirectAwait(stmt); candidate != nil {
			await, awaitAt = candidate, i
			break
		}
	}
	for _, stmt := range loop.Body.Statements[:awaitAt] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
	}
	b.WriteString(l.lowerAsyncDirectCallPush(f, await, resumeState, 2))
	b.WriteString(fmt.Sprintf("  }\n  case %d: {\nasync_resume_%d:\n", resumeState, resumeState))
	if await.ResultType.Name != "void" {
		b.WriteString(fmt.Sprintf("    if (async_operation->child_outcome_size != sizeof(frame->await_result_%d)) concept_async_abort(\"await result size mismatch\");\n    memcpy(&frame->await_result_%d, async_operation->child_outcome.bytes, sizeof(frame->await_result_%d));\n", idx, idx, idx))
		f.scope[0][fmt.Sprintf("#await%d", idx)] = evt1Binding{cName: fmt.Sprintf("frame->await_result_%d", idx), t: await.ResultType}
	}
	b.WriteString(l.lowerAsyncResumedStatement(f, evt1ReplaceDirectAwait(loop.Body.Statements[awaitAt], idx), persistent, idx, 2))
	for _, stmt := range loop.Body.Statements[awaitAt+1:] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, 2))
	}
	b.WriteString(fmt.Sprintf("    frame->foreach_index_%d = frame->foreach_index_%d + 1;\n    frame->state = %d;\n    return;\n  }\n  case %d: {\n", idx, idx, loopState, afterState))
	*state, *awaitIndex = afterState, idx+1
	return b.String()
}

func (l *lowering) lowerAsyncForeach(f *evt1FunctionLowerer, each *ForeachStmt, persistent map[string]Type, state, awaitIndex *int) string {
	source, sourceOK := each.Source.(*NameExpr)
	awaitAt := -1
	var await *AwaitExpr
	for i, stmt := range each.Body.Statements {
		if candidate := evt1DirectAwait(stmt); candidate != nil {
			if await != nil {
				sourceOK = false
				break
			}
			await, awaitAt = candidate, i
		}
	}
	if !sourceOK || await == nil || each.SourceType.ArrayElem == nil || evt1StorageHasRuntimeShape(each.SourceType) {
		return "    concept_async_abort(\"R5f foreach-await requires one await and fixed contiguous storage\");\n    return;\n"
	}
	loopState, resumeState, afterState := *state+1, *state+2, *state+3
	idx := *awaitIndex
	var b strings.Builder
	b.WriteString(fmt.Sprintf("    frame->foreach_index_%d = 0;\n    frame->state = %d;\n    return;\n  }\n  case %d: {\n", idx, loopState, loopState))
	sourcePrelude, sourceValue, _ := f.lowerExpr(source, 2)
	b.WriteString(sourcePrelude)
	count := evt1StorageElementCount(each.SourceType)
	b.WriteString(fmt.Sprintf("    if (frame->foreach_index_%d >= %d) { frame->state = %d; return; }\n", idx, count, afterState))
	f.scope[0][each.ItemName] = evt1Binding{cName: "frame->" + each.ItemName, t: each.ItemType}
	b.WriteString(fmt.Sprintf("    frame->%s = (%s).data[frame->foreach_index_%d];\n", each.ItemName, sourceValue, idx))
	for _, before := range each.Body.Statements[:awaitAt] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, before, persistent, 2))
	}
	call, ok := await.Value.(*CallExpr)
	if !ok {
		return b.String() + "    concept_async_abort(\"foreach await requires direct async call\");\n    return;\n"
	}
	var args []string
	var argTypes []Type
	for _, arg := range call.Args {
		prelude, value, t := f.lowerExpr(arg, 2)
		b.WriteString(prelude)
		args, argTypes = append(args, value), append(argTypes, t)
	}
	child, _ := evt1ResolveGeneratedCall(l.env, call.Callee, argTypes)
	for i := range args {
		if child.Params[i].Type.isBorrowLike() && !argTypes[i].isBorrowLike() {
			args[i] = "&" + args[i]
		}
	}
	b.WriteString(f.lowerAllScopeDrops(2))
	b.WriteString(fmt.Sprintf("    frame->state = %d;\n    concept_async_push(async_operation, %s);\n    %s(async_operation->frames[async_operation->depth - 1u].storage.bytes", resumeState, evt1AsyncStepName(l.outputBase, child.Name), evt1AsyncInitName(l.outputBase, child.Name)))
	for _, arg := range args {
		b.WriteString(", " + arg)
	}
	b.WriteString(fmt.Sprintf(");\n    return;\n  }\n  case %d: {\nasync_resume_%d:\n", resumeState, resumeState))
	if await.ResultType.Name != "void" {
		b.WriteString(fmt.Sprintf("    if (async_operation->child_outcome_size != sizeof(frame->await_result_%d)) concept_async_abort(\"await result size mismatch\");\n    memcpy(&frame->await_result_%d, async_operation->child_outcome.bytes, sizeof(frame->await_result_%d));\n", idx, idx, idx))
		f.scope[0][fmt.Sprintf("#await%d", idx)] = evt1Binding{cName: fmt.Sprintf("frame->await_result_%d", idx), t: await.ResultType}
	}
	replaced := evt1ReplaceDirectAwait(each.Body.Statements[awaitAt], idx)
	b.WriteString(l.lowerAsyncResumedStatement(f, replaced, persistent, idx, 2))
	for _, after := range each.Body.Statements[awaitAt+1:] {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, after, persistent, 2))
	}
	b.WriteString(fmt.Sprintf("    frame->foreach_index_%d = frame->foreach_index_%d + 1;\n    frame->state = %d;\n    return;\n  }\n  case %d: {\n", idx, idx, loopState, afterState))
	*state, *awaitIndex = afterState, idx+1
	return b.String()
}

func (l *lowering) lowerAsyncOrdinaryStatement(f *evt1FunctionLowerer, stmt Statement, persistent map[string]Type, indent int) string {
	if v, ok := stmt.(*VarDecl); ok {
		if _, persisted := persistent[v.Name]; persisted {
			prelude, value, _ := f.lowerExprExpected(v.Value, v.Type, indent)
			f.scope[0][v.Name] = evt1Binding{cName: "frame->" + v.Name, t: v.Type}
			return prelude + ind(indent) + fmt.Sprintf("frame->%s = %s;\n", v.Name, value)
		}
	}
	if r, ok := stmt.(*ReturnStmt); ok {
		if r.Value == nil {
			return l.lowerAsyncPersistentDrops(f, persistent, indent) + ind(indent) + "concept_async_finish(async_operation, NULL, 0u);\n" + ind(indent) + "return;\n"
		}
		prelude, value, t := f.lowerExprExpected(r.Value, fnEventualType(f.fn), indent)
		temp := f.nextTemp("async_return")
		return prelude + ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(t), temp, value) + l.lowerAsyncPersistentDrops(f, persistent, indent) + ind(indent) + fmt.Sprintf("concept_async_finish(async_operation, &%s, sizeof(%s));\n", temp, temp) + ind(indent) + "return;\n"
	}
	return f.lowerStatement(stmt, indent)
}

func (l *lowering) lowerAsyncPersistentDrops(f *evt1FunctionLowerer, persistent map[string]Type, indent int) string {
	var names []string
	for name, t := range persistent {
		if evt1TypeHasDrop(l.env, t) {
			names = append(names, name)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	var b strings.Builder
	for _, name := range names {
		b.WriteString(f.lowerDropValue(persistent[name], "frame->"+name, indent))
	}
	return b.String()
}

func fnEventualType(fn FunctionDecl) Type { return fn.ReturnType }

func (l *lowering) lowerAsyncResumedStatement(f *evt1FunctionLowerer, stmt Statement, persistent map[string]Type, awaitIndex, indent int) string {
	if expression, ok := stmt.(*ExprStmt); ok {
		if name, ok := expression.Value.(*NameExpr); ok && name.Name == fmt.Sprintf("#await%d", awaitIndex) {
			// A void await has no value materialization after the continuation.
			return ""
		}
	}
	if v, ok := stmt.(*VarDecl); ok {
		if failure, ok := v.Value.(*FailureExpr); ok && failure.Op == "?" {
			carrier := fmt.Sprintf("frame->await_result_%d", awaitIndex)
			carrierType := failure.ResolvedType
			if evt1IsResultType(carrierType) && evt1IsResultType(f.fn.ReturnType) {
				temp := f.nextTemp("async_error")
				var b strings.Builder
				b.WriteString(ind(indent) + fmt.Sprintf("if (%s.tag == 1) {\n", carrier))
				b.WriteString(ind(indent+1) + fmt.Sprintf("%s %s = %s(%s.payload.error.error);\n", evt1CType(f.fn.ReturnType), temp, evt1FailureConstructorName(f.fn.ReturnType, "Error"), carrier))
				b.WriteString(l.lowerAsyncPersistentDrops(f, persistent, indent+1))
				b.WriteString(ind(indent+1) + fmt.Sprintf("concept_async_finish(async_operation, &%s, sizeof(%s));\n", temp, temp))
				b.WriteString(ind(indent+1) + "return;\n" + ind(indent) + "}\n")
				if _, persisted := persistent[v.Name]; persisted {
					f.scope[0][v.Name] = evt1Binding{cName: "frame->" + v.Name, t: v.Type}
					b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = %s.payload.ok.value;\n", v.Name, carrier))
				} else {
					f.scope[0][v.Name] = evt1Binding{cName: v.Name, t: v.Type}
					b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s.payload.ok.value;\n", evt1CType(v.Type), v.Name, carrier))
				}
				return b.String()
			}
		}
	}
	return l.lowerAsyncOrdinaryStatement(f, stmt, persistent, indent)
}

func evt1DirectAwait(stmt Statement) *AwaitExpr {
	var find func(Expr) *AwaitExpr
	find = func(e Expr) *AwaitExpr {
		switch x := e.(type) {
		case *AwaitExpr:
			return x
		case *FailureExpr:
			return find(x.Value)
		case *ParenExpr:
			return find(x.Value)
		case *UnaryExpr:
			return find(x.Value)
		case *BinaryExpr:
			if await := find(x.Left); await != nil {
				return await
			}
			return find(x.Right)
		}
		return nil
	}
	switch s := stmt.(type) {
	case *VarDecl:
		return find(s.Value)
	case *AssignStmt:
		return find(s.Value)
	case *ExprStmt:
		return find(s.Value)
	case *ReturnStmt:
		return find(s.Value)
	}
	return nil
}

func evt1ReplaceDirectAwait(stmt Statement, index int) Statement {
	var replace func(Expr) Expr
	replace = func(e Expr) Expr {
		switch x := e.(type) {
		case *AwaitExpr:
			return &NameExpr{Name: fmt.Sprintf("#await%d", index), Span: x.Span}
		case *FailureExpr:
			copy := *x
			copy.Value = replace(x.Value)
			return &copy
		case *ParenExpr:
			copy := *x
			copy.Value = replace(x.Value)
			return &copy
		case *UnaryExpr:
			copy := *x
			copy.Value = replace(x.Value)
			return &copy
		case *BinaryExpr:
			copy := *x
			copy.Left, copy.Right = replace(x.Left), replace(x.Right)
			return &copy
		default:
			return e
		}
	}
	switch s := stmt.(type) {
	case *VarDecl:
		copy := *s
		copy.Value = replace(s.Value)
		return &copy
	case *AssignStmt:
		copy := *s
		copy.Value = replace(s.Value)
		return &copy
	case *ExprStmt:
		copy := *s
		copy.Value = replace(s.Value)
		return &copy
	case *ReturnStmt:
		copy := *s
		copy.Value = replace(s.Value)
		return &copy
	}
	return stmt
}

// evt1AsyncAnalysis is the bounded, per-function continuation analysis. It is
// deliberately not whole-program SSA/NLL: source order plus lexical name use
// is sufficient to determine the storage mechanically required by each await.
type evt1AsyncAnalysis struct {
	Awaits          []*AwaitExpr
	AwaitOrder      []int
	Uses            map[string][]int
	DeclOrder       map[string]int
	DeclTypes       map[string]Type
	Iterators       map[int]string
	IteratorSources map[int][]string
	IteratorItems   map[int]MIRName
	AwaitContexts   map[int]string
	Deps            map[string][]string
}

func evt1AnalyzeAsync(fn FunctionDecl) evt1AsyncAnalysis {
	a := evt1AsyncAnalysis{Uses: map[string][]int{}, DeclOrder: map[string]int{}, DeclTypes: map[string]Type{}, Iterators: map[int]string{}, IteratorSources: map[int][]string{}, IteratorItems: map[int]MIRName{}, AwaitContexts: map[int]string{}, Deps: map[string][]string{}}
	for _, p := range fn.Params {
		a.DeclOrder[p.Name], a.DeclTypes[p.Name] = -1, p.Type
	}
	order := 0
	var expr func(Expr)
	expr = func(value Expr) {
		switch e := value.(type) {
		case *AwaitExpr:
			a.Awaits = append(a.Awaits, e)
			a.AwaitOrder = append(a.AwaitOrder, order)
			expr(e.Value)
		case *NameExpr:
			a.Uses[e.Name] = append(a.Uses[e.Name], order)
		case *ParenExpr:
			expr(e.Value)
		case *UnaryExpr:
			expr(e.Value)
		case *MoveExpr:
			expr(e.Value)
		case *RefExpr:
			expr(e.Value)
		case *FailureExpr:
			expr(e.Value)
		case *BindExpr:
			expr(e.Source)
		case *FieldExpr:
			expr(e.Receiver)
		case *IndexExpr:
			expr(e.Base)
			for _, i := range evt1StorageIndices(e) {
				expr(i)
			}
		case *BinaryExpr:
			expr(e.Left)
			expr(e.Right)
		case *CallExpr:
			for _, x := range e.Args {
				expr(x)
			}
		case *TemplateCallExpr:
			for _, x := range e.Args {
				expr(x)
			}
		case *ConstructExpr:
			for _, x := range e.Args {
				expr(x)
			}
		case *StructConstructExpr:
			for _, x := range e.Args {
				expr(x)
			}
		case *ArrayLiteralExpr:
			for _, x := range e.Elements {
				expr(x)
			}
		case *WithExpr:
			expr(e.Base)
			for _, u := range e.Updates {
				expr(u.Value)
			}
		case *IfExpr:
			expr(e.Condition)
			expr(e.Then)
			expr(e.Else)
		case *MatchExpr:
			expr(e.Subject)
			for _, arm := range e.Arms {
				expr(arm.Value)
			}
		}
	}
	var block func(Block)
	block = func(b Block) {
		for _, stmt := range b.Statements {
			order++
			switch s := stmt.(type) {
			case *VarDecl:
				a.DeclOrder[s.Name], a.DeclTypes[s.Name] = order, s.Type
				a.Deps[s.Name] = evt1ExprNames(s.Value)
				expr(s.Value)
			case *AssignStmt:
				expr(s.Target)
				expr(s.Value)
			case *ReturnStmt:
				expr(s.Value)
			case *ExprStmt:
				expr(s.Value)
			case *AssertStmt:
				expr(s.Condition)
				expr(s.Reason)
			case *IfStmt:
				expr(s.Condition)
				before := len(a.Awaits)
				block(s.Then)
				if s.Else != nil {
					block(*s.Else)
				}
				for i := before; i < len(a.Awaits); i++ {
					a.AwaitContexts[i] = "if"
				}
			case *WhileStmt:
				expr(s.Condition)
				before := len(a.Awaits)
				block(s.Body)
				for i := before; i < len(a.Awaits); i++ {
					a.AwaitContexts[i] = "while"
				}
			case *ForeachStmt:
				expr(s.Source)
				a.DeclOrder[s.ItemName], a.DeclTypes[s.ItemName] = order, s.ItemType
				before := len(a.Awaits)
				block(s.Body)
				for i := before; i < len(a.Awaits); i++ {
					a.Iterators[i] = fmt.Sprintf("#iterator@%d", s.Span.Line)
					a.IteratorSources[i] = evt1ExprNames(s.Source)
					a.IteratorItems[i] = MIRName{Name: s.ItemName, Type: s.ItemType}
				}
			case *MatchStmt:
				expr(s.Subject)
				for _, arm := range s.Arms {
					block(arm.Block)
				}
			case *TryStmt:
				block(s.Body)
				for _, arm := range s.Except {
					block(arm.Body)
				}
			case *Block:
				block(*s)
			}
		}
	}
	if fn.Body != nil {
		block(*fn.Body)
	}
	return a
}

func (a evt1AsyncAnalysis) liveAcross(index int) []string {
	if index < 0 || index >= len(a.AwaitOrder) {
		return nil
	}
	at := a.AwaitOrder[index]
	var live []string
	for name, decl := range a.DeclOrder {
		if decl > at {
			continue
		}
		for _, use := range a.Uses[name] {
			if use > at {
				live = append(live, name)
				break
			}
		}
	}
	if iterator := a.Iterators[index]; iterator != "" {
		live = append(live, iterator)
	}
	live = append(live, a.IteratorSources[index]...)
	if item := a.IteratorItems[index]; item.Name != "" {
		live = append(live, item.Name)
	}
	seen := map[string]bool{}
	for _, name := range live {
		seen[name] = true
	}
	for changed := true; changed; {
		changed = false
		for name := range seen {
			t := a.DeclTypes[name]
			if !t.isBorrowLike() && !evt1IsTensorType(t) && !evt1IsSpanType(t) {
				continue
			}
			for _, dep := range a.Deps[name] {
				if _, declared := a.DeclTypes[dep]; declared && !seen[dep] {
					seen[dep], changed = true, true
				}
			}
		}
	}
	live = live[:0]
	for name := range seen {
		live = append(live, name)
	}
	sort.Strings(live)
	return live
}

func evt1ExprNames(expr Expr) []string {
	seen := map[string]bool{}
	var visit func(Expr)
	visit = func(value Expr) {
		switch e := value.(type) {
		case *NameExpr:
			seen[e.Name] = true
		case *AwaitExpr:
			visit(e.Value)
		case *ParenExpr:
			visit(e.Value)
		case *UnaryExpr:
			visit(e.Value)
		case *MoveExpr:
			visit(e.Value)
		case *RefExpr:
			visit(e.Value)
		case *FailureExpr:
			visit(e.Value)
		case *CallExpr:
			for _, x := range e.Args {
				visit(x)
			}
		case *FieldExpr:
			visit(e.Receiver)
		case *BinaryExpr:
			visit(e.Left)
			visit(e.Right)
		case *IndexExpr:
			visit(e.Base)
			for _, x := range evt1StorageIndices(e) {
				visit(x)
			}
		}
	}
	visit(expr)
	var out []string
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func evt1AsyncType(eventual Type, origin string, span Span) Type {
	return Type{Name: "Async", Kind: TypeAsync, TypeArgs: []Type{eventual}, AsyncOrigin: origin, Span: span}
}

func evt1BuildMIRAsync(fn FunctionDecl, env *semanticEnv) *MIRAsyncFunction {
	if !fn.Async {
		return nil
	}
	a := evt1AnalyzeAsync(fn)
	cfg := evt1NormalizeAsyncCFG(fn, env)
	identity := fn.Name + "#async"
	out := &MIRAsyncFunction{
		Identity: identity, MachineIdentity: identity + "#machine", StateIdentity: identity + "#state",
		OperationType: evt1AsyncType(evt1MIRType(env, fn.ReturnType), fn.Name, fn.Span), EventualType: evt1MIRType(env, fn.ReturnType),
		FrameStorage: "Inline", ContinuationStrategy: "ExplicitGeneratedState", ChildInvocation: "MachinePush", Scheduler: "None", SavedPC: "None",
		GeneratedStates: evt1AsyncCFGGeneratedNames(cfg), ControlFlowStrategy: "StructuredStateGraph",
	}
	persistent := map[string]bool{}
	for _, p := range fn.Params {
		persistent[p.Name] = true
	}
	for i, await := range a.Awaits {
		live := a.liveAcross(i)
		for _, name := range live {
			if name[0] != '#' {
				persistent[name] = true
			}
		}
		operandType := evt1AsyncType(await.ResultType, "", await.Span)
		if call, ok := await.Value.(*CallExpr); ok {
			operandType.AsyncOrigin = call.Callee
		}
		continuation := fmt.Sprintf("%s#AfterAwait%d", fn.Name, i)
		if cfg != nil {
			for _, state := range cfg.Nodes {
				if state.Kind == "AfterAwait" && state.AwaitIndex == i {
					continuation = state.Identity
					break
				}
			}
		}
		out.AwaitPoints = append(out.AwaitPoints, MIRAwaitPoint{Index: i, Continuation: continuation, Operand: evt1ExprIdentity(await.Value), OperandType: operandType, ResultType: await.ResultType, LiveAcross: live, Evaluation: "ExactlyOnce", ChildPush: "BoundedMachineFramePush", OutcomeConsume: "ExactlyOnce", SourceSpan: await.Span})
	}
	if cfg != nil {
		out.States, out.Edges = cfg.mir(evt1AsyncPersistentMap(fn, a, cfg))
		out.BranchCount, out.JoinCount, out.LoopCount = cfg.BranchCount, cfg.JoinCount, cfg.LoopCount
	}
	for name, t := range evt1AsyncPersistentMap(fn, a, cfg) {
		if _, exists := persistent[name]; !exists {
			persistent[name] = true
			a.DeclTypes[name] = t
		}
	}
	for name := range persistent {
		out.PersistentFields = append(out.PersistentFields, MIRName{Name: name, Type: evt1MIRType(env, a.DeclTypes[name])})
	}
	sort.Slice(out.PersistentFields, func(i, j int) bool { return out.PersistentFields[i].Name < out.PersistentFields[j].Name })
	return out
}

func evt1ValidateAsyncShape(fn FunctionDecl) error {
	if !fn.Async || fn.Body == nil {
		return nil
	}
	var validateStructured func(Block) error
	validateStructured = func(block Block) error {
		for _, stmt := range block.Statements {
			if count := evt1SimpleStatementAwaitCount(stmt); count > 0 && (count != 1 || evt1DirectAwait(stmt) == nil) {
				return evt1Diagnostic("ASYNC_CONTROL_FLOW_NOT_REDUCIBLE", "this await expression shape cannot be split into one explicit continuation state", stmt.statementSpan())
			}
			switch s := stmt.(type) {
			case *IfStmt:
				if evt1ExprContainsAwait(s.Condition) {
					return evt1Diagnostic("ASYNC_CONTROL_FLOW_NOT_REDUCIBLE", "await in an if condition is deferred; use an explicit preceding await", s.Span)
				}
				if err := validateStructured(s.Then); err != nil {
					return err
				}
				if s.Else != nil {
					if err := validateStructured(*s.Else); err != nil {
						return err
					}
				}
			case *WhileStmt:
				if evt1ExprContainsAwait(s.Condition) {
					return evt1Diagnostic("ASYNC_CONTROL_FLOW_NOT_REDUCIBLE", "await in a while condition is deferred; use an explicit preceding await", s.Span)
				}
				if err := validateStructured(s.Body); err != nil {
					return err
				}
			case *MatchStmt:
				if evt1ExprContainsAwait(s.Subject) {
					return evt1Diagnostic("ASYNC_MATCH_NORMALIZATION_INVALID", "await in a match subject is deferred; use an explicit preceding await", s.Span)
				}
				for _, arm := range s.Arms {
					if err := validateStructured(arm.Block); err != nil {
						return err
					}
				}
			case *ForeachStmt:
				if evt1ExprContainsAwait(s.Source) {
					return evt1Diagnostic("ASYNC_FOREACH_NORMALIZATION_INVALID", "await in a foreach source is deferred; use an explicit preceding await", s.Span)
				}
				if err := validateStructured(s.Body); err != nil {
					return err
				}
			case *TryStmt:
				if err := validateStructured(s.Body); err != nil {
					return err
				}
				for _, arm := range s.Except {
					if err := validateStructured(arm.Body); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	return validateStructured(*fn.Body)
}

func evt1SimpleStatementAwaitCount(stmt Statement) int {
	switch s := stmt.(type) {
	case *VarDecl:
		return evt1ExprAwaitCount(s.Value)
	case *AssignStmt:
		return evt1ExprAwaitCount(s.Value)
	case *ExprStmt:
		return evt1ExprAwaitCount(s.Value)
	case *ReturnStmt:
		return evt1ExprAwaitCount(s.Value)
	}
	return 0
}

func evt1ExprAwaitCount(expr Expr) int {
	switch e := expr.(type) {
	case nil:
		return 0
	case *AwaitExpr:
		return 1
	case *ParenExpr:
		return evt1ExprAwaitCount(e.Value)
	case *UnaryExpr:
		return evt1ExprAwaitCount(e.Value)
	case *MoveExpr:
		return evt1ExprAwaitCount(e.Value)
	case *RefExpr:
		return evt1ExprAwaitCount(e.Value)
	case *FailureExpr:
		return evt1ExprAwaitCount(e.Value)
	case *BinaryExpr:
		return evt1ExprAwaitCount(e.Left) + evt1ExprAwaitCount(e.Right)
	case *CallExpr:
		count := 0
		for _, arg := range e.Args {
			count += evt1ExprAwaitCount(arg)
		}
		return count
	}
	return 0
}

func evt1AwaitCountBlock(block Block) int {
	count := 0
	for _, stmt := range block.Statements {
		if evt1StatementContainsAwait(stmt) {
			count++
		}
	}
	return count
}

func evt1ValidateAsyncPersistence(fn FunctionDecl) error {
	if !fn.Async {
		return nil
	}
	a := evt1AnalyzeAsync(fn)
	persistent := map[string]bool{}
	for _, p := range fn.Params {
		persistent[p.Name] = true
	}
	for i := range a.Awaits {
		for _, name := range a.liveAcross(i) {
			persistent[name] = true
		}
	}
	params := map[string]bool{}
	for _, p := range fn.Params {
		params[p.Name] = true
	}
	for name := range persistent {
		t := a.DeclTypes[name]
		if t.isReference() && (!params[name] || t.Scoped) {
			return evt1Diagnostic("ASYNC_PERSISTENT_REF_ESCAPE", fmt.Sprintf("reference %s cannot be proven to outlive async operation %s", name, fn.Name), t.Span)
		}
	}
	return nil
}

func evt1BlockContainsAwait(block Block) bool {
	for _, stmt := range block.Statements {
		if evt1StatementContainsAwait(stmt) {
			return true
		}
	}
	return false
}

func evt1StatementContainsAwait(stmt Statement) bool {
	switch s := stmt.(type) {
	case *VarDecl:
		return evt1ExprContainsAwait(s.Value)
	case *AssignStmt:
		return evt1ExprContainsAwait(s.Target) || evt1ExprContainsAwait(s.Value)
	case *ReturnStmt:
		return evt1ExprContainsAwait(s.Value)
	case *ExprStmt:
		return evt1ExprContainsAwait(s.Value)
	case *IfStmt:
		return evt1ExprContainsAwait(s.Condition) || evt1BlockContainsAwait(s.Then) || (s.Else != nil && evt1BlockContainsAwait(*s.Else))
	case *WhileStmt:
		return evt1ExprContainsAwait(s.Condition) || evt1BlockContainsAwait(s.Body)
	case *ForeachStmt:
		return evt1ExprContainsAwait(s.Source) || evt1BlockContainsAwait(s.Body)
	case *MatchStmt:
		if evt1ExprContainsAwait(s.Subject) {
			return true
		}
		for _, arm := range s.Arms {
			if evt1BlockContainsAwait(arm.Block) {
				return true
			}
		}
	case *TryStmt:
		if evt1BlockContainsAwait(s.Body) {
			return true
		}
		for _, arm := range s.Except {
			if evt1BlockContainsAwait(arm.Body) {
				return true
			}
		}
	case *Block:
		return evt1BlockContainsAwait(*s)
	}
	return false
}

func evt1ExprContainsAwait(expr Expr) bool {
	if expr == nil {
		return false
	}
	if _, ok := expr.(*AwaitExpr); ok {
		return true
	}
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1ExprContainsAwait(e.Value)
	case *UnaryExpr:
		return evt1ExprContainsAwait(e.Value)
	case *MoveExpr:
		return evt1ExprContainsAwait(e.Value)
	case *RefExpr:
		return evt1ExprContainsAwait(e.Value)
	case *FailureExpr:
		return evt1ExprContainsAwait(e.Value)
	case *BinaryExpr:
		return evt1ExprContainsAwait(e.Left) || evt1ExprContainsAwait(e.Right)
	case *CallExpr:
		for _, x := range e.Args {
			if evt1ExprContainsAwait(x) {
				return true
			}
		}
	}
	return false
}
