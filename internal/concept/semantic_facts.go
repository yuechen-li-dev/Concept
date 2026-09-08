package concept

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SemanticFactKind is the closed R4j vocabulary shared by concepts, MIR, and
// future backend-independent optimization consumers.
type SemanticFactKind string

const (
	FactContiguous          SemanticFactKind = "Contiguous"
	FactBounded             SemanticFactKind = "Bounded"
	FactMutable             SemanticFactKind = "Mutable"
	FactReadonly            SemanticFactKind = "Readonly"
	FactAligned             SemanticFactKind = "Aligned"
	FactRegionIdentity      SemanticFactKind = "RegionIdentity"
	FactDisjoint            SemanticFactKind = "Disjoint"
	FactSameRegion          SemanticFactKind = "SameRegion"
	FactFixedShape          SemanticFactKind = "FixedShape"
	FactRuntimeShape        SemanticFactKind = "RuntimeShape"
	FactRank                SemanticFactKind = "Rank"
	FactShape               SemanticFactKind = "Shape"
	FactNoAllocation        SemanticFactKind = "NoAllocation"
	FactNoCopy              SemanticFactKind = "NoCopy"
	FactNoOwnershipTransfer SemanticFactKind = "NoOwnershipTransfer"
	FactLifetimeSafe        SemanticFactKind = "LifetimeSafe"
	FactNonEscaping         SemanticFactKind = "NonEscaping"
	FactOutlives            SemanticFactKind = "Outlives"
)

type SemanticFactOrigin string

const (
	FactOriginDeclared         SemanticFactOrigin = "Declared"
	FactOriginType             SemanticFactOrigin = "DerivedFromType"
	FactOriginLayout           SemanticFactOrigin = "DerivedFromLayout"
	FactOriginBind             SemanticFactOrigin = "DerivedFromBind"
	FactOriginSpan             SemanticFactOrigin = "DerivedFromSpan"
	FactOriginStream           SemanticFactOrigin = "DerivedFromStream"
	FactOriginTensorBacking    SemanticFactOrigin = "DerivedFromTensorBacking"
	FactOriginControlFlow      SemanticFactOrigin = "DerivedFromControlFlow"
	FactOriginCompilerAnalysis SemanticFactOrigin = "CompilerAnalysis"
)

type SemanticFactCertainty string

const (
	FactProven    SemanticFactCertainty = "proven"
	FactDisproven SemanticFactCertainty = "disproven"
	FactUnknown   SemanticFactCertainty = "unknown"
)

type SemanticFactSubject struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Function string `json:"function,omitempty"`
	Type     string `json:"type,omitempty"`
	RegionID string `json:"region_id,omitempty"`
}

type SemanticFactEvidence struct {
	RegionIDs []string           `json:"region_ids,omitempty"`
	Offset    int                `json:"offset,omitempty"`
	Extent    int                `json:"extent,omitempty"`
	Alignment int                `json:"alignment,omitempty"`
	Rank      int                `json:"rank,omitempty"`
	Shape     []StorageDimension `json:"shape,omitempty"`
	Detail    string             `json:"detail,omitempty"`
}

type MIRSemanticFact struct {
	ID         string                `json:"id"`
	Kind       SemanticFactKind      `json:"kind"`
	Subjects   []SemanticFactSubject `json:"subjects"`
	Parameters []int                 `json:"parameters,omitempty"`
	Outcome    SemanticFactCertainty `json:"outcome"`
	Origin     SemanticFactOrigin    `json:"origin"`
	Evidence   SemanticFactEvidence  `json:"evidence,omitempty"`
	SourceSpan Span                  `json:"source_span"`
}

type semanticFactResult struct {
	Outcome  SemanticFactCertainty
	Origin   SemanticFactOrigin
	Evidence SemanticFactEvidence
}

func evt1FactKind(name string) SemanticFactKind { return SemanticFactKind(name) }

func evt1SemanticFactID(kind SemanticFactKind, subjects []SemanticFactSubject, parameters []int, span Span) string {
	parts := []string{string(kind), fmt.Sprintf("%d:%d", span.Line, span.Column)}
	for _, subject := range subjects {
		parts = append(parts, subject.Kind+":"+subject.Function+":"+subject.Name+":"+subject.Type+":"+subject.RegionID)
	}
	for _, parameter := range parameters {
		parts = append(parts, strconv.Itoa(parameter))
	}
	return "fact-" + digest([]byte(strings.Join(parts, "|")))[:16]
}

func evt1TypeFact(env *semanticEnv, kind SemanticFactKind, t Type, parameters []int) semanticFactResult {
	resolved, err := evt1ResolveType(env, nil, t)
	if err == nil {
		t = resolved
	}
	origin := FactOriginType
	isStorage := t.ArrayElem != nil && t.Contiguous
	isSpan := evt1IsSpanType(t)
	isTensor := evt1IsTensorType(t)
	_, isLayout := env.layouts[t.Name]
	_, isStream := env.streams[t.Name]
	if isSpan {
		origin = FactOriginSpan
	}
	if isTensor {
		origin = FactOriginTensorBacking
	}
	if isLayout {
		origin = FactOriginLayout
	}
	if isStream {
		origin = FactOriginStream
	}
	result := semanticFactResult{Outcome: FactUnknown, Origin: origin}
	switch kind {
	case FactContiguous, FactBounded:
		if isStorage || isSpan || isTensor {
			result.Outcome = FactProven
		}
	case FactFixedShape:
		if isStorage {
			if evt1StorageHasRuntimeShape(t) {
				result.Outcome = FactDisproven
			} else {
				result.Outcome = FactProven
			}
		}
	case FactRuntimeShape:
		if isSpan {
			result.Outcome = FactProven
		} else if isStorage {
			if evt1StorageHasRuntimeShape(t) {
				result.Outcome = FactProven
			} else {
				result.Outcome = FactDisproven
			}
		}
	case FactRank:
		if len(parameters) != 1 {
			return result
		}
		rank := 0
		if isStorage {
			rank = evt1StorageRank(t)
		}
		if isSpan {
			rank = 1
		}
		if isTensor {
			rank = t.TensorRank
		}
		if rank > 0 {
			result.Evidence.Rank = rank
			if rank == parameters[0] {
				result.Outcome = FactProven
			} else {
				result.Outcome = FactDisproven
			}
		}
	case FactAligned:
		if len(parameters) != 1 {
			return result
		}
		alignment := 0
		if isLayout {
			alignment = env.layouts[t.Name].Alignment
		}
		if isStream {
			alignment = env.layouts[env.streams[t.Name].LayoutName].Alignment
		}
		if isStorage && !evt1StorageHasRuntimeShape(t) {
			alignment = evt1InlineStorageAlignment
		}
		if (isStorage || isSpan || isTensor) && alignment == 0 {
			element := Type{}
			if t.ArrayElem != nil {
				element = *t.ArrayElem
			}
			if isSpan {
				element = evt1SpanElement(t)
			}
			if isTensor {
				element = evt1TensorElement(t)
			}
			_, alignment, _ = evt1TypeGeometry(env, element.valueType())
		}
		result.Evidence.Alignment = alignment
		if alignment > 0 && alignment%parameters[0] == 0 {
			result.Outcome = FactProven
		}
	case FactMutable:
		if t.Const || (isSpan && !evt1SpanMutable(t)) {
			result.Outcome = FactDisproven
		} else if isStorage || isSpan || isTensor || isLayout || isStream {
			result.Outcome = FactProven
		}
	case FactReadonly:
		if t.Const || (isSpan && !evt1SpanMutable(t)) {
			result.Outcome = FactProven
		} else if isStorage || isSpan || isTensor || isLayout || isStream {
			result.Outcome = FactDisproven
		}
	case FactNoAllocation:
		if isStorage || isSpan || isTensor || isLayout || isStream {
			result.Outcome = FactProven
		}
	case FactNoCopy, FactNoOwnershipTransfer:
		if isSpan || isTensor || isLayout || isStream || (isStorage && t.isReference()) {
			result.Outcome = FactProven
		}
	case FactLifetimeSafe:
		if t.isReference() || t.Scoped || evt1IsRefStructType(env, t) {
			result.Outcome = FactProven
		} else {
			result.Outcome = FactDisproven
		}
	case FactNonEscaping:
		if t.Scoped || evt1IsRefStructType(env, t) {
			result.Outcome = FactProven
		} else {
			result.Outcome = FactDisproven
		}
	}
	return result
}

// SemanticFactSet is the bounded optimizer-facing query surface. It consumes
// qualified MIR facts and never reconstructs source intent.
type SemanticFactSet struct{ facts []MIRSemanticFact }

func NewSemanticFactSet(facts []MIRSemanticFact) SemanticFactSet {
	copyFacts := append([]MIRSemanticFact{}, facts...)
	return SemanticFactSet{facts: copyFacts}
}

func (s SemanticFactSet) FactsFor(subject string) []MIRSemanticFact {
	if subject == "" {
		return nil
	}
	var out []MIRSemanticFact
	for _, fact := range s.facts {
		for _, candidate := range fact.Subjects {
			if candidate.Name == subject || candidate.RegionID == subject {
				out = append(out, fact)
				break
			}
		}
	}
	return out
}

func (s SemanticFactSet) Prove(kind SemanticFactKind, subjects ...string) SemanticFactCertainty {
	want := append([]string{}, subjects...)
	sort.Strings(want)
	for _, fact := range s.facts {
		if fact.Kind != kind {
			continue
		}
		got := make([]string, 0, len(fact.Subjects))
		for _, subject := range fact.Subjects {
			identity := subject.Name
			for _, requested := range want {
				if subject.RegionID != "" && requested == subject.RegionID {
					identity = subject.RegionID
					break
				}
			}
			got = append(got, identity)
		}
		sort.Strings(got)
		if strings.Join(got, "\x00") == strings.Join(want, "\x00") {
			return fact.Outcome
		}
	}
	return FactUnknown
}

func (s SemanticFactSet) KnownAlignment(subject string) (int, bool) {
	best := 0
	for _, fact := range s.FactsFor(subject) {
		if fact.Kind == FactAligned && fact.Outcome == FactProven && fact.Evidence.Alignment > best {
			best = fact.Evidence.Alignment
		}
	}
	return best, best > 0
}

func (s SemanticFactSet) RegionOf(subject string) (string, bool) {
	for _, fact := range s.FactsFor(subject) {
		if fact.Kind == FactRegionIdentity && fact.Outcome == FactProven && len(fact.Evidence.RegionIDs) == 1 {
			return fact.Evidence.RegionIDs[0], true
		}
	}
	return "", false
}

func (s SemanticFactSet) AreDisjoint(a, b string) SemanticFactCertainty {
	return s.Prove(FactDisjoint, a, b)
}

func evt1AppendFact(facts *[]MIRSemanticFact, kind SemanticFactKind, subjects []SemanticFactSubject, parameters []int, origin SemanticFactOrigin, evidence SemanticFactEvidence, span Span) {
	fact := MIRSemanticFact{Kind: kind, Subjects: subjects, Parameters: parameters, Outcome: FactProven, Origin: origin, Evidence: evidence, SourceSpan: span}
	fact.ID = evt1SemanticFactID(kind, subjects, parameters, span)
	*facts = append(*facts, fact)
}

func evt1QualifyMIRFacts(mir *MIR) {
	regionEvidence := map[string]SemanticFactEvidence{}
	disjointRegions := map[string]bool{}
	for _, storage := range mir.StorageTypes {
		subject := SemanticFactSubject{Kind: "type", Name: storage.Type.String(), Type: storage.Type.String()}
		evidence := SemanticFactEvidence{Alignment: evt1InlineStorageAlignment, Rank: storage.Rank, Shape: append([]StorageDimension{}, storage.Shape...), Detail: "contiguous row-major storage type"}
		for _, kind := range []SemanticFactKind{FactContiguous, FactBounded, FactFixedShape, FactShape, FactNoAllocation} {
			evt1AppendFact(&mir.SemanticFacts, kind, []SemanticFactSubject{subject}, nil, FactOriginType, evidence, storage.Type.Span)
		}
		evt1AppendFact(&mir.SemanticFacts, FactRank, []SemanticFactSubject{subject}, []int{storage.Rank}, FactOriginType, evidence, storage.Type.Span)
		evt1AppendFact(&mir.SemanticFacts, FactAligned, []SemanticFactSubject{subject}, []int{evt1InlineStorageAlignment}, FactOriginType, evidence, storage.Type.Span)
	}
	for _, layout := range mir.Layouts {
		for _, region := range layout.Regions {
			subject := SemanticFactSubject{Kind: "storage_region", Name: region.ID, Type: region.Type.String(), RegionID: region.ID}
			evidence := SemanticFactEvidence{RegionIDs: []string{region.ID}, Offset: region.Offset, Extent: region.ByteExtent, Alignment: region.Alignment, Rank: evt1StorageRank(region.Type), Shape: append([]StorageDimension{}, region.Shape...)}
			regionEvidence[region.ID] = evidence
			for _, kind := range []SemanticFactKind{FactRegionIdentity, FactContiguous, FactBounded, FactFixedShape, FactNoAllocation} {
				evt1AppendFact(&mir.SemanticFacts, kind, []SemanticFactSubject{subject}, nil, FactOriginLayout, evidence, layout.SourceSpan)
			}
			evt1AppendFact(&mir.SemanticFacts, FactAligned, []SemanticFactSubject{subject}, []int{region.Alignment}, FactOriginLayout, evidence, layout.SourceSpan)
			if evidence.Rank > 0 {
				evt1AppendFact(&mir.SemanticFacts, FactRank, []SemanticFactSubject{subject}, []int{evidence.Rank}, FactOriginLayout, evidence, layout.SourceSpan)
			}
			for _, other := range region.DisjointWith {
				disjointRegions[region.ID+"\x00"+other] = true
				disjointRegions[other+"\x00"+region.ID] = true
				if region.ID < other {
					otherSubject := SemanticFactSubject{Kind: "storage_region", Name: other, RegionID: other}
					evt1AppendFact(&mir.SemanticFacts, FactDisjoint, []SemanticFactSubject{subject, otherSubject}, nil, FactOriginLayout, SemanticFactEvidence{RegionIDs: []string{region.ID, other}, Detail: "distinct declared layout regions"}, layout.SourceSpan)
				}
			}
		}
	}
	for _, stream := range mir.Streams {
		for _, channel := range stream.Channels {
			subject := SemanticFactSubject{Kind: "stream_channel", Name: stream.Name + "." + channel.Name, Type: channel.Type.String(), RegionID: channel.RegionID}
			evidence := regionEvidence[channel.RegionID]
			evidence.Rank = evt1StorageRank(channel.Type)
			evidence.Shape = append([]StorageDimension{}, channel.Type.Shape...)
			evidence.Detail = "zero-storage channel preserves mapped layout region"
			for _, kind := range []SemanticFactKind{FactRegionIdentity, FactContiguous, FactBounded, FactFixedShape, FactNoAllocation, FactNoCopy, FactNoOwnershipTransfer} {
				evt1AppendFact(&mir.SemanticFacts, kind, []SemanticFactSubject{subject}, nil, FactOriginStream, evidence, stream.SourceSpan)
			}
			if evidence.Alignment > 0 {
				evt1AppendFact(&mir.SemanticFacts, FactAligned, []SemanticFactSubject{subject}, []int{evidence.Alignment}, FactOriginStream, evidence, stream.SourceSpan)
			}
			regionSubject := SemanticFactSubject{Kind: "storage_region", Name: channel.RegionID, RegionID: channel.RegionID}
			evt1AppendFact(&mir.SemanticFacts, FactSameRegion, []SemanticFactSubject{regionSubject, subject}, nil, FactOriginStream, evidence, stream.SourceSpan)
		}
	}
	for _, fn := range mir.Functions {
		inline := []SemanticFactSubject{}
		regionValues := map[string][]struct {
			subject  SemanticFactSubject
			evidence SemanticFactEvidence
		}{}
		for _, operation := range fn.Operations {
			if operation.RegionID == "" {
				continue
			}
			name := operation.Detail
			if name == "" {
				name = operation.ID
			}
			subject := SemanticFactSubject{Kind: "value", Name: name, Function: fn.Name, Type: operation.Type, RegionID: operation.RegionID}
			evidence := SemanticFactEvidence{RegionIDs: []string{operation.RegionID}, Offset: operation.Offset, Alignment: operation.Alignment, Rank: operation.TargetRank, Shape: append([]StorageDimension{}, operation.TargetShape...), Detail: operation.Kind}
			if extent, ok := evt1FactStaticInt(operation.ByteExtentExpression); ok {
				evidence.Extent = extent
			}
			if elementOffset, ok := evt1FactStaticInt(operation.BaseOffset); ok && evidence.Extent > 0 {
				if length, lengthOK := evt1FactStaticInt(operation.Length); lengthOK && length > 0 {
					evidence.Offset += elementOffset * (evidence.Extent / length)
				}
			}
			evt1AppendFact(&mir.SemanticFacts, FactRegionIdentity, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			if operation.Contiguous || operation.Kind == "bind_storage" {
				evt1AppendFact(&mir.SemanticFacts, FactContiguous, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.Length != "" || operation.TargetRank > 0 {
				evt1AppendFact(&mir.SemanticFacts, FactBounded, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.Alignment > 0 {
				evt1AppendFact(&mir.SemanticFacts, FactAligned, []SemanticFactSubject{subject}, []int{operation.Alignment}, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.Mutability == "mutable" {
				evt1AppendFact(&mir.SemanticFacts, FactMutable, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			} else if operation.Mutability == "readonly" {
				evt1AppendFact(&mir.SemanticFacts, FactReadonly, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.NoAllocation {
				evt1AppendFact(&mir.SemanticFacts, FactNoAllocation, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.NoCopy {
				evt1AppendFact(&mir.SemanticFacts, FactNoCopy, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.NoOwnershipTransfer {
				evt1AppendFact(&mir.SemanticFacts, FactNoOwnershipTransfer, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.TargetRank > 0 {
				evt1AppendFact(&mir.SemanticFacts, FactRank, []SemanticFactSubject{subject}, []int{operation.TargetRank}, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
				fixed := true
				for _, dimension := range operation.TargetShape {
					fixed = fixed && !dimension.Runtime
				}
				shapeKind := FactRuntimeShape
				if fixed {
					shapeKind = FactFixedShape
				}
				evt1AppendFact(&mir.SemanticFacts, shapeKind, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
				evt1AppendFact(&mir.SemanticFacts, FactShape, []SemanticFactSubject{subject}, nil, evt1FactOriginForOperation(operation), evidence, operation.SourceSpan)
			}
			if operation.Kind == "tensor_inline_storage" {
				inline = append(inline, subject)
			}
			regionValues[operation.RegionID] = append(regionValues[operation.RegionID], struct {
				subject  SemanticFactSubject
				evidence SemanticFactEvidence
			}{subject, evidence})
		}
		for i := 0; i < len(inline); i++ {
			for j := i + 1; j < len(inline); j++ {
				evt1AppendFact(&mir.SemanticFacts, FactDisjoint, []SemanticFactSubject{inline[i], inline[j]}, nil, FactOriginTensorBacking, SemanticFactEvidence{RegionIDs: []string{inline[i].RegionID, inline[j].RegionID}, Detail: "independent inline tensor backing regions"}, fn.SourceSpan)
			}
		}
		var allValues []struct {
			subject  SemanticFactSubject
			evidence SemanticFactEvidence
		}
		for _, values := range regionValues {
			for i := 0; i < len(values); i++ {
				for j := i + 1; j < len(values); j++ {
					evt1AppendFact(&mir.SemanticFacts, FactSameRegion, []SemanticFactSubject{values[i].subject, values[j].subject}, nil, FactOriginCompilerAnalysis, SemanticFactEvidence{RegionIDs: []string{values[i].subject.RegionID}, Detail: "stable parent region identity"}, fn.SourceSpan)
					if values[i].evidence.Extent > 0 && values[j].evidence.Extent > 0 && (values[i].evidence.Offset <= int(^uint(0)>>1)-values[i].evidence.Extent && values[j].evidence.Offset <= int(^uint(0)>>1)-values[j].evidence.Extent) {
						endI, endJ := values[i].evidence.Offset+values[i].evidence.Extent, values[j].evidence.Offset+values[j].evidence.Extent
						if endI <= values[j].evidence.Offset || endJ <= values[i].evidence.Offset {
							evt1AppendFact(&mir.SemanticFacts, FactDisjoint, []SemanticFactSubject{values[i].subject, values[j].subject}, nil, FactOriginCompilerAnalysis, SemanticFactEvidence{RegionIDs: []string{values[i].subject.RegionID}, Detail: "non-overlapping proven half-open intervals"}, fn.SourceSpan)
						}
					}
				}
			}
			allValues = append(allValues, values...)
		}
		for i := 0; i < len(allValues); i++ {
			for j := i + 1; j < len(allValues); j++ {
				if disjointRegions[allValues[i].subject.RegionID+"\x00"+allValues[j].subject.RegionID] {
					evt1AppendFact(&mir.SemanticFacts, FactDisjoint, []SemanticFactSubject{allValues[i].subject, allValues[j].subject}, nil, FactOriginLayout, SemanticFactEvidence{RegionIDs: []string{allValues[i].subject.RegionID, allValues[j].subject.RegionID}, Detail: "distinct declared layout regions"}, fn.SourceSpan)
				}
			}
		}
	}
	sort.SliceStable(mir.SemanticFacts, func(i, j int) bool { return mir.SemanticFacts[i].ID < mir.SemanticFacts[j].ID })
}

func evt1FactStaticInt(expression string) (int, bool) {
	expression = strings.TrimSpace(expression)
	for strings.HasPrefix(expression, "(") && strings.HasSuffix(expression, ")") {
		expression = strings.TrimSpace(expression[1 : len(expression)-1])
	}
	parts := strings.Split(expression, "+")
	total := 0
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "()")
		value, err := strconv.Atoi(part)
		if err != nil {
			return 0, false
		}
		total += value
	}
	return total, len(parts) > 0
}

func evt1ValidateSemanticFacts(facts []MIRSemanticFact) error {
	seen := map[string]bool{}
	for _, fact := range facts {
		if fact.ID == "" || fact.Kind == "" || len(fact.Subjects) == 0 || fact.Origin == "" || (fact.Outcome != FactProven && fact.Outcome != FactDisproven && fact.Outcome != FactUnknown) || seen[fact.ID] {
			return evt1Diagnostic("CV4649", "MIR semantic fact is incomplete or has unstable duplicate identity", fact.SourceSpan)
		}
		seen[fact.ID] = true
		for _, subject := range fact.Subjects {
			if subject.Kind == "" || subject.Name == "" {
				return evt1Diagnostic("CV4649", "MIR semantic fact omits its typed subject identity", fact.SourceSpan)
			}
		}
	}
	return nil
}

func evt1FactOriginForOperation(operation MIROperation) SemanticFactOrigin {
	switch operation.Kind {
	case "layout_bind", "stream_bind", "bind_storage":
		return FactOriginBind
	case "span_from_region", "span_to_readonly", "span_subregion", "span_index":
		return FactOriginSpan
	case "tensor_inline_storage", "tensor_view":
		return FactOriginTensorBacking
	case "region_projection":
		return FactOriginLayout
	default:
		return FactOriginType
	}
}
