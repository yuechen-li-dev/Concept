package concept

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// AccessKind is deliberately closed. New spellings require a concrete
// synchronization proposition, not a library or method-name convention.
type AccessKind string

const (
	AccessRead        AccessKind = "Read"
	AccessWrite       AccessKind = "Write"
	AccessAtomicRead  AccessKind = "AtomicRead"
	AccessAtomicWrite AccessKind = "AtomicWrite"
	AccessPublish     AccessKind = "Publish"
	AccessConsume     AccessKind = "Consume"
	AccessClaim       AccessKind = "Claim"
	AccessCommit      AccessKind = "Commit"
)

type AccessResolution string

const (
	AccessExact    AccessResolution = "Exact"
	AccessInterval AccessResolution = "Interval"
	AccessOpaque   AccessResolution = "Opaque"
)

type AccessIdentity struct {
	Kind     string `json:"kind"`
	Module   string `json:"module,omitempty"`
	Function string `json:"function,omitempty"`
	Ordinal  int    `json:"ordinal,omitempty"`
	Name     string `json:"name,omitempty"` // display spelling, not sole identity
	Type     Type   `json:"type,omitempty"`
}

type AccessPathElement struct {
	Kind    string `json:"kind"`
	Field   string `json:"field,omitempty"`
	Ordinal int    `json:"ordinal,omitempty"`
	Index   int64  `json:"index,omitempty"`
}

type AccessSubject struct {
	Root       AccessIdentity      `json:"root"`
	Path       []AccessPathElement `json:"path,omitempty"`
	Type       Type                `json:"type,omitempty"`
	RegionID   string              `json:"region_id,omitempty"`
	Offset     int                 `json:"offset,omitempty"`
	Extent     int                 `json:"extent,omitempty"`
	Resolution AccessResolution    `json:"resolution"`
}

type MIRAccessEntry struct {
	ID               string             `json:"id"`
	OpenGenericOwner string             `json:"open_generic_owner,omitempty"`
	Instance         string             `json:"instance,omitempty"`
	Subject          AccessSubject      `json:"subject"`
	Operation        AccessKind         `json:"operation"`
	Context          AccessIdentity     `json:"context"`
	Function         AccessIdentity     `json:"function"`
	Module           AccessIdentity     `json:"module"`
	Origin           SemanticFactOrigin `json:"origin"`
	Resolution       AccessResolution   `json:"resolution"`
	MemoryOrder      string             `json:"memory_order,omitempty"`
	Mechanism        string             `json:"mechanism,omitempty"`
	Sequence         int                `json:"sequence,omitempty"`
	SourceSpan       Span               `json:"source_span"`
}

type evt1AccessCall struct {
	callee   string
	args     []Expr
	span     Span
	sequence int
}

type evt1AccessFunction struct {
	decl              FunctionDecl
	key               string
	openGenericParams map[string]bool
	locals            map[string]AccessSubject
	entries           []MIRAccessEntry
	calls             []evt1AccessCall
	sequence          int
}

func evt1SemanticAccessAttribute(name string) bool {
	return name == "semantic_access" || name == "execution_context" || name == "synchronization"
}

func evt1AccessSummaryDemanded(module Module) bool {
	if len(module.AccessSummaries) != 0 {
		return true
	}
	for _, assertion := range module.Assertions {
		if evt1IsSharedAccessAnalysis(assertion.ConceptName) {
			return true
		}
	}
	for _, fn := range module.Functions {
		for _, attribute := range fn.Attributes {
			if evt1SemanticAccessAttribute(attribute.Name) {
				return true
			}
		}
	}
	for _, generic := range module.GenericTypes {
		for _, method := range generic.Struct.Methods {
			for _, attribute := range method.Attributes {
				if evt1SemanticAccessAttribute(attribute.Name) {
					return true
				}
			}
		}
	}
	return false
}

func evt1AccessFunctionKey(fn FunctionDecl) string {
	return fn.Module + "::" + fn.Name + "(" + evt1FunctionParamSignature(fn) + ")"
}

func evt1AccessIdentityKey(id AccessIdentity) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s", id.Kind, id.Module, id.Function, id.Ordinal, id.Type.String())
}

func evt1AccessSubjectKey(subject AccessSubject) string {
	var path strings.Builder
	for _, part := range subject.Path {
		fmt.Fprintf(&path, "/%s:%s:%d:%d", part.Kind, part.Field, part.Ordinal, part.Index)
	}
	return evt1AccessIdentityKey(subject.Root) + path.String() + "|" + subject.Type.String() + "|" + subject.RegionID + fmt.Sprintf("|%d|%d|%s", subject.Offset, subject.Extent, subject.Resolution)
}

func evt1AccessEntryKey(entry MIRAccessEntry) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%d:%d", entry.OpenGenericOwner, entry.Instance, entry.Operation, evt1AccessSubjectKey(entry.Subject), evt1AccessIdentityKey(entry.Context), evt1AccessIdentityKey(entry.Function), entry.Origin, entry.MemoryOrder, entry.Mechanism, entry.SourceSpan.Line, entry.SourceSpan.Column)
}

func evt1FinalizeAccessEntry(entry MIRAccessEntry) MIRAccessEntry {
	entry.Resolution = entry.Subject.Resolution
	if entry.Origin == "" {
		entry.Origin = FactOriginDerivedAccessSummary
	}
	sum := sha256.Sum256([]byte(evt1AccessEntryKey(entry)))
	entry.ID = "access:" + hex.EncodeToString(sum[:12])
	return entry
}

func evt1DeriveAccessSummaries(env *semanticEnv, module Module) error {
	// Imported summaries are authoritative artifact data. Their payload bodies
	// are intentionally not re-analysed.
	for _, imported := range module.AccessSummaries {
		entry := imported
		entry.Origin = FactOriginModuleAccessSummary
		entry = evt1FinalizeAccessEntry(entry)
		env.accessSummaries = append(env.accessSummaries, entry)
	}

	functions := map[string]*evt1AccessFunction{}
	byName := map[string][]*evt1AccessFunction{}
	for _, fn := range module.Functions {
		if fn.Module != module.Name {
			continue
		}
		state := &evt1AccessFunction{decl: fn, key: evt1AccessFunctionKey(fn), locals: map[string]AccessSubject{}}
		for i, param := range fn.Params {
			state.locals[param.Name] = AccessSubject{
				Root: AccessIdentity{Kind: "Parameter", Module: fn.Module, Function: state.key, Ordinal: i, Name: param.Name, Type: param.Type},
				Type: param.Type, Resolution: AccessExact,
			}
		}
		if err := evt1AccessApplyAttributes(env, state); err != nil {
			return err
		}
		if fn.Body != nil {
			evt1AccessBlock(env, state, *fn.Body)
			evt1AccessAttachSynchronizationOrders(state)
		} else if fn.ExternABI != "" && len(state.entries) == 0 {
			for i, param := range fn.Params {
				if !param.Type.isBorrowLike() || param.Type.Const {
					continue
				}
				subject := state.locals[param.Name]
				subject.Resolution = AccessOpaque
				entry := evt1AccessEntry(state, AccessWrite, subject, fn.Span)
				entry.Origin = FactOriginDeclaredForeign
				entry.Subject.Root.Ordinal = i
				state.entries = append(state.entries, evt1FinalizeAccessEntry(entry))
			}
		}
		functions[state.key] = state
		byName[fn.Name] = append(byName[fn.Name], state)
	}
	for _, template := range module.Templates {
		if template.Body == nil {
			continue
		}
		fn := FunctionDecl{Name: template.Name, Module: module.Name, ReturnType: template.ReturnType, Params: template.Params, Body: template.Body, Span: template.Span}
		state := &evt1AccessFunction{decl: fn, key: module.Name + "::template:" + template.Name, locals: map[string]AccessSubject{}}
		for i, param := range fn.Params {
			state.locals[param.Name] = AccessSubject{Root: AccessIdentity{Kind: "Parameter", Module: module.Name, Function: state.key, Ordinal: i, Name: param.Name, Type: param.Type}, Type: param.Type, Resolution: AccessExact}
		}
		evt1AccessBlock(env, state, *template.Body)
		evt1AccessAttachSynchronizationOrders(state)
		functions[state.key] = state
		byName[template.Name] = append(byName[template.Name], state)
	}
	// Open generic methods are semantic declarations, not module.Functions.
	// Retain their symbolic access contracts in the exporting artifact. The
	// consumer closes these entries against its own concrete instantiations.
	var genericNames []string
	for name := range env.genericTypes {
		genericNames = append(genericNames, name)
	}
	sort.Strings(genericNames)
	for _, name := range genericNames {
		generic := env.genericTypes[name]
		if generic.Module != module.Name {
			continue
		}
		owner := Type{Name: name, Kind: TypeStruct}
		for _, parameter := range generic.Parameters {
			kind := TypeConceptParam
			if parameter.Kind == "value" {
				kind = TypeTemplateValue
			}
			owner.TypeArgs = append(owner.TypeArgs, Type{Name: parameter.Name, Kind: kind})
		}
		for _, method := range generic.Struct.Methods {
			fn := method
			fn.Params = append([]Param(nil), method.Params...)
			fn.Module = module.Name
			fn.MethodOf = owner.String()
			for i := range fn.Params {
				if fn.Params[i].Name == "self" {
					fn.Params[i].Type = owner
				}
			}
			state := &evt1AccessFunction{decl: fn, key: evt1AccessFunctionKey(fn), locals: map[string]AccessSubject{}, openGenericParams: map[string]bool{}}
			for _, parameter := range generic.Parameters {
				state.openGenericParams[parameter.Name] = true
			}
			for i, param := range fn.Params {
				state.locals[param.Name] = AccessSubject{Root: AccessIdentity{Kind: "Parameter", Module: fn.Module, Function: state.key, Ordinal: i, Name: param.Name, Type: param.Type}, Type: param.Type, Resolution: AccessExact}
			}
			if err := evt1AccessApplyAttributes(env, state); err != nil {
				return err
			}
			if fn.Body != nil {
				evt1AccessBlock(env, state, *fn.Body)
				evt1AccessAttachSynchronizationOrders(state)
			}
			for i := range state.entries {
				state.entries[i].OpenGenericOwner = name
				state.entries[i] = evt1FinalizeAccessEntry(state.entries[i])
			}
			functions[state.key] = state
			byName[fn.Name] = append(byName[fn.Name], state)
		}
	}

	// Monotone finite fixpoint. Subject paths are capped by resolver semantics,
	// so recursive substitution cannot construct an unbounded term.
	for changed := true; changed; {
		changed = false
		keys := make([]string, 0, len(functions))
		for key := range functions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			caller := functions[key]
			seen := map[string]int{}
			for index, entry := range caller.entries {
				seen[evt1AccessEntryKey(entry)] = index
			}
			for _, call := range caller.calls {
				candidates := byName[call.callee]
				var callee *evt1AccessFunction
				for _, candidate := range candidates {
					if len(candidate.decl.Params) == len(call.args) {
						if callee != nil {
							callee = nil
							break
						}
						callee = candidate
					}
				}
				if callee == nil {
					continue
				}
				for _, incoming := range callee.entries {
					entry := incoming
					entry.Function = AccessIdentity{Kind: "Function", Module: caller.decl.Module, Function: caller.key, Name: caller.decl.Name}
					entry.Sequence = call.sequence
					if entry.Subject.Root.Kind == "Parameter" && entry.Subject.Root.Ordinal < len(call.args) {
						base := evt1AccessResolveSubject(env, caller, call.args[entry.Subject.Root.Ordinal])
						if len(base.Path)+len(entry.Subject.Path) > 16 {
							base.Resolution = AccessOpaque
							base.Path = nil
						} else {
							base.Path = append(base.Path, entry.Subject.Path...)
							if entry.Subject.Type.Kind != TypeConceptParam && entry.Subject.Type.Name != "T" {
								base.Type = entry.Subject.Type
							}
							if entry.Subject.Resolution == AccessOpaque {
								base.Resolution = AccessOpaque
							}
						}
						entry.Subject = base
					}
					entry = evt1FinalizeAccessEntry(entry)
					entryKey := evt1AccessEntryKey(entry)
					if prior, found := seen[entryKey]; found {
						// Recursive and converging call paths can describe the
						// same structural access at different sequence positions.
						// Keep one bounded entry and choose its earliest position
						// deterministically, independent of fixpoint visitation.
						if entry.Sequence < caller.entries[prior].Sequence {
							caller.entries[prior] = entry
							changed = true
						}
					} else {
						seen[entryKey] = len(caller.entries)
						caller.entries = append(caller.entries, entry)
						changed = true
					}
				}
			}
		}
	}

	contexts := evt1AccessContexts(functions, byName)
	for key, state := range functions {
		contextSet := contexts[key]
		for _, entry := range state.entries {
			if len(contextSet) == 1 {
				for _, context := range contextSet {
					entry.Context = context
				}
			} else if len(contextSet) > 1 {
				entry.Context = AccessIdentity{Kind: "SharedContext", Module: state.decl.Module, Function: state.key}
			} else {
				entry.Context = AccessIdentity{Kind: "OrdinaryContext", Module: state.decl.Module, Function: state.key}
			}
			entry = evt1FinalizeAccessEntry(entry)
			env.accessSummaries = append(env.accessSummaries, entry)
		}
	}
	if err := evt1CloseGenericAccessSummaries(env); err != nil {
		return err
	}
	env.accessSummaries = evt1CanonicalizeAccessEntries(env.accessSummaries)
	evt1SortAccessEntries(env.accessSummaries)
	return evt1ValidateDerivedAccessContracts(env)
}

func evt1AccessApplyAttributes(env *semanticEnv, state *evt1AccessFunction) error {
	for _, attribute := range state.decl.Attributes {
		switch attribute.Name {
		case "semantic_access":
			if len(attribute.Args) != 2 {
				return evt1Diagnostic("ACCESS_CONTRACT_INVALID", "[[semantic_access]] requires an access-kind string and a parameter", attribute.Span)
			}
			kindLiteral, ok := attribute.Args[0].(*StringLiteral)
			parameter, parameterOK := attribute.Args[1].(*NameExpr)
			kind := AccessKind("")
			if ok {
				kind = AccessKind(kindLiteral.Value)
			}
			if !parameterOK || !evt1AccessKindValid(kind) {
				return evt1Diagnostic("ACCESS_CONTRACT_INVALID", "semantic access must use Read, Write, AtomicRead, AtomicWrite, Publish, Consume, Claim, or Commit and name a parameter", attribute.Span)
			}
			subject, found := state.locals[parameter.Name]
			if !found || subject.Root.Kind != "Parameter" {
				return evt1Diagnostic("ACCESS_CONTRACT_INVALID", "semantic access subject must be a function parameter", parameter.Span)
			}
			entry := evt1AccessEntry(state, kind, subject, attribute.Span)
			if state.decl.ExternABI != "" {
				entry.Origin = FactOriginDeclaredForeign
			}
			state.entries = append(state.entries, evt1FinalizeAccessEntry(entry))
		case "execution_context":
			if len(attribute.Args) != 1 {
				return evt1Diagnostic("EXECUTION_CONTEXT_INVALID", "[[execution_context]] requires one context type", attribute.Span)
			}
			name, ok := attribute.Args[0].(*NameExpr)
			if !ok {
				return evt1Diagnostic("EXECUTION_CONTEXT_INVALID", "execution context must be a semantic type name", attribute.Span)
			}
			t := Type{Name: name.Name, Kind: TypeStruct, Span: name.Span}
			if !state.openGenericParams[name.Name] {
				if err := validateKnownType(env, t, name.Span, "", false); err != nil {
					return err
				}
			}
		case "synchronization":
			if len(attribute.Args) != 2 {
				return evt1Diagnostic("SYNCHRONIZATION_CONTRACT_INVALID", "[[synchronization]] requires a mechanism string and a parameter", attribute.Span)
			}
			mechanism, mechanismOK := attribute.Args[0].(*StringLiteral)
			parameter, parameterOK := attribute.Args[1].(*NameExpr)
			if !mechanismOK || !parameterOK || (mechanism.Value != "Acquire" && mechanism.Value != "Release") {
				return evt1Diagnostic("SYNCHRONIZATION_CONTRACT_INVALID", "synchronization mechanism must be Acquire or Release and name a parameter", attribute.Span)
			}
			if state.decl.ReturnType.Name != "void" {
				return evt1Diagnostic("SYNCHRONIZATION_CONTRACT_INVALID", "synchronization acquire/release functions must return void", attribute.Span)
			}
			if subject, found := state.locals[parameter.Name]; !found || subject.Root.Kind != "Parameter" {
				return evt1Diagnostic("SYNCHRONIZATION_CONTRACT_INVALID", "synchronization authority must be a function parameter", parameter.Span)
			}
		}
	}
	return nil
}

func evt1AccessKindValid(kind AccessKind) bool {
	switch kind {
	case AccessRead, AccessWrite, AccessAtomicRead, AccessAtomicWrite, AccessPublish, AccessConsume, AccessClaim, AccessCommit:
		return true
	}
	return false
}

func evt1SynchronizationEffect(fn FunctionDecl) string {
	for _, attribute := range fn.Attributes {
		if attribute.Name != "synchronization" || len(attribute.Args) != 2 {
			continue
		}
		if mechanism, ok := attribute.Args[0].(*StringLiteral); ok {
			return mechanism.Value
		}
	}
	return ""
}

func evt1NamedSynchronizationEffect(env *semanticEnv, name string) string {
	effect := ""
	for _, fn := range env.functions[name] {
		candidate := evt1SynchronizationEffect(fn)
		if candidate == "" {
			continue
		}
		if effect != "" && effect != candidate {
			return ""
		}
		effect = candidate
	}
	return effect
}

func evt1AccessAttachSynchronizationOrders(state *evt1AccessFunction) {
	for i := range state.entries {
		entry := &state.entries[i]
		want := AccessAtomicWrite
		if entry.Operation == AccessConsume {
			want = AccessAtomicRead
		} else if entry.Operation != AccessPublish {
			continue
		}
		for _, atomic := range state.entries {
			if atomic.Operation == want && evt1AccessSubjectsOverlap(entry.Subject, atomic.Subject) && atomic.MemoryOrder != "" {
				entry.MemoryOrder = atomic.MemoryOrder
				entry.Mechanism = atomic.Mechanism
				entry.Sequence = atomic.Sequence
				break
			}
		}
		*entry = evt1FinalizeAccessEntry(*entry)
	}
}

func evt1AccessEntry(state *evt1AccessFunction, kind AccessKind, subject AccessSubject, span Span) MIRAccessEntry {
	state.sequence++
	return MIRAccessEntry{
		Subject: subject, Operation: kind,
		Function: AccessIdentity{Kind: "Function", Module: state.decl.Module, Function: state.key, Name: state.decl.Name},
		Module:   AccessIdentity{Kind: "Module", Module: state.decl.Module, Name: state.decl.Module},
		Origin:   FactOriginDerivedAccessSummary, Sequence: state.sequence, SourceSpan: span,
	}
}

func evt1AccessContexts(functions map[string]*evt1AccessFunction, byName map[string][]*evt1AccessFunction) map[string]map[string]AccessIdentity {
	contexts := map[string]map[string]AccessIdentity{}
	for key, state := range functions {
		for _, attribute := range state.decl.Attributes {
			if attribute.Name != "execution_context" || len(attribute.Args) != 1 {
				continue
			}
			name, ok := attribute.Args[0].(*NameExpr)
			if !ok {
				continue
			}
			kind := TypeStruct
			if state.openGenericParams[name.Name] {
				kind = TypeConceptParam
			}
			id := AccessIdentity{Kind: "ExecutionContext", Module: state.decl.Module, Name: name.Name, Type: Type{Name: name.Name, Kind: kind}}
			contexts[key] = map[string]AccessIdentity{evt1AccessIdentityKey(id): id}
		}
	}
	for changed := true; changed; {
		changed = false
		for key, state := range functions {
			for _, call := range state.calls {
				for _, callee := range byName[call.callee] {
					if len(callee.decl.Params) != len(call.args) {
						continue
					}
					if contexts[callee.key] == nil {
						contexts[callee.key] = map[string]AccessIdentity{}
					}
					for contextKey, context := range contexts[key] {
						if _, exists := contexts[callee.key][contextKey]; !exists {
							contexts[callee.key][contextKey] = context
							changed = true
						}
					}
				}
			}
		}
	}
	return contexts
}

func evt1AccessBlock(env *semanticEnv, state *evt1AccessFunction, block Block) {
	for _, statement := range block.Statements {
		switch s := statement.(type) {
		case *VarDecl:
			evt1AccessExpr(env, state, s.Value)
			state.locals[s.Name] = AccessSubject{Root: AccessIdentity{Kind: "Value", Module: state.decl.Module, Function: state.key, Name: s.Name, Type: s.Type}, Type: s.Type, Resolution: AccessExact}
		case *AssignStmt:
			evt1AccessExpr(env, state, s.Value)
			subject := evt1AccessResolveSubject(env, state, s.Target)
			if evt1AccessRelevant(subject) || subject.Root.Kind == "Opaque" {
				state.entries = append(state.entries, evt1FinalizeAccessEntry(evt1AccessEntry(state, AccessWrite, subject, s.Span)))
			}
			evt1AccessTargetOperands(env, state, s.Target)
		case *ExprStmt:
			evt1AccessExpr(env, state, s.Value)
		case *ReturnStmt:
			evt1AccessExpr(env, state, s.Value)
		case *AssertStmt:
			evt1AccessExpr(env, state, s.Condition)
			evt1AccessExpr(env, state, s.Reason)
		case *StaticAssertStmt:
			evt1AccessExpr(env, state, s.Condition)
			evt1AccessExpr(env, state, s.Message)
		case *IfStmt:
			evt1AccessExpr(env, state, s.Condition)
			evt1AccessBlock(env, state, s.Then)
			if s.Else != nil {
				evt1AccessBlock(env, state, *s.Else)
			}
		case *Block:
			evt1AccessBlock(env, state, *s)
		case *WhileStmt:
			evt1AccessExpr(env, state, s.Condition)
			evt1AccessExpr(env, state, s.Bound)
			evt1AccessBlock(env, state, s.Body)
		case *ForeachStmt:
			evt1AccessExpr(env, state, s.Source)
			evt1AccessBlock(env, state, s.Body)
		case *TryStmt:
			evt1AccessBlock(env, state, s.Body)
			for _, arm := range s.Except {
				evt1AccessBlock(env, state, arm.Body)
			}
		case *MatchStmt:
			evt1AccessExpr(env, state, s.Subject)
			for _, arm := range s.Arms {
				evt1AccessBlock(env, state, arm.Block)
			}
		case *ActuatorLocalDecl:
			evt1AccessExpr(env, state, s.Mechanism)
		case *InstanceDecl:
			evt1AccessExpr(env, state, s.Context)
			for _, arg := range s.StateArgs {
				evt1AccessExpr(env, state, arg)
			}
		case *MachineCompleteStmt:
			evt1AccessExpr(env, state, s.Value)
		case *TransitionMatchStmt:
			evt1AccessExpr(env, state, s.Subject)
		case *TransitionDecideStmt:
			for _, c := range s.Candidates {
				evt1AccessExpr(env, state, c.Guard)
				evt1AccessExpr(env, state, c.Score)
			}
		case *TransitionInferStmt:
			for _, c := range s.Candidates {
				evt1AccessExpr(env, state, c.Guard)
				evt1AccessExpr(env, state, c.Score)
			}
		}
	}
}

func evt1AccessTargetOperands(env *semanticEnv, state *evt1AccessFunction, expr Expr) {
	switch e := expr.(type) {
	case *IndexExpr:
		evt1AccessExpr(env, state, e.Index)
		for _, index := range e.Indices {
			evt1AccessExpr(env, state, index)
		}
	}
}

func evt1AccessExpr(env *semanticEnv, state *evt1AccessFunction, expr Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *NameExpr, *FieldExpr, *IndexExpr:
		subject := evt1AccessResolveSubject(env, state, expr)
		if evt1AccessRelevant(subject) {
			state.entries = append(state.entries, evt1FinalizeAccessEntry(evt1AccessEntry(state, AccessRead, subject, expr.exprSpan())))
		}
		if index, ok := e.(*IndexExpr); ok {
			evt1AccessExpr(env, state, index.Index)
			for _, part := range index.Indices {
				evt1AccessExpr(env, state, part)
			}
		}
	case *CallExpr:
		atomicKind := map[string][]AccessKind{
			evt1AtomicLoad: {AccessAtomicRead}, evt1AtomicStore: {AccessAtomicWrite},
			evt1AtomicExchange: {AccessAtomicRead, AccessAtomicWrite}, evt1AtomicCompareSwap: {AccessAtomicRead, AccessAtomicWrite}, evt1AtomicFetchAdd: {AccessAtomicRead, AccessAtomicWrite},
		}[e.Callee]
		for _, arg := range e.Args {
			evt1AccessExpr(env, state, arg)
		}
		if e.Receiver != nil {
			evt1AccessExpr(env, state, e.Receiver)
		}
		if len(atomicKind) != 0 && len(e.Args) != 0 {
			subject := evt1AccessResolveSubject(env, state, e.Args[0])
			orderAt := len(e.Args) - 1
			if e.Callee == evt1AtomicCompareSwap {
				orderAt = 3
			}
			order := ""
			if orderAt >= 0 && orderAt < len(e.Args) {
				order = evt1ExprIdentity(e.Args[orderAt])
			}
			for _, kind := range atomicKind {
				entry := evt1AccessEntry(state, kind, subject, e.Span)
				entry.MemoryOrder = order
				entry.Mechanism = e.Intrinsic
				if entry.Mechanism == "" {
					entry.Mechanism = e.Callee
				}
				state.entries = append(state.entries, evt1FinalizeAccessEntry(entry))
			}
		}
		state.sequence++
		state.calls = append(state.calls, evt1AccessCall{callee: e.Callee, args: e.Args, span: e.Span, sequence: state.sequence})
	case *TemplateCallExpr:
		for _, arg := range e.Args {
			evt1AccessExpr(env, state, arg)
		}
		state.sequence++
		state.calls = append(state.calls, evt1AccessCall{callee: e.Callee, args: e.Args, span: e.Span, sequence: state.sequence})
	case *BinaryExpr:
		evt1AccessExpr(env, state, e.Left)
		evt1AccessExpr(env, state, e.Right)
	case *UnaryExpr:
		evt1AccessExpr(env, state, e.Value)
	case *MoveExpr:
		evt1AccessExpr(env, state, e.Value)
	case *RefExpr:
		// Forming a reference evaluates its address but does not read the
		// referenced storage. Index operands are still ordinary reads.
		evt1AccessTargetOperands(env, state, e.Value)
	case *ParenExpr:
		evt1AccessExpr(env, state, e.Value)
	case *AwaitExpr:
		evt1AccessExpr(env, state, e.Value)
	case *BindExpr:
		evt1AccessExpr(env, state, e.Source)
	case *WithExpr:
		evt1AccessExpr(env, state, e.Base)
		for _, update := range e.Updates {
			evt1AccessExpr(env, state, update.Value)
		}
	case *ArrayLiteralExpr:
		for _, item := range e.Elements {
			evt1AccessExpr(env, state, item)
		}
	case *RepeatInitializer:
		evt1AccessExpr(env, state, e.Value)
		if e.Count != nil {
			evt1AccessExpr(env, state, e.Count)
		}
	case *StructConstructExpr:
		for _, arg := range e.Args {
			evt1AccessExpr(env, state, arg)
		}
	case *ConstructExpr:
		for _, arg := range e.Args {
			evt1AccessExpr(env, state, arg)
		}
	case *IfExpr:
		evt1AccessExpr(env, state, e.Condition)
		evt1AccessExpr(env, state, e.Then)
		evt1AccessExpr(env, state, e.Else)
	case *MatchExpr:
		evt1AccessExpr(env, state, e.Subject)
		for _, arm := range e.Arms {
			evt1AccessExpr(env, state, arm.Value)
		}
	case *DispatchExpr:
		evt1AccessExpr(env, state, e.Signal)
	}
}

func evt1AccessRelevant(subject AccessSubject) bool {
	// Local scalar/value temporaries cannot be observed by another execution
	// context. Parameter-rooted storage is the bounded shared-state surface in
	// R7d4; persistent machine/capture roots can be added structurally later.
	return subject.Root.Kind == "Parameter"
}

func evt1AccessResolveSubject(env *semanticEnv, state *evt1AccessFunction, expr Expr) AccessSubject {
	switch e := expr.(type) {
	case *RefExpr:
		return evt1AccessResolveSubject(env, state, e.Value)
	case *ParenExpr:
		return evt1AccessResolveSubject(env, state, e.Value)
	case *NameExpr:
		if subject, ok := state.locals[e.Name]; ok {
			return subject
		}
		return AccessSubject{Root: AccessIdentity{Kind: "Value", Module: state.decl.Module, Function: state.key, Name: e.Name}, Resolution: AccessOpaque}
	case *FieldExpr:
		base := evt1AccessResolveSubject(env, state, e.Receiver)
		ordinal, fieldType := evt1AccessField(env, base.Type, e.Field)
		base.Path = append(base.Path, AccessPathElement{Kind: "Field", Field: e.Field, Ordinal: ordinal})
		if fieldType.Name != "" {
			base.Type = fieldType
		}
		if e.RegionID != "" {
			base.RegionID, base.Offset, base.Extent, base.Resolution = e.RegionID, e.RegionOffset, e.RegionExtent, AccessInterval
		}
		return base
	case *IndexExpr:
		base := evt1AccessResolveSubject(env, state, e.Base)
		indices := e.Indices
		if len(indices) == 0 && e.Index != nil {
			indices = []Expr{e.Index}
		}
		for _, index := range indices {
			literal, ok := index.(*IntLiteral)
			if !ok || literal.Negative || literal.Magnitude > uint64(^uint(0)>>1) {
				base.Resolution = AccessOpaque
				base.Path = append(base.Path, AccessPathElement{Kind: "DynamicIndex"})
				return base
			}
			base.Path = append(base.Path, AccessPathElement{Kind: "Index", Index: int64(literal.Magnitude)})
			base.Offset += int(literal.Magnitude)
			base.Extent = 1
			base.Resolution = AccessInterval
		}
		if base.Type.ArrayElem != nil {
			base.Type = *base.Type.ArrayElem
		}
		return base
	default:
		return AccessSubject{Root: AccessIdentity{Kind: "Opaque", Module: state.decl.Module, Function: state.key}, Resolution: AccessOpaque}
	}
}

func evt1AccessField(env *semanticEnv, owner Type, name string) (int, Type) {
	decl, ok := env.structs[owner.Name]
	if !ok {
		decl, ok = evt1StructView(env, owner)
		if !ok {
			return -1, Type{}
		}
	}
	for i, field := range decl.Fields {
		if field.Name == name {
			return i, field.Type
		}
	}
	return -1, Type{}
}

func evt1SortAccessEntries(entries []MIRAccessEntry) {
	sort.Slice(entries, func(i, j int) bool { return evt1AccessEntryKey(entries[i]) < evt1AccessEntryKey(entries[j]) })
}

func evt1CanonicalizeAccessEntries(entries []MIRAccessEntry) []MIRAccessEntry {
	byKey := map[string]MIRAccessEntry{}
	for _, entry := range entries {
		key := evt1AccessEntryKey(entry)
		prior, found := byKey[key]
		if !found || entry.Sequence < prior.Sequence {
			byKey[key] = entry
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]MIRAccessEntry, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func evt1AccessSubjectMatches(subject AccessSubject, requested Type) bool {
	wanted := requested.valueType().String()
	return subject.Type.valueType().String() == wanted || subject.Root.Type.valueType().String() == wanted
}

func evt1DerivedSharedAccess(env *semanticEnv, kind SemanticFactKind, args []Type) semanticFactResult {
	if kind == FactSynchronizedAccess && len(args) == 1 {
		return evt1DerivedSynchronizedAccess(env, args[0])
	}
	if kind == FactPublishedBefore && len(args) == 2 {
		return evt1DerivedPublishedBefore(env, args[0], args[1])
	}
	if kind == FactExactlyOnce && len(args) == 1 {
		return evt1DerivedExactlyOnce(env, args[0])
	}
	if len(args) != 2 {
		return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginCompilerAnalysis, Evidence: SemanticFactEvidence{Detail: "synchronization relation has unsupported subject arity"}}
	}
	operation := AccessWrite
	switch kind {
	case FactExclusiveWriter:
		operation = AccessWrite
	case FactSingleProducer:
		operation = AccessPublish
	case FactSingleConsumer:
		operation = AccessConsume
	default:
		return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginCompilerAnalysis, Evidence: SemanticFactEvidence{Detail: "no bounded derived rule matches this synchronization relation"}}
	}
	wantedContext, wantedSubject := args[0], args[1]
	contexts := map[string]string{}
	var resolved []MIRAccessEntry
	opaque := false
	var evidence []string
	for _, entry := range env.accessSummaries {
		matchesOperation := entry.Operation == operation
		if kind == FactExclusiveWriter {
			matchesOperation = entry.Operation == AccessWrite || entry.Operation == AccessAtomicWrite
		}
		if !matchesOperation || !evt1AccessSubjectMatches(entry.Subject, wantedSubject) {
			continue
		}
		if entry.Resolution == AccessOpaque || entry.Context.Kind == "SharedContext" || entry.Context.Kind == "OrdinaryContext" || entry.Context.Type.Name == "" {
			opaque = true
			evidence = append(evidence, fmt.Sprintf("%s %s in %s is unresolved", entry.Operation, wantedSubject.String(), entry.Function.Name))
			continue
		}
		key := entry.Context.Type.String()
		contexts[key] = entry.Context.Name
		resolved = append(resolved, entry)
		evidence = append(evidence, fmt.Sprintf("%s %s by %s in %s", entry.Operation, wantedSubject.String(), key, entry.Function.Name))
	}
	sort.Strings(evidence)
	detail := strings.Join(evidence, "; ")
	if len(contexts) > 1 {
		for i := range resolved {
			for j := i + 1; j < len(resolved); j++ {
				if resolved[i].Context.Type.String() != resolved[j].Context.Type.String() && evt1AccessSubjectsOverlap(resolved[i].Subject, resolved[j].Subject) {
					return semanticFactResult{Outcome: FactDisproven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail, Authority: "derived access set"}}
				}
			}
		}
		return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail + "; known fixed paths are disjoint", Authority: "derived access set"}}
	}
	if opaque || len(contexts) == 0 {
		if detail == "" {
			detail = "no closed access set for " + wantedSubject.String()
		}
		return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail, Authority: "derived access set"}}
	}
	for context := range contexts {
		if context == wantedContext.String() {
			return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail, Authority: "derived access set"}}
		}
	}
	return semanticFactResult{Outcome: FactDisproven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail, Authority: "derived access set"}}
}

func evt1MemoryOrderAcquire(order string) bool {
	return strings.Contains(order, "Acquire") || strings.Contains(order, "Sequential")
}

func evt1MemoryOrderRelease(order string) bool {
	return strings.Contains(order, "Release") || strings.Contains(order, "Sequential")
}

func evt1DerivedSynchronizedAccess(env *semanticEnv, subject Type) semanticFactResult {
	var accesses []MIRAccessEntry
	contexts := map[string]bool{}
	allAtomic := true
	for _, entry := range env.accessSummaries {
		if !evt1AccessSubjectMatches(entry.Subject, subject) {
			continue
		}
		switch entry.Operation {
		case AccessRead, AccessWrite, AccessAtomicRead, AccessAtomicWrite:
			accesses = append(accesses, entry)
			if entry.Context.Type.Name == "" || entry.Context.Kind == "SharedContext" || entry.Context.Kind == "OrdinaryContext" || entry.Resolution == AccessOpaque {
				return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "shared access set contains unresolved context or subject", Authority: "derived synchronization mechanisms"}}
			}
			contexts[entry.Context.Type.String()] = true
			allAtomic = allAtomic && (entry.Operation == AccessAtomicRead || entry.Operation == AccessAtomicWrite)
		}
	}
	if len(accesses) == 0 {
		return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "no closed access set for " + subject.String(), Authority: "derived synchronization mechanisms"}}
	}
	if len(contexts) == 1 {
		return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "all accesses are confined to one explicit execution context", Authority: "exclusive context ownership"}}
	}
	if allAtomic {
		return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "all cross-context accesses use C11 atomic operations", Authority: "derived synchronization mechanisms"}}
	}
	for i := range accesses {
		for j := i + 1; j < len(accesses); j++ {
			left, right := accesses[i], accesses[j]
			if left.Context.Type.String() == right.Context.Type.String() || !evt1AccessSubjectsOverlap(left.Subject, right.Subject) {
				continue
			}
			leftWrite := left.Operation == AccessWrite || left.Operation == AccessAtomicWrite
			rightWrite := right.Operation == AccessWrite || right.Operation == AccessAtomicWrite
			if (leftWrite || rightWrite) && (left.Operation == AccessRead || left.Operation == AccessWrite || right.Operation == AccessRead || right.Operation == AccessWrite) {
				return semanticFactResult{Outcome: FactDisproven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "overlapping cross-context access is not mediated by one of the observed synchronization mechanisms", Authority: "derived synchronization mechanisms"}}
			}
		}
	}
	return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "all cross-context conflicts are atomic or structurally disjoint", Authority: "derived synchronization mechanisms"}}
}

func evt1DerivedPublishedBefore(env *semanticEnv, written, observed Type) semanticFactResult {
	var writes, reads, publishes, consumes []MIRAccessEntry
	for _, entry := range env.accessSummaries {
		switch entry.Operation {
		case AccessWrite, AccessAtomicWrite:
			if evt1AccessSubjectMatches(entry.Subject, written) {
				writes = append(writes, entry)
			}
		case AccessRead, AccessAtomicRead:
			if evt1AccessSubjectMatches(entry.Subject, observed) {
				reads = append(reads, entry)
			}
		case AccessPublish:
			if evt1MemoryOrderRelease(entry.MemoryOrder) {
				publishes = append(publishes, entry)
			}
		case AccessConsume:
			if evt1MemoryOrderAcquire(entry.MemoryOrder) {
				consumes = append(consumes, entry)
			}
		}
	}
	for _, write := range writes {
		for _, publish := range publishes {
			if write.Context.Type.String() == "" || write.Context.Type.String() != publish.Context.Type.String() || write.Sequence >= publish.Sequence {
				continue
			}
			for _, consume := range consumes {
				if !evt1AccessSubjectsOverlap(publish.Subject, consume.Subject) || consume.Context.Type.String() == "" || consume.Context.Type.String() == write.Context.Type.String() {
					continue
				}
				for _, read := range reads {
					if read.Context.Type.String() == consume.Context.Type.String() && consume.Sequence < read.Sequence {
						detail := fmt.Sprintf("%s writes %s before release publication; %s acquires the same publication before reading %s", write.Context.Type.String(), written.String(), read.Context.Type.String(), observed.String())
						return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: detail, Authority: "release/acquire publication graph"}}
					}
				}
			}
		}
	}
	return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "no release/acquire Write-Publish-Consume-Read path connects the ordered subjects", Authority: "release/acquire publication graph"}}
}

func evt1DerivedExactlyOnce(env *semanticEnv, obligation Type) semanticFactResult {
	var claims, commits []MIRAccessEntry
	casByFunction := map[string]bool{}
	resetWrite := false
	for _, entry := range env.accessSummaries {
		if !evt1AccessSubjectMatches(entry.Subject, obligation) || entry.Resolution == AccessOpaque {
			continue
		}
		switch entry.Operation {
		case AccessClaim:
			claims = append(claims, entry)
		case AccessCommit:
			commits = append(commits, entry)
		case AccessAtomicWrite:
			if entry.Mechanism == "atomic_compare_exchange" || entry.Mechanism == evt1AtomicCompareSwap {
				casByFunction[entry.Function.Function] = true
			} else {
				resetWrite = true
			}
		}
	}
	if len(claims) == 1 && len(commits) == 1 && casByFunction[claims[0].Function.Function] && casByFunction[commits[0].Function.Function] && !resetWrite {
		return semanticFactResult{Outcome: FactProven, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "exact obligation identity uses one CAS claim transition and one CAS commit transition with no reset", Authority: "derived claim/commit state machine"}}
	}
	return semanticFactResult{Outcome: FactUnknown, Origin: FactOriginDerivedAccessSummary, Evidence: SemanticFactEvidence{Detail: "claim/commit CAS authority or no-reset evidence is incomplete", Authority: "derived claim/commit state machine"}}
}

func evt1AccessSubjectsOverlap(a, b AccessSubject) bool {
	if a.Resolution == AccessOpaque || b.Resolution == AccessOpaque {
		return true
	}
	limit := len(a.Path)
	if len(b.Path) < limit {
		limit = len(b.Path)
	}
	for i := 0; i < limit; i++ {
		left, right := a.Path[i], b.Path[i]
		if left.Kind == "Field" && right.Kind == "Field" && left.Ordinal >= 0 && right.Ordinal >= 0 && left.Ordinal != right.Ordinal {
			return false
		}
		if left.Kind == "Index" && right.Kind == "Index" && left.Index != right.Index {
			return false
		}
	}
	if a.Resolution == AccessInterval && b.Resolution == AccessInterval && a.RegionID != "" && a.RegionID == b.RegionID && a.Extent > 0 && b.Extent > 0 {
		return a.Offset < b.Offset+b.Extent && b.Offset < a.Offset+a.Extent
	}
	return true
}

func evt1ValidateDerivedAccessContracts(env *semanticEnv) error {
	for _, fact := range env.sharedAccessFacts {
		if fact.Origin != FactOriginDeclared || len(fact.Subjects) != 2 {
			continue
		}
		if fact.Kind != FactExclusiveWriter && fact.Kind != FactSingleProducer && fact.Kind != FactSingleConsumer {
			continue
		}
		args := []Type{{Name: fact.Subjects[0].Type}, {Name: fact.Subjects[1].Type}}
		result := evt1DerivedSharedAccess(env, fact.Kind, args)
		if result.Outcome == FactDisproven {
			return evt1Diagnostic("SYNC_AUTHORITY_CONTRADICTION", result.Evidence.Detail, fact.SourceSpan)
		}
	}
	return nil
}

func evt1LocalAccessSummaries(module Module, env *semanticEnv) []MIRAccessEntry {
	var entries []MIRAccessEntry
	for _, entry := range env.accessSummaries {
		if entry.Module.Module == module.Name && entry.Origin == FactOriginDerivedAccessSummary {
			entries = append(entries, entry)
		}
	}
	evt1SortAccessEntries(entries)
	return entries
}
