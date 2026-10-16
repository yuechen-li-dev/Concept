package concept

import (
	"math"
)

// LocalBoundsProof is an operation-local proof over checked source values.
// It has no target representation, and is never imported as authority. A view
// length is a value, not a claim that its backing memory is independent.
type LocalBoundsProof struct {
	IndexMin  int64  `json:"index_min"`
	IndexMax  int64  `json:"index_max"`
	LengthMin int64  `json:"length_min"`
	Origin    string `json:"origin"`
}

type localInterval struct{ lo, hi int64 }

var localInt32 = localInterval{math.MinInt32, math.MaxInt32}

type localValue struct {
	t          Type
	integer    localInterval
	intKnown   bool
	lengthMin  int64
	lengthHeld bool
}

type localBoundsState map[string]localValue

func (s localBoundsState) clone() localBoundsState {
	out := localBoundsState{}
	for k, v := range s {
		out[k] = v
	}
	return out
}

type localBoundsAnalysis struct {
	// A scalar/view whose address occurs anywhere in the function is not
	// tracked. This also handles a later opaque call through an escaped ref.
	exposed   map[string]bool
	env       *semanticEnv
	calls     *localBoundsCalls
	summaries map[string]bool
}

// evt1ProveLocalBounds runs on validated, closed typed bodies, before MIR
// collection or backend selection. Unknown constructs never grant a proof.
func evt1ProveLocalBounds(module Module, env *semanticEnv) {
	inferLocalBounds(module, env)
}

func proveFunctionLocalBounds(fn FunctionDecl, env *semanticEnv, inputs []int64, calls *localBoundsCalls) {
	if fn.Body == nil || fn.Comptime || fn.Async {
		return
	}
	a := localBoundsAnalysis{exposed: map[string]bool{}, env: env, calls: calls}
	evt1WalkTypedBlock(*fn.Body, func(expr Expr) {
		if index, ok := expr.(*IndexExpr); ok {
			index.LocalBounds = nil
			index.BoundsOperationID = ""
		}
		if ref, ok := expr.(*RefExpr); ok {
			if name, ok := ref.Value.(*NameExpr); ok {
				a.exposed[name.Name] = true
			}
		}
		if closure, ok := expr.(*CallableExpr); ok {
			// Captures can retain a mutable reference without RefExpr syntax.
			for _, capture := range closure.Captures {
				evt1WalkTypedExpr(capture.Source, func(e Expr) {
					if name, ok := e.(*NameExpr); ok {
						a.exposed[name.Name] = true
					}
				})
			}
		}
	})
	state := localBoundsState{}
	for i, p := range fn.Params {
		v := localValue{t: p.Type}
		if p.Type.Name == "int" && !p.Type.isReference() {
			v.integer, v.intKnown = localInt32, true
		}
		if evt1IsSpanType(p.Type) && !p.Type.isReference() {
			v.lengthHeld = true
			if i < len(inputs) {
				v.lengthMin = inputs[i]
			}
		}
		state[p.Name] = v
	}
	a.block(*fn.Body, state)
}

// value is deliberately read-only: condition refinement must not revisit an
// already executed access using its own successful-check postcondition.
func (a *localBoundsAnalysis) value(e Expr, s localBoundsState) localValue {
	switch e := e.(type) {
	case *IntLiteral:
		if e.Magnitude <= math.MaxInt32 {
			n := int64(e.Magnitude)
			if e.Negative {
				n = -n
			}
			return localInteger(n)
		}
	case *ComptimeValueExpr:
		if e.Value.Kind == ValueInt && !e.Value.WideUint && e.Value.IntValue >= math.MinInt32 && e.Value.IntValue <= math.MaxInt32 {
			return localInteger(int64(e.Value.IntValue))
		}
	case *NameExpr:
		return a.get(s, e.Name)
	case *ParenExpr:
		return a.value(e.Value, s)
	case *CastExpr:
		v := a.value(e.Value, s)
		if e.Target.Name == "int" && v.intKnown {
			return v
		}
	case *UnaryExpr:
		if e.Op == "-" {
			return localArithmetic("-", localInteger(0), a.value(e.Value, s))
		}
	case *BinaryExpr:
		if e.ResolvedType.Name == "int" {
			return localArithmetic(e.Op, a.value(e.Left, s), a.value(e.Right, s))
		}
	case *CallExpr:
		args := make([]localValue, len(e.Args))
		for i, arg := range e.Args {
			args[i] = a.value(arg, s)
		}
		return a.constantCall(e, args)
	case *IfExpr:
		if condition, known := a.constantBool(e.Condition, s); known {
			if condition {
				return a.value(e.Then, s)
			}
			return a.value(e.Else, s)
		}
	}
	return localValue{}
}

func localJoin(a, b localBoundsState) localBoundsState {
	out := a.clone()
	for name, x := range out {
		y, ok := b[name]
		if !ok {
			delete(out, name)
			continue
		}
		x.intKnown = x.intKnown && y.intKnown
		if x.intKnown {
			x.integer = localInterval{min(x.integer.lo, y.integer.lo), max(x.integer.hi, y.integer.hi)}
		}
		x.lengthHeld = x.lengthHeld && y.lengthHeld
		x.lengthMin = min(x.lengthMin, y.lengthMin)
		out[name] = x
	}
	return out
}

func (a *localBoundsAnalysis) get(s localBoundsState, name string) localValue {
	v := s[name]
	if a.exposed[name] {
		v.intKnown, v.lengthHeld = false, false
	}
	return v
}

func localInteger(v int64) localValue {
	return localValue{integer: localInterval{v, v}, intKnown: true}
}

func localArithmetic(op string, x, y localValue) localValue {
	if !x.intKnown || !y.intKnown {
		return localValue{}
	}
	// Keep the domain bounded to int32. Products of two int32 endpoints fit
	// int64; no host overflow is used to establish a source proof.
	l, r := x.integer, y.integer
	if l.lo < math.MinInt32 || l.hi > math.MaxInt32 || r.lo < math.MinInt32 || r.hi > math.MaxInt32 {
		return localValue{}
	}
	var interval localInterval
	switch op {
	case "+":
		interval = localInterval{l.lo + r.lo, l.hi + r.hi}
	case "-":
		interval = localInterval{l.lo - r.hi, l.hi - r.lo}
	case "*":
		products := []int64{l.lo * r.lo, l.lo * r.hi, l.hi * r.lo, l.hi * r.hi}
		interval = localInterval{products[0], products[0]}
		for _, p := range products[1:] {
			interval.lo, interval.hi = min(interval.lo, p), max(interval.hi, p)
		}
	default:
		return localValue{}
	}
	if interval.lo < math.MinInt32 || interval.hi > math.MaxInt32 {
		return localValue{}
	}
	return localValue{integer: interval, intKnown: true}
}

func (a *localBoundsAnalysis) expr(e Expr, s localBoundsState) localValue {
	switch e := e.(type) {
	case *IntLiteral:
		if e.Magnitude <= math.MaxInt32 {
			n := int64(e.Magnitude)
			if e.Negative {
				n = -n
			}
			return localInteger(n)
		}
	case *ComptimeValueExpr:
		if e.Value.Kind == ValueInt && !e.Value.WideUint && e.Value.IntValue >= math.MinInt32 && e.Value.IntValue <= math.MaxInt32 {
			return localInteger(int64(e.Value.IntValue))
		}
	case *NameExpr:
		return a.get(s, e.Name)
	case *ParenExpr:
		return a.expr(e.Value, s)
	case *CastExpr:
		v := a.expr(e.Value, s)
		if e.Target.Name == "int" && v.intKnown && v.integer.lo >= math.MinInt32 && v.integer.hi <= math.MaxInt32 {
			return v
		}
	case *UnaryExpr:
		v := a.expr(e.Value, s)
		if e.Op == "-" {
			return localArithmetic("-", localInteger(0), v)
		}
	case *BinaryExpr:
		left := a.expr(e.Left, s)
		if e.Op == "and" || e.Op == "or" {
			// The right operand may not run. Its successful accesses cannot
			// establish a postcondition outside the short-circuit expression.
			rightState := s.clone()
			a.assume(e.Left, e.Op == "and", rightState)
			a.expr(e.Right, rightState)
			joined := localJoin(s, rightState)
			for k, v := range joined {
				s[k] = v
			}
			return localValue{}
		}
		right := a.expr(e.Right, s)
		// Float arithmetic is never folded/reassociated by this analysis.
		if e.ResolvedType.Name == "int" {
			return localArithmetic(e.Op, left, right)
		}
	case *IndexExpr:
		base := a.expr(e.Base, s)
		indices := evt1StorageIndices(e)
		if len(indices) != 1 {
			for _, i := range indices {
				a.expr(i, s)
			}
			return localValue{}
		}
		i := a.expr(indices[0], s)
		if name, ok := e.Base.(*NameExpr); ok && e.SpanIndex && base.lengthHeld && i.intKnown && i.integer.lo >= 0 {
			if i.integer.hi < base.lengthMin {
				e.LocalBounds = &LocalBoundsProof{i.integer.lo, i.integer.hi, base.lengthMin, "LocalCheckedControlFlow"}
			}
			// A successful index proves length > the *lower* endpoint, not
			// the upper endpoint of a runtime-dependent interval.
			base.lengthMin = max(base.lengthMin, i.integer.lo+1)
			s[name.Name] = base
		}
	case *CallExpr:
		args := make([]localValue, len(e.Args))
		for j, arg := range e.Args {
			args[j] = a.expr(arg, s)
		}
		if e.Intrinsic == "span_subregion" && len(args) == 3 && args[2].intKnown {
			return localValue{lengthMin: max(0, args[2].integer.lo), lengthHeld: true}
		}
		if e.SpanElementType != nil {
			if len(args) == 1 && args[0].lengthHeld {
				return localValue{lengthMin: args[0].lengthMin, lengthHeld: true}
			}
			if len(args) == 1 && args[0].t.ArrayElem != nil && !evt1StorageHasRuntimeShape(args[0].t) {
				return localValue{lengthMin: int64(evt1StorageElementCount(args[0].t)), lengthHeld: true}
			}
		}
		a.calls.record(e, args)
		return a.constantCall(e, args)
	case *TemplateCallExpr:
		args := make([]localValue, len(e.Args))
		for i, arg := range e.Args {
			args[i] = a.expr(arg, s)
		}
		a.calls.record(e, args)
	case *IfExpr:
		a.expr(e.Condition, s)
		x, y := s.clone(), s.clone()
		a.assume(e.Condition, true, x)
		a.assume(e.Condition, false, y)
		l, r := a.expr(e.Then, x), a.expr(e.Else, y)
		joined := localJoin(x, y)
		for k, v := range joined {
			s[k] = v
		}
		if l.intKnown && r.intKnown {
			return localValue{integer: localInterval{min(l.integer.lo, r.integer.lo), max(l.integer.hi, r.integer.hi)}, intKnown: true}
		}
	default:
		// Visit ordinary eager expression children, but never execute a
		// closure body as if construction invoked it.
		if e != nil {
			switch e := e.(type) {
			case *FieldExpr:
				a.expr(e.Receiver, s)
			case *StructConstructExpr:
				for _, arg := range e.Args {
					a.expr(arg, s)
				}
			case *ArrayLiteralExpr:
				for _, arg := range e.Elements {
					a.expr(arg, s)
				}
			case *ConstructExpr:
				for _, arg := range e.Args {
					a.expr(arg, s)
				}
			}
		}
	}
	return localValue{}
}

func (a *localBoundsAnalysis) assume(e Expr, truth bool, s localBoundsState) {
	if e, ok := e.(*ParenExpr); ok {
		a.assume(e.Value, truth, s)
		return
	}
	if e, ok := e.(*UnaryExpr); ok && e.Op == "not" {
		a.assume(e.Value, !truth, s)
		return
	}
	b, ok := e.(*BinaryExpr)
	if !ok {
		return
	}
	if b.Op == "and" && truth || b.Op == "or" && !truth {
		a.assume(b.Left, truth, s)
		a.assume(b.Right, truth, s)
		return
	}
	// Only side-effect-free constant bounds authorize refinements. The
	// state used to evaluate a bound is discarded (it grants no postfacts).
	right := a.value(b.Right, s)
	if !right.intKnown || right.integer.lo != right.integer.hi {
		return
	}
	n := right.integer.lo
	op := b.Op
	if !truth {
		op = map[string]string{"<": ">=", "<=": ">", ">": "<=", ">=": "<", "==": "!=", "!=": "=="}[op]
	}
	if query, ok := b.Left.(*CallExpr); ok && query.Callee == "Len" && len(query.Args) == 1 && !query.Member && evt1ImplicitValueIntrinsic(a.env, query) {
		if name, ok := query.Args[0].(*NameExpr); ok {
			v := a.get(s, name.Name)
			if v.lengthHeld && (op == ">=" || op == ">" || op == "==") {
				if op == ">" {
					n++
				}
				v.lengthMin = max(v.lengthMin, n)
				s[name.Name] = v
			}
		}
	}
	if name, ok := b.Left.(*NameExpr); ok {
		v := a.get(s, name.Name)
		if !v.intKnown {
			return
		}
		switch op {
		case ">=":
			v.integer.lo = max(v.integer.lo, n)
		case ">":
			v.integer.lo = max(v.integer.lo, n+1)
		case "<=":
			v.integer.hi = min(v.integer.hi, n)
		case "<":
			v.integer.hi = min(v.integer.hi, n-1)
		case "==":
			v.integer.lo, v.integer.hi = max(v.integer.lo, n), min(v.integer.hi, n)
		}
		if v.integer.lo > v.integer.hi {
			v.intKnown = false
		}
		s[name.Name] = v
	}
}

func localForgetWrites(b Block, s localBoundsState) {
	// Nested blocks are scanned too. False invalidation is conservative.
	for _, stmt := range b.Statements {
		switch v := stmt.(type) {
		case *AssignStmt:
			if n, ok := v.Target.(*NameExpr); ok {
				x := s[n.Name]
				x.intKnown, x.lengthHeld = false, false
				s[n.Name] = x
			}
		case *IfStmt:
			localForgetWrites(v.Then, s)
			if v.Else != nil {
				localForgetWrites(*v.Else, s)
			}
		case *ForeachStmt:
			localForgetWrites(v.Body, s)
		case *WhileStmt:
			localForgetWrites(v.Body, s)
		case *Block:
			localForgetWrites(*v, s)
		case *VarDecl, *ReturnStmt, *ExprStmt, *AssertStmt, *StaticAssertStmt:
		default:
			for name, v := range s {
				v.intKnown, v.lengthHeld = false, false
				s[name] = v
			}
		}
	}
}

// block returns the surviving continuation; nil denotes a terminal path.
func (a *localBoundsAnalysis) block(b Block, s localBoundsState) localBoundsState {
	shadowed := map[string]localValue{}
	declared := map[string]bool{}
	for _, stmt := range b.Statements {
		switch v := stmt.(type) {
		case *VarDecl:
			x := a.expr(v.Value, s)
			if v.Type.Name != "int" && !v.Comptime {
				x.intKnown = false // narrow/unsigned stores may wrap
			}
			if old, ok := s[v.Name]; ok {
				shadowed[v.Name] = old
			}
			declared[v.Name] = true
			x.t = v.Type
			s[v.Name] = x
		case *AssignStmt:
			a.expr(v.Target, s)
			x := a.expr(v.Value, s)
			if n, ok := v.Target.(*NameExpr); ok {
				old := a.get(s, n.Name)
				if v.CompoundOp != "" {
					x = localArithmetic(v.CompoundOp, old, x)
				}
				x.t = old.t
				if old.t.Name != "int" {
					x.intKnown = false
				}
				s[n.Name] = x
			}
		case *ExprStmt:
			a.expr(v.Value, s)
		case *ReturnStmt:
			a.expr(v.Value, s)
			return nil
		case *IfStmt:
			a.expr(v.Condition, s)
			x, y := s.clone(), s.clone()
			a.assume(v.Condition, true, x)
			a.assume(v.Condition, false, y)
			x = a.block(v.Then, x)
			if v.Else != nil {
				y = a.block(*v.Else, y)
			}
			if x == nil && y == nil {
				return nil
			}
			if x == nil {
				s = y
			} else if y == nil {
				s = x
			} else {
				s = localJoin(x, y)
			}
		case *ForeachStmt:
			// Only the checked ascending unit-step range has this interval.
			rangeExpr, ok := v.Source.(*BinaryExpr)
			var lo, hi localValue
			if ok && rangeExpr.Op == ".." {
				lo, hi = a.expr(rangeExpr.Left, s), a.expr(rangeExpr.Right, s)
			} else {
				a.expr(v.Source, s)
			}
			localForgetWrites(v.Body, s)
			child := s.clone()
			item := localValue{t: v.ElementType}
			if ok && rangeExpr.Op == ".." && lo.intKnown && hi.intKnown && lo.integer.lo < hi.integer.hi {
				item.integer, item.intKnown = localInterval{lo.integer.lo, hi.integer.hi - 1}, true
			}
			child[v.ItemName] = item
			a.block(v.Body, child)
		case *WhileStmt:
			localForgetWrites(v.Body, s)
			a.expr(v.Condition, s)
			child := s.clone()
			a.assume(v.Condition, true, child)
			a.block(v.Body, child)
			if v.Else != nil {
				localForgetWrites(*v.Else, s)
				a.block(*v.Else, s.clone())
			}
		case *Block:
			s = a.block(*v, s)
			if s == nil {
				return nil
			}
		case *StaticAssertStmt:
			// It is not an optimizer permission.
		case *AssertStmt:
			a.expr(v.Condition, s)
		default:
			// No speculative facts cross an unmodeled statement.
			for name, x := range s {
				x.intKnown, x.lengthHeld = false, false
				s[name] = x
			}
		}
	}
	for name := range declared {
		if old, ok := shadowed[name]; ok {
			s[name] = old
		} else {
			delete(s, name)
		}
	}
	return s
}
