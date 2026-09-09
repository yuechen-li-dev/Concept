package concept

import (
	"fmt"
	"sort"
	"strings"
)

// evt1AsyncCFG is the explicit boundary between structured async source
// and generated machine-state lowering. Nodes contain structured semantic
// operations, never instruction addresses or continuation pointers.
type evt1AsyncCFG struct {
	Function     string
	Entry        *evt1AsyncCFGNode
	Nodes        []*evt1AsyncCFGNode
	Foreaches    []evt1AsyncCFGForeach
	MatchFields  map[string]Type
	BranchCount  int
	JoinCount    int
	LoopCount    int
	awaitIndices map[*AwaitExpr]int
}

type evt1AsyncCFGNode struct {
	Index      int
	Identity   string
	Kind       string
	Span       Span
	Statements []Statement
	Next       *evt1AsyncCFGNode
	Await      *AwaitExpr
	AwaitIndex int
	Resume     Statement
	Condition  Expr
	True       *evt1AsyncCFGNode
	False      *evt1AsyncCFGNode
	Match      *MatchStmt
	MatchArms  []evt1AsyncCFGMatchArm
	Foreach    *ForeachStmt
	ForeachID  int
	Body       *evt1AsyncCFGNode
	Exit       *evt1AsyncCFGNode
	LoopBound  int
	HasBound   bool
	Handlers   map[string]evt1AsyncCFGHandler
}

type evt1AsyncCFGHandler struct {
	Target  *evt1AsyncCFGNode
	Binding string
	Type    Type
}

type evt1AsyncCFGMatchArm struct {
	Pattern Pattern
	Target  *evt1AsyncCFGNode
}

type evt1AsyncCFGForeach struct {
	ID          int
	Statement   *ForeachStmt
	SourceField string
	SourceByRef bool
	IndexField  string
	Iterator    string
}

type evt1AsyncCFGBuilder struct {
	cfg      *evt1AsyncCFG
	created  []*evt1AsyncCFGNode
	complete *evt1AsyncCFGNode
	handlers map[string]evt1AsyncCFGHandler
}

func evt1NormalizeAsyncCFG(fn FunctionDecl, env *semanticEnv) *evt1AsyncCFG {
	if !fn.Async || fn.Body == nil {
		return nil
	}
	analysis := evt1AnalyzeAsync(fn)
	cfg := &evt1AsyncCFG{Function: fn.Name, MatchFields: map[string]Type{}, awaitIndices: map[*AwaitExpr]int{}}
	for i, await := range analysis.Awaits {
		cfg.awaitIndices[await] = i
	}
	b := &evt1AsyncCFGBuilder{cfg: cfg}
	b.complete = b.node("Complete", fn.Span)
	cfg.Entry = b.compile(fn.Body.Statements, b.complete, env)
	b.finalize()
	return cfg
}

func (b *evt1AsyncCFGBuilder) node(kind string, span Span) *evt1AsyncCFGNode {
	n := &evt1AsyncCFGNode{Index: -1, Kind: kind, Span: span, AwaitIndex: -1, ForeachID: -1}
	b.created = append(b.created, n)
	return n
}

func (b *evt1AsyncCFGBuilder) compile(statements []Statement, next *evt1AsyncCFGNode, env *semanticEnv) *evt1AsyncCFGNode {
	if len(statements) == 0 {
		return next
	}
	ordinary := make([]Statement, 0, len(statements))
	for i, stmt := range statements {
		if await := evt1DirectAwait(stmt); await != nil {
			after := b.compile(statements[i+1:], next, env)
			resume := b.node("AfterAwait", stmt.statementSpan())
			resume.AwaitIndex, resume.Resume, resume.Next = b.cfg.awaitIndices[await], evt1ReplaceDirectAwait(stmt, b.cfg.awaitIndices[await]), after
			resume.Handlers = b.cloneHandlers()
			suspend := b.node("AwaitSuspend", await.Span)
			suspend.Statements, suspend.Await, suspend.AwaitIndex, suspend.Next = ordinary, await, b.cfg.awaitIndices[await], resume
			return suspend
		}
		switch s := stmt.(type) {
		case *IfStmt:
			if evt1StatementContainsAwait(s) {
				after := b.compile(statements[i+1:], next, env)
				join := b.node("BranchJoin", s.Span)
				join.Next = after
				b.cfg.JoinCount++
				thenEntry := b.compile(s.Then.Statements, join, env)
				elseEntry := join
				if s.Else != nil {
					elseEntry = b.compile(s.Else.Statements, join, env)
				}
				branch := b.node("Branch", s.Span)
				branch.Statements, branch.Condition, branch.True, branch.False = ordinary, s.Condition, thenEntry, elseEntry
				b.cfg.BranchCount++
				return branch
			}
		case *MatchStmt:
			if evt1StatementContainsAwait(s) {
				after := b.compile(statements[i+1:], next, env)
				join := b.node("MatchJoin", s.Span)
				join.Next = after
				b.cfg.JoinCount++
				dispatch := b.node("MatchDispatch", s.Span)
				dispatch.Statements, dispatch.Match = ordinary, s
				for _, arm := range s.Arms {
					dispatch.MatchArms = append(dispatch.MatchArms, evt1AsyncCFGMatchArm{Pattern: arm.Pattern, Target: b.compile(arm.Block.Statements, join, env)})
					if decl, ok := env.enums[arm.Pattern.EnumName]; ok {
						if variant, ok := evt1LookupVariant(decl, arm.Pattern.VariantName); ok {
							for j, name := range arm.Pattern.Bindings {
								if evt1BlockUsesName(arm.Block, name) {
									b.cfg.MatchFields[name] = evt1CanonicalType(env, variant.Payload[j].Type)
								}
							}
						}
					}
				}
				b.cfg.BranchCount++
				return dispatch
			}
		case *WhileStmt:
			if evt1BlockContainsAwait(s.Body) {
				after := b.compile(statements[i+1:], next, env)
				init := b.node("LoopInit", s.Span)
				header := b.node("LoopHeader", s.Span)
				back := b.node("LoopBackedge", s.Span)
				init.Statements, init.Next = ordinary, header
				back.Next = header
				header.Condition, header.False = s.Condition, after
				if s.Bound != nil {
					if value, err := evt1EvalExpr(newEVT1ComptimeState(env), nil, s.Bound); err == nil {
						header.HasBound, header.LoopBound = true, value.IntValue
					}
				}
				header.True = b.compile(s.Body.Statements, back, env)
				b.cfg.LoopCount++
				return init
			}
		case *ForeachStmt:
			if evt1BlockContainsAwait(s.Body) {
				after := b.compile(statements[i+1:], next, env)
				id := len(b.cfg.Foreaches)
				info := evt1AsyncCFGForeach{ID: id, Statement: s, SourceField: fmt.Sprintf("foreach_source_%d", id), IndexField: fmt.Sprintf("foreach_index_%d", id), Iterator: fmt.Sprintf("foreach_iterator_%d", id)}
				_, info.SourceByRef = s.Source.(*NameExpr)
				b.cfg.Foreaches = append(b.cfg.Foreaches, info)
				init := b.node("ForeachInit", s.Span)
				header := b.node("ForeachHeader", s.Span)
				back := b.node("ForeachContinue", s.Span)
				init.Statements, init.Foreach, init.ForeachID, init.Next = ordinary, s, id, header
				header.Foreach, header.ForeachID, header.Exit = s, id, after
				back.Foreach, back.ForeachID, back.Next = s, id, header
				header.Body = b.compile(s.Body.Statements, back, env)
				b.cfg.LoopCount++
				return init
			}
		case *TryStmt:
			if evt1StatementContainsAwait(s) {
				after := b.compile(statements[i+1:], next, env)
				join := b.node("TryJoin", s.Span)
				join.Next = after
				b.cfg.JoinCount++
				handlers := map[string]evt1AsyncCFGHandler{}
				for _, arm := range s.Except {
					t := evt1CanonicalType(env, arm.ErrorType)
					b.cfg.MatchFields[arm.Binding] = t
					handlers[evt1TypeIdentity(t)] = evt1AsyncCFGHandler{Target: b.compile(arm.Body.Statements, join, env), Binding: arm.Binding, Type: t}
				}
				prior := b.handlers
				b.handlers = handlers
				body := b.compile(s.Body.Statements, join, env)
				b.handlers = prior
				if len(ordinary) == 0 {
					return body
				}
				entry := b.node("BasicBlock", s.Span)
				entry.Statements, entry.Next = ordinary, body
				return entry
			}
		}
		ordinary = append(ordinary, stmt)
	}
	n := b.node("BasicBlock", statements[0].statementSpan())
	n.Statements, n.Next = ordinary, next
	return n
}

func evt1BlockUsesName(block Block, sought string) bool {
	usesExpr := func(expr Expr) bool {
		for _, name := range evt1ExprNames(expr) {
			if name == sought {
				return true
			}
		}
		return false
	}
	var usesBlock func(Block) bool
	usesBlock = func(current Block) bool {
		for _, stmt := range current.Statements {
			switch s := stmt.(type) {
			case *VarDecl:
				if usesExpr(s.Value) {
					return true
				}
			case *AssignStmt:
				if usesExpr(s.Target) || usesExpr(s.Value) {
					return true
				}
			case *ReturnStmt:
				if usesExpr(s.Value) {
					return true
				}
			case *ExprStmt:
				if usesExpr(s.Value) {
					return true
				}
			case *AssertStmt:
				if usesExpr(s.Condition) || usesExpr(s.Reason) {
					return true
				}
			case *IfStmt:
				if usesExpr(s.Condition) || usesBlock(s.Then) || (s.Else != nil && usesBlock(*s.Else)) {
					return true
				}
			case *WhileStmt:
				if usesExpr(s.Condition) || usesBlock(s.Body) {
					return true
				}
			case *ForeachStmt:
				if usesExpr(s.Source) || usesBlock(s.Body) {
					return true
				}
			case *MatchStmt:
				if usesExpr(s.Subject) {
					return true
				}
				for _, arm := range s.Arms {
					if usesBlock(arm.Block) {
						return true
					}
				}
			case *TryStmt:
				if usesBlock(s.Body) {
					return true
				}
				for _, arm := range s.Except {
					if usesBlock(arm.Body) {
						return true
					}
				}
			case *Block:
				if usesBlock(*s) {
					return true
				}
			}
		}
		return false
	}
	return usesBlock(block)
}

func (b *evt1AsyncCFGBuilder) cloneHandlers() map[string]evt1AsyncCFGHandler {
	if len(b.handlers) == 0 {
		return nil
	}
	out := map[string]evt1AsyncCFGHandler{}
	for key, handler := range b.handlers {
		out[key] = handler
	}
	return out
}

func (b *evt1AsyncCFGBuilder) finalize() {
	seen := map[*evt1AsyncCFGNode]bool{}
	var visit func(*evt1AsyncCFGNode)
	visit = func(n *evt1AsyncCFGNode) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		n.Index = len(b.cfg.Nodes)
		b.cfg.Nodes = append(b.cfg.Nodes, n)
		visit(n.Next)
		visit(n.True)
		visit(n.False)
		for _, arm := range n.MatchArms {
			visit(arm.Target)
		}
		var handlerKeys []string
		for key := range n.Handlers {
			handlerKeys = append(handlerKeys, key)
		}
		sort.Strings(handlerKeys)
		for _, key := range handlerKeys {
			visit(n.Handlers[key].Target)
		}
		visit(n.Body)
		visit(n.Exit)
	}
	visit(b.cfg.Entry)
	counters := map[string]int{}
	for _, n := range b.cfg.Nodes {
		kind := n.Kind
		ordinal := counters[kind]
		counters[kind]++
		n.Identity = fmt.Sprintf("%s#%s%d", b.cfg.Function, kind, ordinal)
	}
}

func (cfg *evt1AsyncCFG) mir(persistent map[string]Type) ([]MIRAsyncState, []MIRAsyncEdge) {
	var states []MIRAsyncState
	var edges []MIRAsyncEdge
	declaredAt, usesAt := evt1AsyncGraphNameSites(cfg)
	add := func(from *evt1AsyncCFGNode, to *evt1AsyncCFGNode, kind string, await int, source string) {
		if from != nil && to != nil {
			edges = append(edges, MIRAsyncEdge{From: from.Identity, To: to.Identity, Kind: kind, AwaitIndex: await, Source: source})
		}
	}
	for _, n := range cfg.Nodes {
		var active []string
		for name := range persistent {
			if usesAt[name][n] || declaredAt[name] == n {
				if _, stored := persistent[name]; stored {
					active = append(active, name)
				}
			}
		}
		sort.Strings(active)
		states = append(states, MIRAsyncState{Identity: n.Identity, Kind: n.Kind, ActiveFields: active})
		switch n.Kind {
		case "Branch", "LoopHeader":
			add(n, n.True, "true", -1, evt1ExprIdentity(n.Condition))
			add(n, n.False, "false", -1, evt1ExprIdentity(n.Condition))
		case "MatchDispatch":
			for _, arm := range n.MatchArms {
				add(n, arm.Target, "match", -1, arm.Pattern.EnumName+"::"+arm.Pattern.VariantName)
			}
		case "ForeachHeader":
			add(n, n.Body, "foreach_item", -1, n.Foreach.ItemName)
			add(n, n.Exit, "foreach_exit", -1, n.Foreach.ItemName)
		case "AwaitSuspend":
			add(n, n.Next, "await_resume", n.AwaitIndex, evt1ExprIdentity(n.Await.Value))
		case "AfterAwait":
			add(n, n.Next, "next", -1, "")
			var keys []string
			for key := range n.Handlers {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				add(n, n.Handlers[key].Target, "failure_handler", n.AwaitIndex, key)
			}
		default:
			add(n, n.Next, "next", -1, "")
		}
	}
	return states, edges
}

func evt1AsyncCFGGeneratedNames(cfg *evt1AsyncCFG) []string {
	if cfg == nil {
		return nil
	}
	out := make([]string, 0, len(cfg.Nodes))
	for _, state := range cfg.Nodes {
		out = append(out, state.Identity)
	}
	return out
}

func evt1AsyncPersistentMap(fn FunctionDecl, analysis evt1AsyncAnalysis, cfg *evt1AsyncCFG) map[string]Type {
	persistent := map[string]Type{}
	for _, p := range fn.Params {
		persistent[p.Name] = p.Type
	}
	for i := range analysis.Awaits {
		for _, name := range analysis.liveAcross(i) {
			if !strings.HasPrefix(name, "#") {
				persistent[name] = analysis.DeclTypes[name]
			}
		}
	}
	for _, stmt := range fn.Body.Statements {
		if v, ok := stmt.(*VarDecl); ok && evt1ExprContainsAwait(v.Value) {
			persistent[v.Name] = v.Type
		}
	}
	if cfg != nil {
		for name := range evt1AsyncCrossStateNames(cfg) {
			if t, declared := analysis.DeclTypes[name]; declared {
				persistent[name] = t
			}
		}
		for name, t := range cfg.MatchFields {
			persistent[name] = t
		}
		for _, each := range cfg.Foreaches {
			persistent[each.Statement.ItemName] = each.Statement.ItemType
			for _, sourceName := range evt1ExprNames(each.Statement.Source) {
				if t, declared := analysis.DeclTypes[sourceName]; declared {
					persistent[sourceName] = t
				}
			}
		}
	}
	return persistent
}

func evt1AsyncGraphNameSites(cfg *evt1AsyncCFG) (map[string]*evt1AsyncCFGNode, map[string]map[*evt1AsyncCFGNode]bool) {
	declaredAt := map[string]*evt1AsyncCFGNode{}
	usesAt := map[string]map[*evt1AsyncCFGNode]bool{}
	addExpr := func(node *evt1AsyncCFGNode, expr Expr) {
		for _, name := range evt1ExprNames(expr) {
			if usesAt[name] == nil {
				usesAt[name] = map[*evt1AsyncCFGNode]bool{}
			}
			usesAt[name][node] = true
		}
	}
	var addStatement func(*evt1AsyncCFGNode, Statement)
	addStatement = func(node *evt1AsyncCFGNode, stmt Statement) {
		switch s := stmt.(type) {
		case *VarDecl:
			declaredAt[s.Name] = node
			addExpr(node, s.Value)
		case *AssignStmt:
			addExpr(node, s.Target)
			addExpr(node, s.Value)
		case *ReturnStmt:
			addExpr(node, s.Value)
		case *ExprStmt:
			addExpr(node, s.Value)
		case *AssertStmt:
			addExpr(node, s.Condition)
			addExpr(node, s.Reason)
		case *IfStmt:
			addExpr(node, s.Condition)
			for _, child := range s.Then.Statements {
				addStatement(node, child)
			}
			if s.Else != nil {
				for _, child := range s.Else.Statements {
					addStatement(node, child)
				}
			}
		case *WhileStmt:
			addExpr(node, s.Condition)
			for _, child := range s.Body.Statements {
				addStatement(node, child)
			}
		case *ForeachStmt:
			addExpr(node, s.Source)
			for _, child := range s.Body.Statements {
				addStatement(node, child)
			}
		case *MatchStmt:
			addExpr(node, s.Subject)
			for _, arm := range s.Arms {
				for _, child := range arm.Block.Statements {
					addStatement(node, child)
				}
			}
		case *TryStmt:
			for _, child := range s.Body.Statements {
				addStatement(node, child)
			}
			for _, arm := range s.Except {
				for _, child := range arm.Body.Statements {
					addStatement(node, child)
				}
			}
		case *Block:
			for _, child := range s.Statements {
				addStatement(node, child)
			}
		}
	}
	for _, node := range cfg.Nodes {
		for _, stmt := range node.Statements {
			addStatement(node, stmt)
		}
		if node.Resume != nil {
			addStatement(node, node.Resume)
		}
		addExpr(node, node.Condition)
		if node.Await != nil {
			addExpr(node, node.Await.Value)
		}
		if node.Match != nil {
			addExpr(node, node.Match.Subject)
		}
		if node.Foreach != nil {
			addExpr(node, node.Foreach.Source)
		}
	}
	return declaredAt, usesAt
}

// evt1AsyncCrossStateNames finds ordinary locals whose declaration and use
// land in different generated states. This is graph liveness, rather than a
// blanket rule that promotes every local in a structured region.
func evt1AsyncCrossStateNames(cfg *evt1AsyncCFG) map[string]bool {
	declaredAt, usesAt := evt1AsyncGraphNameSites(cfg)
	out := map[string]bool{}
	for name, nodes := range usesAt {
		decl := declaredAt[name]
		if decl == nil {
			continue
		}
		for node := range nodes {
			if node != decl {
				out[name] = true
				break
			}
		}
	}
	return out
}

func evt1SortedPersistentNames(persistent map[string]Type) []string {
	names := make([]string, 0, len(persistent))
	for name := range persistent {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (l *lowering) asyncStateLowerer(fn FunctionDecl, persistent map[string]Type, stepName string) *evt1FunctionLowerer {
	f := newEVT1FunctionLowerer(l, fn, stepName, true)
	for name, t := range persistent {
		f.scope[0][name] = evt1Binding{cName: "frame->" + name, t: evt1CanonicalType(l.env, t)}
	}
	return f
}

func (l *lowering) lowerAsyncCFG(fn FunctionDecl, analysis evt1AsyncAnalysis, cfg *evt1AsyncCFG, persistent map[string]Type, frameName, stepName string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("static void %s(concept_async_operation* async_operation, void* storage) {\n  %s* frame = (%s*)storage;\nasync_dispatch:\n  switch (frame->state) {\n", stepName, frameName, frameName))
	for _, node := range cfg.Nodes {
		f := l.asyncStateLowerer(fn, persistent, stepName)
		b.WriteString(fmt.Sprintf("  case %d: { /* %s */\n", node.Index, node.Identity))
		switch node.Kind {
		case "BasicBlock":
			b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, 2))
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "AwaitSuspend":
			b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, 2))
			b.WriteString(l.lowerAsyncDirectCallPush(f, node.Await, node.Next.Index, 2))
		case "AfterAwait":
			if node.AwaitIndex >= 0 && node.AwaitIndex < len(analysis.Awaits) {
				await := analysis.Awaits[node.AwaitIndex]
				if await.ResultType.Name != "void" {
					b.WriteString(fmt.Sprintf("    if (async_operation->child_outcome_size != sizeof(frame->await_result_%d)) concept_async_abort(\"await result size mismatch\");\n    memcpy(&frame->await_result_%d, async_operation->child_outcome.bytes, sizeof(frame->await_result_%d));\n", node.AwaitIndex, node.AwaitIndex, node.AwaitIndex))
					f.scope[0][fmt.Sprintf("#await%d", node.AwaitIndex)] = evt1Binding{cName: fmt.Sprintf("frame->await_result_%d", node.AwaitIndex), t: await.ResultType}
				}
			}
			b.WriteString(l.lowerAsyncCFGResumedStatement(f, node, persistent, 2))
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "Branch":
			b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, 2))
			prelude, condition, _ := f.lowerExpr(node.Condition, 2)
			b.WriteString(prelude)
			b.WriteString(fmt.Sprintf("    frame->state = (%s) ? %d : %d;\n    goto async_dispatch;\n", condition, node.True.Index, node.False.Index))
		case "MatchDispatch":
			b.WriteString(l.lowerAsyncCFGMatch(f, node, persistent, 2))
		case "LoopInit":
			b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, 2))
			if node.Next != nil && node.Next.HasBound {
				b.WriteString(fmt.Sprintf("    frame->loop_iterations_%d = 0;\n", node.Next.Index))
			}
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "LoopHeader":
			if node.HasBound {
				b.WriteString(fmt.Sprintf("    if (frame->loop_iterations_%d >= %d) { frame->state = %d; goto async_dispatch; }\n", node.Index, node.LoopBound, node.False.Index))
			}
			prelude, condition, _ := f.lowerExpr(node.Condition, 2)
			b.WriteString(prelude)
			b.WriteString(fmt.Sprintf("    frame->state = (%s) ? %d : %d;\n    goto async_dispatch;\n", condition, node.True.Index, node.False.Index))
		case "LoopBackedge":
			if node.Next != nil && node.Next.HasBound {
				b.WriteString(fmt.Sprintf("    frame->loop_iterations_%d = frame->loop_iterations_%d + 1;\n", node.Next.Index, node.Next.Index))
			}
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "ForeachInit":
			b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, 2))
			b.WriteString(l.lowerAsyncCFGForeachInit(f, cfg.Foreaches[node.ForeachID], 2))
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "ForeachHeader":
			b.WriteString(l.lowerAsyncCFGForeachHeader(f, cfg.Foreaches[node.ForeachID], node.Body, node.Exit, 2))
		case "ForeachContinue":
			info := cfg.Foreaches[node.ForeachID]
			if info.Statement.SourceKind != "custom" {
				b.WriteString(fmt.Sprintf("    frame->%s = frame->%s + 1u;\n", info.IndexField, info.IndexField))
			}
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		case "Complete":
			if fn.ReturnType.Name == "void" {
				b.WriteString(l.lowerAsyncPersistentDrops(f, persistent, 2))
				b.WriteString("    concept_async_finish(async_operation, NULL, 0u);\n    return;\n")
			} else {
				b.WriteString("    concept_async_abort(\"value-bearing async function reached completion without a value\");\n    return;\n")
			}
		default:
			b.WriteString(l.asyncGotoNode(f, node.Next, 2))
		}
		b.WriteString("  }\n")
	}
	b.WriteString("  default: concept_async_abort(\"invalid generated async state\");\n  }\n}\n")
	return b.String()
}

func (l *lowering) lowerAsyncCFGResumedStatement(f *evt1FunctionLowerer, node *evt1AsyncCFGNode, persistent map[string]Type, indent int) string {
	failure := evt1AwaitFailureInStatement(node.Resume)
	if failure == nil || failure.Op != "?" || !evt1IsResultType(failure.ResolvedType) {
		return l.lowerAsyncResumedStatement(f, node.Resume, persistent, node.AwaitIndex, indent)
	}
	carrier := fmt.Sprintf("frame->await_result_%d", node.AwaitIndex)
	errorType := evt1FailureErrorType(failure.ResolvedType)
	var b strings.Builder
	b.WriteString(ind(indent) + fmt.Sprintf("if (%s.tag == 1) {\n", carrier))
	if handler, ok := node.Handlers[evt1TypeIdentity(errorType)]; ok {
		b.WriteString(ind(indent+1) + fmt.Sprintf("frame->%s = %s.payload.error.error;\n", handler.Binding, carrier))
		b.WriteString(ind(indent+1) + fmt.Sprintf("frame->state = %d; goto async_dispatch;\n", handler.Target.Index))
	} else if evt1IsResultType(f.fn.ReturnType) {
		temp := f.nextTemp("async_error")
		b.WriteString(ind(indent+1) + fmt.Sprintf("%s %s = %s(%s.payload.error.error);\n", evt1CType(f.fn.ReturnType), temp, evt1FailureConstructorName(f.fn.ReturnType, "Error"), carrier))
		b.WriteString(l.lowerAsyncPersistentDrops(f, persistent, indent+1))
		b.WriteString(ind(indent+1) + fmt.Sprintf("concept_async_finish(async_operation, &%s, sizeof(%s));\n", temp, temp))
		b.WriteString(ind(indent+1) + "return;\n")
	} else {
		b.WriteString(ind(indent+1) + "concept_async_abort(\"unhandled async Result error\");\n")
	}
	b.WriteString(ind(indent) + "}\n")
	successType := evt1FailureSuccessType(failure.ResolvedType)
	if successType.Name == "void" {
		return b.String()
	}
	successName := fmt.Sprintf("#await_success%d", node.AwaitIndex)
	f.scope[0][successName] = evt1Binding{cName: carrier + ".payload.ok.value", t: successType}
	b.WriteString(l.lowerAsyncOrdinaryStatement(f, evt1ReplaceAwaitFailure(node.Resume, failure, successName), persistent, indent))
	return b.String()
}

func evt1AwaitFailureInStatement(stmt Statement) *FailureExpr {
	var find func(Expr) *FailureExpr
	find = func(expr Expr) *FailureExpr {
		switch e := expr.(type) {
		case *FailureExpr:
			if name, ok := e.Value.(*NameExpr); ok && strings.HasPrefix(name.Name, "#await") {
				return e
			}
			return find(e.Value)
		case *ParenExpr:
			return find(e.Value)
		case *UnaryExpr:
			return find(e.Value)
		case *BinaryExpr:
			if failure := find(e.Left); failure != nil {
				return failure
			}
			return find(e.Right)
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

func evt1ReplaceAwaitFailure(stmt Statement, target *FailureExpr, name string) Statement {
	var replace func(Expr) Expr
	replace = func(expr Expr) Expr {
		if expr == target {
			return &NameExpr{Name: name, Span: target.Span}
		}
		switch e := expr.(type) {
		case *ParenExpr:
			copy := *e
			copy.Value = replace(e.Value)
			return &copy
		case *UnaryExpr:
			copy := *e
			copy.Value = replace(e.Value)
			return &copy
		case *BinaryExpr:
			copy := *e
			copy.Left, copy.Right = replace(e.Left), replace(e.Right)
			return &copy
		}
		return expr
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

func (l *lowering) lowerAsyncCFGStatements(f *evt1FunctionLowerer, statements []Statement, persistent map[string]Type, indent int) string {
	var b strings.Builder
	for _, stmt := range statements {
		b.WriteString(l.lowerAsyncOrdinaryStatement(f, stmt, persistent, indent))
	}
	return b.String()
}

func (l *lowering) asyncGotoNode(f *evt1FunctionLowerer, next *evt1AsyncCFGNode, indent int) string {
	if next == nil {
		return ind(indent) + "concept_async_abort(\"async graph has no successor\");\n" + ind(indent) + "return;\n"
	}
	return f.lowerAllScopeDrops(indent) + ind(indent) + fmt.Sprintf("frame->state = %d;\n", next.Index) + ind(indent) + "goto async_dispatch;\n"
}

func (l *lowering) lowerAsyncCFGMatch(f *evt1FunctionLowerer, node *evt1AsyncCFGNode, persistent map[string]Type, indent int) string {
	var b strings.Builder
	b.WriteString(l.lowerAsyncCFGStatements(f, node.Statements, persistent, indent))
	prelude, subject, subjectType := f.lowerExpr(node.Match.Subject, indent)
	enumDecl := l.env.enums[subjectType.Name]
	if decl, ok := evt1FailureEnumDecl(subjectType); ok {
		enumDecl = decl
	}
	temp := f.nextTemp("async_match_subject")
	b.WriteString(prelude)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s; /* scrutinee evaluated once */\n", evt1CType(subjectType), temp, subject))
	b.WriteString(ind(indent) + fmt.Sprintf("switch (%s.tag) {\n", temp))
	for _, arm := range node.MatchArms {
		variant, _ := evt1LookupVariant(enumDecl, arm.Pattern.VariantName)
		tag := evt1TagName(enumDecl.Name, variant.Name)
		if evt1IsFailureType(subjectType) {
			tag = fmt.Sprintf("%d", variant.Tag)
		}
		b.WriteString(ind(indent) + fmt.Sprintf("case %s:\n", tag))
		for i, binding := range arm.Pattern.Bindings {
			if _, stored := persistent[binding]; !stored {
				continue
			}
			field := variant.Payload[i]
			b.WriteString(ind(indent+1) + fmt.Sprintf("frame->%s = %s.payload.%s.%s;\n", binding, temp, evt1PayloadFieldName(variant.Name), field.Name))
		}
		b.WriteString(ind(indent+1) + fmt.Sprintf("frame->state = %d; goto async_dispatch;\n", arm.Target.Index))
	}
	b.WriteString(ind(indent) + "default:\n")
	b.WriteString(ind(indent+1) + fmt.Sprintf("concept_abort_invalid_tag(\"%s\");\n", enumDecl.Name))
	b.WriteString(ind(indent) + "}\n    return;\n")
	return b.String()
}

func (l *lowering) lowerAsyncCFGForeachInit(f *evt1FunctionLowerer, info evt1AsyncCFGForeach, indent int) string {
	stmt := info.Statement
	prelude, source, sourceType := f.lowerExpr(stmt.Source, indent)
	var b strings.Builder
	b.WriteString(prelude)
	if info.SourceByRef && sourceType.ArrayElem != nil {
		b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = &(%s); /* source evaluated once; no backing copy */\n", info.SourceField, source))
	} else {
		b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = %s; /* source evaluated once */\n", info.SourceField, source))
	}
	b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = 0u;\n", info.IndexField))
	if stmt.SourceKind == "custom" {
		get := evt1ForeachProtocolFunction(l.env, "GetIterator", sourceType)
		arg := "frame->" + info.SourceField
		if get.Params[0].Type.isBorrowLike() {
			arg = "&(" + arg + ")"
		}
		b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = %s(%s);\n", info.Iterator, evt1FunctionSymbolForDecl(l.outputBase, l.env, get), arg))
	}
	return b.String()
}

func (l *lowering) lowerAsyncCFGForeachHeader(f *evt1FunctionLowerer, info evt1AsyncCFGForeach, body, exit *evt1AsyncCFGNode, indent int) string {
	stmt := info.Statement
	var b strings.Builder
	if stmt.SourceKind == "custom" {
		move := evt1ForeachProtocolFunction(l.env, "MoveNext", stmt.IteratorType)
		current := evt1ForeachProtocolFunction(l.env, "Current", stmt.IteratorType)
		iterator := "frame->" + info.Iterator
		moveArg, currentArg := iterator, iterator
		if move.Params[0].Type.isBorrowLike() {
			moveArg = "&(" + iterator + ")"
		}
		if current.Params[0].Type.isBorrowLike() {
			currentArg = "&(" + iterator + ")"
		}
		b.WriteString(ind(indent) + fmt.Sprintf("if (!(%s(%s))) { frame->state = %d; goto async_dispatch; }\n", evt1FunctionSymbolForDecl(l.outputBase, l.env, move), moveArg, exit.Index))
		b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = %s(%s); /* Current once per successful iteration */\n", stmt.ItemName, evt1FunctionSymbolForDecl(l.outputBase, l.env, current), currentArg))
	} else {
		source := "frame->" + info.SourceField
		if info.SourceByRef && stmt.SourceType.ArrayElem != nil {
			source = "(*" + source + ")"
		}
		length := fmt.Sprintf("%d", evt1StorageElementCount(stmt.SourceType))
		if evt1IsSpanType(stmt.SourceType) {
			length = source + ".length"
		}
		b.WriteString(ind(indent) + fmt.Sprintf("if (frame->%s >= (size_t)(%s)) { frame->state = %d; goto async_dispatch; }\n", info.IndexField, length, exit.Index))
		item := fmt.Sprintf("%s.data[frame->%s]", source, info.IndexField)
		if stmt.ItemType.Ownership == "ref" {
			item = "&(" + item + ")"
		}
		b.WriteString(ind(indent) + fmt.Sprintf("frame->%s = %s;\n", stmt.ItemName, item))
	}
	b.WriteString(ind(indent) + fmt.Sprintf("frame->state = %d; goto async_dispatch;\n", body.Index))
	return b.String()
}

func evt1ValidateAsyncGraph(async *MIRAsyncFunction, span Span) error {
	if async == nil {
		return nil
	}
	if async.ControlFlowStrategy != "StructuredStateGraph" || len(async.States) == 0 || len(async.GeneratedStates) != len(async.States) {
		return evt1Diagnostic("ASYNC_GRAPH_INVALID", "async normalization omitted its structured generated-state graph", span)
	}
	states := map[string]bool{}
	kinds := map[string]string{}
	for i, state := range async.States {
		if state.Identity == "" || state.Kind == "" || states[state.Identity] || async.GeneratedStates[i] != state.Identity {
			return evt1Diagnostic("ASYNC_GENERATED_STATE_INVALID", "async state identity, kind, order, or uniqueness is invalid", span)
		}
		states[state.Identity] = true
		kinds[state.Identity] = state.Kind
	}
	reachable := map[string]bool{async.States[0].Identity: true}
	outgoing := map[string][]MIRAsyncEdge{}
	awaitEdges := map[int]int{}
	for _, edge := range async.Edges {
		if !states[edge.From] || !states[edge.To] || edge.Kind == "" {
			return evt1Diagnostic("ASYNC_GRAPH_INVALID", "async graph edge targets an unknown generated state", span)
		}
		outgoing[edge.From] = append(outgoing[edge.From], edge)
		if edge.Kind == "await_resume" {
			if edge.AwaitIndex < 0 || edge.AwaitIndex >= len(async.AwaitPoints) || kinds[edge.From] != "AwaitSuspend" || kinds[edge.To] != "AfterAwait" {
				return evt1Diagnostic("ASYNC_GRAPH_INVALID", "async await edge has an invalid index or continuation shape", span)
			}
			awaitEdges[edge.AwaitIndex]++
		}
	}
	for changed := true; changed; {
		changed = false
		for _, edge := range async.Edges {
			if reachable[edge.From] && !reachable[edge.To] {
				reachable[edge.To], changed = true, true
			}
		}
	}
	for _, state := range async.States {
		if !reachable[state.Identity] {
			return evt1Diagnostic("ASYNC_GRAPH_INVALID", "async graph contains an unreachable nonterminal state", span)
		}
	}
	for _, state := range async.States {
		count := len(outgoing[state.Identity])
		switch state.Kind {
		case "Complete":
			if count != 0 {
				return evt1Diagnostic("ASYNC_GRAPH_INVALID", "async completion state must be terminal", span)
			}
		case "Branch", "LoopHeader", "ForeachHeader":
			if count != 2 {
				return evt1Diagnostic("ASYNC_JOIN_INVALID", "async branch or loop header must have two explicit targets", span)
			}
		case "MatchDispatch":
			if count == 0 {
				return evt1Diagnostic("ASYNC_MATCH_NORMALIZATION_INVALID", "async match dispatch has no arm target", span)
			}
		default:
			if count == 0 {
				return evt1Diagnostic("ASYNC_GENERATED_STATE_INVALID", "nonterminal async state has no successor", span)
			}
		}
	}
	for _, await := range async.AwaitPoints {
		if !states[await.Continuation] || kinds[await.Continuation] != "AfterAwait" || awaitEdges[await.Index] != 1 {
			return evt1Diagnostic("ASYNC_GRAPH_INVALID", "await continuation is absent from the generated graph", await.SourceSpan)
		}
	}
	return nil
}
