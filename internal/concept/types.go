package concept

import (
	"fmt"
	"sort"
	"strings"
)

// CompilerID is the stable identity of the active Concept Stage 0 compiler.
const CompilerID = "concept-evt1-stage0-go"

type TypeKind string

const (
	TypeBuiltin      TypeKind = "builtin"
	TypeEnum         TypeKind = "enum"
	TypeStruct       TypeKind = "struct"
	TypePointer      TypeKind = "pointer"
	TypeArray        TypeKind = "array"
	TypeNDArray      TypeKind = "ndarray"
	TypeLayout       TypeKind = "layout"
	TypeStream       TypeKind = "stream"
	TypeSpan         TypeKind = "span"
	TypeTensor       TypeKind = "tensor"
	TypeConceptParam TypeKind = "concept_param"
	TypeApplied      TypeKind = "applied"
)

type StorageKind string

const (
	StorageArray   StorageKind = "array"
	StorageNDArray StorageKind = "ndarray"
)

type StorageDimension struct {
	Extent     int    `json:"extent,omitempty"`
	Runtime    bool   `json:"runtime,omitempty"`
	Expression string `json:"expression"`
	Expr       Expr   `json:"-"`
}

type Type struct {
	Name            string             `json:"name"`
	Kind            TypeKind           `json:"kind"`
	Ownership       string             `json:"ownership,omitempty"`
	Const           bool               `json:"const,omitempty"`
	Scoped          bool               `json:"scoped,omitempty"`
	Imported        bool               `json:"imported,omitempty"`
	Unsafe          bool               `json:"unsafe,omitempty"`
	PointerTo       *Type              `json:"pointer_to,omitempty"`
	TypeArgs        []Type             `json:"type_args,omitempty"`
	TensorRank      int                `json:"tensor_rank,omitempty"`
	TensorSpelling  string             `json:"-"`
	ArrayElem       *Type              `json:"array_elem,omitempty"`
	ArrayLength     int                `json:"array_length,omitempty"`
	ArrayLengthExpr Expr               `json:"-"`
	StorageKind     StorageKind        `json:"storage_kind,omitempty"`
	Shape           []StorageDimension `json:"shape,omitempty"`
	Contiguous      bool               `json:"contiguous,omitempty"`
	Layout          string             `json:"layout,omitempty"`
	Span            Span               `json:"span"`
}

func (t Type) String() string {
	var parts []string
	if t.Unsafe {
		parts = append(parts, "unsafe")
	}
	if t.Imported {
		parts = append(parts, "imported")
	}
	if t.Scoped {
		parts = append(parts, "scoped")
	}
	if t.Ownership != "" {
		parts = append(parts, t.Ownership)
	}
	if t.Const {
		parts = append(parts, "const")
	}
	base := t.Name
	if t.PointerTo != nil {
		base = t.PointerTo.String() + "*"
	} else if t.ArrayElem != nil {
		if t.StorageKind != "" {
			dimensions := make([]string, 0, len(t.Shape))
			for _, dimension := range t.Shape {
				dimensions = append(dimensions, dimension.String())
			}
			base = fmt.Sprintf("%s<%s>[%s]", t.ArrayElem.String(), t.StorageKind, strings.Join(dimensions, ", "))
		} else {
			base = fmt.Sprintf("%s[%s]", t.ArrayElem.String(), evt1ArrayLengthString(t))
		}
	} else if t.Kind == TypeTensor && len(t.TypeArgs) == 1 {
		base = fmt.Sprintf("tensor<%s, %d>", t.TypeArgs[0].String(), t.TensorRank)
	} else if len(t.TypeArgs) > 0 {
		var args []string
		for _, arg := range t.TypeArgs {
			args = append(args, arg.String())
		}
		base = base + "<" + strings.Join(args, ", ") + ">"
	}
	parts = append(parts, base)
	return strings.Join(parts, " ")
}

func (t Type) Equal(other Type) bool {
	if t.Name != other.Name ||
		t.Kind != other.Kind ||
		t.Ownership != other.Ownership ||
		t.Const != other.Const ||
		t.Scoped != other.Scoped ||
		t.Imported != other.Imported ||
		t.Unsafe != other.Unsafe ||
		len(t.TypeArgs) != len(other.TypeArgs) ||
		t.TensorRank != other.TensorRank ||
		t.StorageKind != other.StorageKind ||
		len(t.Shape) != len(other.Shape) {
		return false
	}
	for i := range t.TypeArgs {
		if !t.TypeArgs[i].Equal(other.TypeArgs[i]) {
			return false
		}
	}
	for i := range t.Shape {
		if t.Shape[i].String() != other.Shape[i].String() {
			return false
		}
	}
	if len(t.Shape) == 0 && t.ArrayLength != other.ArrayLength {
		return false
	}
	if t.PointerTo == nil || other.PointerTo == nil {
		if t.PointerTo != nil || other.PointerTo != nil {
			return false
		}
	} else if !t.PointerTo.Equal(*other.PointerTo) {
		return false
	}
	if t.ArrayElem == nil || other.ArrayElem == nil {
		return t.ArrayElem == nil && other.ArrayElem == nil
	}
	return t.ArrayElem.Equal(*other.ArrayElem)
}

func (d StorageDimension) String() string {
	if d.Expression != "" {
		return d.Expression
	}
	if d.Expr != nil {
		return evt1ExprIdentity(d.Expr)
	}
	return fmt.Sprintf("%d", d.Extent)
}

func (t Type) SameValueType(other Type) bool {
	a := t.valueType()
	b := other.valueType()
	return a.Equal(b)
}

func (t Type) valueType() Type {
	out := t
	out.Ownership = ""
	out.Const = false
	out.Scoped = false
	out.Imported = false
	out.Unsafe = false
	if out.PointerTo != nil {
		base := out.PointerTo.valueType()
		out.PointerTo = &base
	}
	if out.ArrayElem != nil {
		elem := out.ArrayElem.valueType()
		out.ArrayElem = &elem
	}
	return out
}

func (t Type) isBorrow() bool {
	return t.Ownership == "borrow"
}

func (t Type) isReference() bool {
	return t.Ownership == "ref"
}

func (t Type) isOwned() bool {
	return t.Ownership == "owned"
}

func (t Type) isBorrowLike() bool {
	return t.isBorrow() || t.isReference() || t.PointerTo != nil
}

func (t Type) borrowBase() Type {
	if t.PointerTo != nil {
		base := *t.PointerTo
		base.Const = base.Const || t.Const
		return base.valueType()
	}
	return t.valueType()
}

type Field struct {
	Type Type   `json:"type"`
	Name string `json:"name"`
	Span Span   `json:"span"`
}

// LayoutDecl is a zero-allocation semantic description of fixed memory
// geometry. Region offsets and extents are derived during semantic analysis.
type LayoutDecl struct {
	Name      string         `json:"name"`
	Regions   []LayoutRegion `json:"regions"`
	Size      int            `json:"size"`
	Alignment int            `json:"alignment"`
	Span      Span           `json:"span"`
}

type LayoutRegion struct {
	ID             string `json:"id,omitempty"`
	Name           string `json:"name"`
	Type           Type   `json:"type"`
	RequestedAlign int    `json:"requested_alignment,omitempty"`
	ExplicitOffset *int   `json:"explicit_offset,omitempty"`
	Offset         int    `json:"offset,omitempty"`
	ByteExtent     int    `json:"byte_extent,omitempty"`
	Alignment      int    `json:"alignment,omitempty"`
	Span           Span   `json:"span"`
}

type StreamDecl struct {
	Name       string          `json:"name"`
	LayoutName string          `json:"layout_name"`
	Channels   []StreamChannel `json:"channels"`
	Span       Span            `json:"span"`
}

type StreamChannel struct {
	Name       string `json:"name"`
	RegionName string `json:"region_name"`
	RegionID   string `json:"region_id,omitempty"`
	Type       Type   `json:"type,omitempty"`
	Span       Span   `json:"span"`
}

type StructDecl struct {
	Name       string  `json:"name"`
	Immovable  bool    `json:"immovable"`
	Record     bool    `json:"record"`
	Ref        bool    `json:"ref,omitempty"`
	Fields     []Field `json:"fields"`
	Span       Span    `json:"span"`
	RecordSpan Span    `json:"record_span,omitempty"`
}

type VariantDecl struct {
	Name    string  `json:"name"`
	Payload []Field `json:"payload,omitempty"`
	Tag     int     `json:"tag"`
	Span    Span    `json:"span"`
}

type EnumDecl struct {
	Name     string        `json:"name"`
	Variants []VariantDecl `json:"variants"`
	Span     Span          `json:"span"`
}

type Param struct {
	Type Type   `json:"type"`
	Name string `json:"name"`
	Span Span   `json:"span"`
}

type EffectDecl struct {
	Name   string  `json:"name"`
	Params []Param `json:"params,omitempty"`
	Span   Span    `json:"span"`
}

type ActuatorMapping struct {
	EffectName         string  `json:"effect_name"`
	Params             []Param `json:"params,omitempty"`
	ImplementationName string  `json:"implementation_name"`
	ImplementationArgs []Expr  `json:"implementation_args,omitempty"`
	Span               Span    `json:"span"`
}

type ActuatorDecl struct {
	Name          string            `json:"name"`
	AutomataName  string            `json:"automata_name"`
	MechanismType Type              `json:"mechanism_type"`
	MechanismName string            `json:"mechanism_name"`
	ErrorType     Type              `json:"error_type"`
	Mappings      []ActuatorMapping `json:"mappings,omitempty"`
	Span          Span              `json:"span"`
}

type ConceptRequirement interface {
	evt1ConceptRequirement()
	requirementSpan() Span
}

type OperationRequirement struct {
	ReturnType Type    `json:"return_type"`
	Name       string  `json:"name"`
	Params     []Param `json:"params,omitempty"`
	Span       Span    `json:"span"`
}

func (*OperationRequirement) evt1ConceptRequirement() {}
func (r *OperationRequirement) requirementSpan() Span { return r.Span }

type PrerequisiteRequirement struct {
	ConceptName string `json:"concept_name"`
	TypeArg     Type   `json:"type_arg"`
	Span        Span   `json:"span"`
}

func (*PrerequisiteRequirement) evt1ConceptRequirement() {}
func (r *PrerequisiteRequirement) requirementSpan() Span { return r.Span }

type CompilerAnalysisRequirement struct {
	Analysis    string               `json:"analysis"`
	TypeArgs    []Type               `json:"type_args,omitempty"`
	SubjectArgs []SemanticSubjectRef `json:"subject_args,omitempty"`
	Parameters  []int                `json:"parameters,omitempty"`
	Span        Span                 `json:"span"`
}

type SemanticSubjectRef struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

func (*CompilerAnalysisRequirement) evt1ConceptRequirement() {}
func (r *CompilerAnalysisRequirement) requirementSpan() Span { return r.Span }

type ConceptDecl struct {
	Name         string               `json:"name"`
	TypeParam    string               `json:"type_param"`
	Requirements []ConceptRequirement `json:"requirements,omitempty"`
	Span         Span                 `json:"span"`
}

type ConceptAssertion struct {
	ConceptName  string `json:"concept_name"`
	ConcreteType Type   `json:"concrete_type"`
	Span         Span   `json:"span"`
}

type TemplateConstraint struct {
	ConceptName string `json:"concept_name"`
	TypeArg     Type   `json:"type_arg"`
	Span        Span   `json:"span"`
}

type TemplateDecl struct {
	Name          string             `json:"name"`
	TypeParam     string             `json:"type_param"`
	TypeParamSpan Span               `json:"type_param_span"`
	Constraint    TemplateConstraint `json:"constraint"`
	ReturnType    Type               `json:"return_type"`
	Params        []Param            `json:"params,omitempty"`
	Body          *Block             `json:"body,omitempty"`
	Span          Span               `json:"span"`
}

type FunctionDecl struct {
	Comptime   bool    `json:"comptime,omitempty"`
	Name       string  `json:"name"`
	ReturnType Type    `json:"return_type"`
	Params     []Param `json:"params,omitempty"`
	Body       *Block  `json:"body,omitempty"`
	Span       Span    `json:"span"`
}

type ComptimeDecl struct {
	Type  Type   `json:"type"`
	Name  string `json:"name"`
	Value Expr   `json:"value"`
	Span  Span   `json:"span"`
}

type StaticAssert struct {
	Condition Expr `json:"condition"`
	Message   Expr `json:"message,omitempty"`
	Span      Span `json:"span"`
}

type Module struct {
	Path          string             `json:"path"`
	Profile       string             `json:"profile"`
	Imports       []string           `json:"imports,omitempty"`
	Structs       []StructDecl       `json:"structs,omitempty"`
	Layouts       []LayoutDecl       `json:"layouts,omitempty"`
	Streams       []StreamDecl       `json:"streams,omitempty"`
	Enums         []EnumDecl         `json:"enums,omitempty"`
	Effects       []EffectDecl       `json:"effects,omitempty"`
	Actuators     []ActuatorDecl     `json:"actuators,omitempty"`
	Automata      []AutomataDecl     `json:"automata,omitempty"`
	Concepts      []ConceptDecl      `json:"concepts,omitempty"`
	Assertions    []ConceptAssertion `json:"assertions,omitempty"`
	ComptimeDecls []ComptimeDecl     `json:"comptime_decls,omitempty"`
	StaticAsserts []StaticAssert     `json:"static_asserts,omitempty"`
	Templates     []TemplateDecl     `json:"templates,omitempty"`
	Functions     []FunctionDecl     `json:"functions,omitempty"`
	ComptimeFns   []FunctionDecl     `json:"comptime_functions,omitempty"`
}

type Block struct {
	Statements []Statement `json:"statements,omitempty"`
	Span       Span        `json:"span"`
}

type IfStmt struct {
	Condition Expr   `json:"condition"`
	Then      Block  `json:"then"`
	Else      *Block `json:"else,omitempty"`
	Span      Span   `json:"span"`
}

func (*IfStmt) evt1Statement()        {}
func (s *IfStmt) statementSpan() Span { return s.Span }

func (*Block) evt1Statement()        {}
func (s *Block) statementSpan() Span { return s.Span }

type Statement interface {
	evt1Statement()
	statementSpan() Span
}

type VarDecl struct {
	Comptime     bool              `json:"comptime,omitempty"`
	Const        bool              `json:"const,omitempty"`
	Type         Type              `json:"type"`
	Name         string            `json:"name"`
	Value        Expr              `json:"value"`
	InlineTensor *InlineTensorDecl `json:"inline_tensor,omitempty"`
	Span         Span              `json:"span"`
	ConstSpan    Span              `json:"const_span,omitempty"`
}

// InlineTensorDecl retains the source identity of shaped tensor declaration
// sugar while semantic analysis derives its ordinary fixed ndarray backing.
type InlineTensorDecl struct {
	Spelling        string             `json:"spelling"`
	Shape           []StorageDimension `json:"shape"`
	BackingType     Type               `json:"backing_type"`
	BackingID       string             `json:"backing_id"`
	InitializerKind string             `json:"initializer_kind"`
	Facts           *TensorViewFacts   `json:"facts,omitempty"`
}

func (*VarDecl) evt1Statement()        {}
func (s *VarDecl) statementSpan() Span { return s.Span }

type EffectsDecl struct {
	AutomataName string `json:"automata_name"`
	Name         string `json:"name"`
	Span         Span   `json:"span"`
}

func (*EffectsDecl) evt1Statement()        {}
func (s *EffectsDecl) statementSpan() Span { return s.Span }

type ActuatorLocalDecl struct {
	ActuatorName string `json:"actuator_name"`
	Name         string `json:"name"`
	Mechanism    Expr   `json:"mechanism"`
	Span         Span   `json:"span"`
}

func (*ActuatorLocalDecl) evt1Statement()        {}
func (s *ActuatorLocalDecl) statementSpan() Span { return s.Span }

type InstanceDecl struct {
	AutomataName string `json:"automata_name"`
	Name         string `json:"name"`
	Context      Expr   `json:"context,omitempty"`
	Span         Span   `json:"span"`
}

func (*InstanceDecl) evt1Statement()        {}
func (s *InstanceDecl) statementSpan() Span { return s.Span }

type ActuationDecl struct {
	ActuatorName string `json:"actuator_name"`
	Name         string `json:"name"`
	BatchName    string `json:"batch_name"`
	ExecutorName string `json:"executor_name"`
	Span         Span   `json:"span"`
}

func (*ActuationDecl) evt1Statement()        {}
func (s *ActuationDecl) statementSpan() Span { return s.Span }

type AssignStmt struct {
	Target Expr            `json:"target"`
	Value  Expr            `json:"value"`
	Tensor *TensorSemantic `json:"tensor,omitempty"`
	Span   Span            `json:"span"`
}

func (*AssignStmt) evt1Statement()        {}
func (s *AssignStmt) statementSpan() Span { return s.Span }

type ReturnStmt struct {
	Value Expr `json:"value,omitempty"`
	Span  Span `json:"span"`
}

func (*ReturnStmt) evt1Statement()        {}
func (s *ReturnStmt) statementSpan() Span { return s.Span }

type ExprStmt struct {
	Value Expr `json:"value"`
	Span  Span `json:"span"`
}

// AssertStmt is runtime assertion sugar. It lowers to the same terminal panic
// primitive used by explicit failure escalation; it is not a testing API.
type AssertStmt struct {
	Condition Expr `json:"condition"`
	Reason    Expr `json:"reason,omitempty"`
	Span      Span `json:"span"`
}

func (*AssertStmt) evt1Statement()        {}
func (s *AssertStmt) statementSpan() Span { return s.Span }

// TryStmt is a lexical propagation boundary. Except arms are selected by the
// statically known Result error type of each ? expression in Body.
type TryStmt struct {
	Body   Block       `json:"body"`
	Except []ExceptArm `json:"except"`
	Span   Span        `json:"span"`
}

func (*TryStmt) evt1Statement()        {}
func (s *TryStmt) statementSpan() Span { return s.Span }

type ExceptArm struct {
	ErrorType Type   `json:"error_type"`
	Binding   string `json:"binding"`
	Body      Block  `json:"body"`
	Span      Span   `json:"span"`
}

func (*ExprStmt) evt1Statement()        {}
func (s *ExprStmt) statementSpan() Span { return s.Span }

type StaticAssertStmt struct {
	Condition Expr `json:"condition"`
	Message   Expr `json:"message,omitempty"`
	Span      Span `json:"span"`
}

func (*StaticAssertStmt) evt1Statement()        {}
func (s *StaticAssertStmt) statementSpan() Span { return s.Span }

type MatchStmt struct {
	Subject Expr           `json:"subject"`
	Arms    []StatementArm `json:"arms"`
	Span    Span           `json:"span"`
}

func (*MatchStmt) evt1Statement()        {}
func (s *MatchStmt) statementSpan() Span { return s.Span }

type StatementArm struct {
	Pattern Pattern `json:"pattern"`
	Block   Block   `json:"block"`
	Span    Span    `json:"span"`
}

type WhileStmt struct {
	Condition Expr  `json:"condition"`
	Bound     Expr  `json:"bound,omitempty"`
	Body      Block `json:"body"`
	Span      Span  `json:"span"`
}

func (*WhileStmt) evt1Statement()        {}
func (s *WhileStmt) statementSpan() Span { return s.Span }

type Pattern struct {
	EnumName    string   `json:"enum_name"`
	VariantName string   `json:"variant_name"`
	Bindings    []string `json:"bindings,omitempty"`
	Span        Span     `json:"span"`
}

type Expr interface {
	evt1Expr()
	exprSpan() Span
}

type NameExpr struct {
	Name string `json:"name"`
	Span Span   `json:"span"`
}

func (*NameExpr) evt1Expr()        {}
func (e *NameExpr) exprSpan() Span { return e.Span }

type IntLiteral struct {
	Value int  `json:"value"`
	Span  Span `json:"span"`
}

func (*IntLiteral) evt1Expr()        {}
func (e *IntLiteral) exprSpan() Span { return e.Span }

type FloatLiteral struct {
	Value float64 `json:"value"`
	Span  Span    `json:"span"`
}

func (*FloatLiteral) evt1Expr()        {}
func (e *FloatLiteral) exprSpan() Span { return e.Span }

type StringLiteral struct {
	Value string `json:"value"`
	Span  Span   `json:"span"`
}

func (*StringLiteral) evt1Expr()        {}
func (e *StringLiteral) exprSpan() Span { return e.Span }

type BoolLiteral struct {
	Value bool `json:"value"`
	Span  Span `json:"span"`
}

func (*BoolLiteral) evt1Expr()        {}
func (e *BoolLiteral) exprSpan() Span { return e.Span }

type FieldExpr struct {
	Receiver        Expr   `json:"receiver"`
	Field           string `json:"field"`
	RegionID        string `json:"region_id,omitempty"`
	LayoutName      string `json:"layout_name,omitempty"`
	RegionOffset    int    `json:"region_offset,omitempty"`
	RegionExtent    int    `json:"region_extent,omitempty"`
	RegionAlignment int    `json:"region_alignment,omitempty"`
	Span            Span   `json:"span"`
}

func (*FieldExpr) evt1Expr()        {}
func (e *FieldExpr) exprSpan() Span { return e.Span }

type CallExpr struct {
	Callee               string           `json:"callee"`
	Args                 []Expr           `json:"args,omitempty"`
	Intrinsic            string           `json:"intrinsic,omitempty"`
	SpanElementType      *Type            `json:"span_element_type,omitempty"`
	RegionID             string           `json:"region_id,omitempty"`
	BackingByteOffset    int              `json:"backing_byte_offset,omitempty"`
	BaseOffsetExpression string           `json:"base_offset_expression,omitempty"`
	LengthExpression     string           `json:"length_expression,omitempty"`
	ByteExtentExpression string           `json:"byte_extent_expression,omitempty"`
	Alignment            int              `json:"alignment,omitempty"`
	Mutability           string           `json:"mutability,omitempty"`
	ProvenanceKind       string           `json:"provenance_kind,omitempty"`
	ProvenanceScoped     bool             `json:"provenance_scoped,omitempty"`
	RuntimeBounds        bool             `json:"runtime_bounds,omitempty"`
	TensorFacts          *TensorViewFacts `json:"tensor_facts,omitempty"`
	Span                 Span             `json:"span"`
}

func (*CallExpr) evt1Expr()        {}
func (e *CallExpr) exprSpan() Span { return e.Span }

type DispatchExpr struct {
	InstanceName string `json:"instance_name"`
	Signal       Expr   `json:"signal"`
	BatchName    string `json:"batch_name,omitempty"`
	Span         Span   `json:"span"`
}

func (*DispatchExpr) evt1Expr()        {}
func (e *DispatchExpr) exprSpan() Span { return e.Span }

type TemplateCallExpr struct {
	Callee  string `json:"callee"`
	TypeArg Type   `json:"type_arg"`
	Args    []Expr `json:"args,omitempty"`
	Span    Span   `json:"span"`
}

func (*TemplateCallExpr) evt1Expr()        {}
func (e *TemplateCallExpr) exprSpan() Span { return e.Span }

type BinaryExpr struct {
	Op     string          `json:"op"`
	Left   Expr            `json:"left"`
	Right  Expr            `json:"right"`
	Tensor *TensorSemantic `json:"tensor,omitempty"`
	Span   Span            `json:"span"`
}

func (*BinaryExpr) evt1Expr()        {}
func (e *BinaryExpr) exprSpan() Span { return e.Span }

type UnaryExpr struct {
	Op    string `json:"op"`
	Value Expr   `json:"value"`
	Span  Span   `json:"span"`
}

func (*UnaryExpr) evt1Expr()        {}
func (e *UnaryExpr) exprSpan() Span { return e.Span }

type MoveExpr struct {
	Value Expr `json:"value"`
	Span  Span `json:"span"`
}

func (*MoveExpr) evt1Expr()        {}
func (e *MoveExpr) exprSpan() Span { return e.Span }

type RefExpr struct {
	Value Expr `json:"value"`
	Const bool `json:"const,omitempty"`
	Span  Span `json:"span"`
}

func (*RefExpr) evt1Expr()        {}
func (e *RefExpr) exprSpan() Span { return e.Span }

type BindExpr struct {
	Source           Expr   `json:"source"`
	BindKind         string `json:"bind_kind,omitempty"`
	LayoutName       string `json:"layout_name,omitempty"`
	TargetType       Type   `json:"target_type,omitempty"`
	SourceType       Type   `json:"source_type,omitempty"`
	RuntimeCheck     bool   `json:"runtime_check,omitempty"`
	ProvenanceKind   string `json:"provenance_kind,omitempty"`
	ProvenanceScoped bool   `json:"provenance_scoped,omitempty"`
	Span             Span   `json:"span"`
}

func (*BindExpr) evt1Expr()        {}
func (e *BindExpr) exprSpan() Span { return e.Span }

type ConstructExpr struct {
	EnumName     string `json:"enum_name"`
	VariantName  string `json:"variant_name"`
	Args         []Expr `json:"args,omitempty"`
	ResolvedType Type   `json:"resolved_type,omitempty"`
	Span         Span   `json:"span"`
}

func (*ConstructExpr) evt1Expr()        {}
func (e *ConstructExpr) exprSpan() Span { return e.Span }

type StructConstructExpr struct {
	StructName string `json:"struct_name"`
	Args       []Expr `json:"args,omitempty"`
	Span       Span   `json:"span"`
}

func (*StructConstructExpr) evt1Expr()        {}
func (e *StructConstructExpr) exprSpan() Span { return e.Span }

type FieldUpdate struct {
	Name     string `json:"name"`
	NameSpan Span   `json:"name_span"`
	Value    Expr   `json:"value"`
}

type WithExpr struct {
	Base    Expr          `json:"base"`
	Updates []FieldUpdate `json:"updates"`
	Span    Span          `json:"span"`
}

func (*WithExpr) evt1Expr()        {}
func (e *WithExpr) exprSpan() Span { return e.Span }

type ArrayLiteralExpr struct {
	Elements []Expr `json:"elements,omitempty"`
	Span     Span   `json:"span"`
}

func (*ArrayLiteralExpr) evt1Expr()        {}
func (e *ArrayLiteralExpr) exprSpan() Span { return e.Span }

type IndexExpr struct {
	Base             Expr     `json:"base"`
	Index            Expr     `json:"index"` // first index retained for the legacy rank-1 evaluator
	Indices          []Expr   `json:"indices,omitempty"`
	SpanIndex        bool     `json:"span_index,omitempty"`
	SpanElementType  *Type    `json:"span_element_type,omitempty"`
	RegionID         string   `json:"region_id,omitempty"`
	LengthExpression string   `json:"length_expression,omitempty"`
	SpanMutability   string   `json:"span_mutability,omitempty"`
	SpanAlignment    int      `json:"span_alignment,omitempty"`
	ProvenanceKind   string   `json:"provenance_kind,omitempty"`
	ProvenanceScoped bool     `json:"provenance_scoped,omitempty"`
	TensorIndex      bool     `json:"tensor_index,omitempty"`
	SymbolicIndices  []string `json:"symbolic_indices,omitempty"`
	Span             Span     `json:"span"`
}

func (*IndexExpr) evt1Expr()        {}
func (e *IndexExpr) exprSpan() Span { return e.Span }

type MatchExpr struct {
	Subject Expr      `json:"subject"`
	Arms    []ExprArm `json:"arms"`
	Span    Span      `json:"span"`
}

func (*MatchExpr) evt1Expr()        {}
func (e *MatchExpr) exprSpan() Span { return e.Span }

type ExprArm struct {
	Pattern Pattern `json:"pattern"`
	Value   Expr    `json:"value"`
	Span    Span    `json:"span"`
}

type IfExpr struct {
	Condition Expr `json:"condition"`
	Then      Expr `json:"then"`
	Else      Expr `json:"else"`
	Span      Span `json:"span"`
}

func (*IfExpr) evt1Expr()        {}
func (e *IfExpr) exprSpan() Span { return e.Span }

type ParenExpr struct {
	Value Expr `json:"value"`
	Span  Span `json:"span"`
}

type FailureExpr struct {
	Op           string `json:"op"` // ? propagates; ! escalates.
	Value        Expr   `json:"value"`
	ResolvedType Type   `json:"resolved_type,omitempty"`
	Span         Span   `json:"span"`
}

func (*FailureExpr) evt1Expr()        {}
func (e *FailureExpr) exprSpan() Span { return e.Span }

func (*ParenExpr) evt1Expr()        {}
func (e *ParenExpr) exprSpan() Span { return e.Span }

type MIREffect struct {
	Name       string    `json:"name"`
	Params     []MIRName `json:"params,omitempty"`
	SourceSpan Span      `json:"source_span"`
}

type MIRActuator struct {
	Name          string               `json:"name"`
	AutomataName  string               `json:"automata_name"`
	MechanismName string               `json:"mechanism_name"`
	MechanismType Type                 `json:"mechanism_type"`
	ErrorType     Type                 `json:"error_type"`
	Identity      string               `json:"identity"`
	ResultType    string               `json:"result_type"`
	FailureSlot   string               `json:"failure_slot"`
	Mappings      []MIRActuatorMapping `json:"mappings,omitempty"`
	SourceSpan    Span                 `json:"source_span"`
}

type MIRActuatorMapping struct {
	EffectName         string   `json:"effect_name"`
	ImplementationName string   `json:"implementation_name"`
	ImplementationArgs []string `json:"implementation_args,omitempty"`
	SourceSpan         Span     `json:"source_span"`
}

type MIR struct {
	Schema         string             `json:"schema"`
	Module         string             `json:"module"`
	Profile        string             `json:"profile"`
	Structs        []MIRStruct        `json:"structs,omitempty"`
	Enums          []MIREnum          `json:"enums,omitempty"`
	Effects        []MIREffect        `json:"effects,omitempty"`
	Actuators      []MIRActuator      `json:"actuators,omitempty"`
	Automata       []MIRAutomata      `json:"automata,omitempty"`
	Concepts       []MIRConcept       `json:"concepts,omitempty"`
	Assertions     []MIRAssertion     `json:"assertions,omitempty"`
	ComptimeDecls  []MIRComptimeDecl  `json:"comptime_decls,omitempty"`
	StaticAsserts  []MIRStaticAssert  `json:"static_asserts,omitempty"`
	Templates      []MIRTemplate      `json:"templates,omitempty"`
	Instances      []MIRInstance      `json:"instances,omitempty"`
	Functions      []MIRFunction      `json:"functions"`
	ComptimeFns    []MIRFunction      `json:"comptime_functions,omitempty"`
	SemanticProofs []MIRSemanticProof `json:"semantic_proofs,omitempty"`
	SemanticFacts  []MIRSemanticFact  `json:"semantic_facts,omitempty"`
	StorageTypes   []MIRStorageType   `json:"storage_types,omitempty"`
	Layouts        []MIRLayout        `json:"layouts,omitempty"`
	Streams        []MIRStream        `json:"streams,omitempty"`
}

type MIRLayout struct {
	Name       string            `json:"name"`
	Size       int               `json:"size"`
	Alignment  int               `json:"alignment"`
	Regions    []MIRLayoutRegion `json:"regions"`
	SourceSpan Span              `json:"source_span"`
}

type MIRLayoutRegion struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Type         Type               `json:"type"`
	Offset       int                `json:"offset"`
	ByteExtent   int                `json:"byte_extent"`
	Alignment    int                `json:"alignment"`
	Shape        []StorageDimension `json:"shape,omitempty"`
	DisjointWith []string           `json:"disjoint_with,omitempty"`
}

type MIRStream struct {
	Name        string             `json:"name"`
	LayoutName  string             `json:"layout_name"`
	Channels    []MIRStreamChannel `json:"channels"`
	ZeroStorage bool               `json:"zero_storage"`
	SourceSpan  Span               `json:"source_span"`
}

type MIRStreamChannel struct {
	Name     string `json:"name"`
	RegionID string `json:"region_id"`
	Type     Type   `json:"type"`
}

type MIRStorageType struct {
	Type        Type               `json:"type"`
	ElementType Type               `json:"element_type"`
	StorageKind StorageKind        `json:"storage_kind"`
	Rank        int                `json:"rank"`
	Shape       []StorageDimension `json:"shape"`
	Contiguous  bool               `json:"contiguous"`
	Layout      string             `json:"layout"`
	Ownership   string             `json:"ownership"`
}

type MIRStruct struct {
	Name       string    `json:"name"`
	CName      string    `json:"c_name"`
	Immovable  bool      `json:"immovable"`
	Record     bool      `json:"record,omitempty"`
	Ref        bool      `json:"ref,omitempty"`
	Copyable   bool      `json:"copyable"`
	Movable    bool      `json:"movable"`
	HasDrop    bool      `json:"has_drop"`
	Fields     []MIRName `json:"fields,omitempty"`
	SourceSpan Span      `json:"source_span"`
}

type MIREnum struct {
	Name       string       `json:"name"`
	CName      string       `json:"c_name"`
	SourceSpan Span         `json:"source_span"`
	Variants   []MIRVariant `json:"variants"`
}

type MIRVariant struct {
	Name       string    `json:"name"`
	TagName    string    `json:"tag_name"`
	Payload    []MIRName `json:"payload,omitempty"`
	Tag        int       `json:"tag"`
	SourceSpan Span      `json:"source_span"`
}

type MIRName struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

type MIRConcept struct {
	Name         string                  `json:"name"`
	TypeParam    string                  `json:"type_param"`
	Requirements []MIRConceptRequirement `json:"requirements,omitempty"`
	SourceSpan   Span                    `json:"source_span"`
}

type MIRConceptRequirement struct {
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	ReturnType *Type     `json:"return_type,omitempty"`
	Params     []MIRName `json:"params,omitempty"`
	Detail     string    `json:"detail,omitempty"`
	SourceSpan Span      `json:"source_span"`
}

type MIRAssertion struct {
	ConceptName  string `json:"concept_name"`
	ConcreteType Type   `json:"concrete_type"`
	Satisfied    bool   `json:"satisfied"`
	SourceSpan   Span   `json:"source_span"`
}

type MIRComptimeDecl struct {
	Name       string `json:"name"`
	Type       Type   `json:"type"`
	Value      string `json:"value"`
	SourceSpan Span   `json:"source_span"`
}

type MIRStaticAssert struct {
	Condition  string `json:"condition"`
	Message    string `json:"message,omitempty"`
	Satisfied  bool   `json:"satisfied"`
	SourceSpan Span   `json:"source_span"`
}

type MIRTemplateConstraint struct {
	ConceptName string `json:"concept_name"`
	TypeParam   string `json:"type_param"`
	SourceSpan  Span   `json:"source_span"`
}

type MIRClosureEntry struct {
	ConceptName string   `json:"concept_name"`
	Path        []string `json:"path,omitempty"`
}

type MIRRequirementBinding struct {
	RequirementID string   `json:"requirement_id"`
	ConceptName   string   `json:"concept_name"`
	Name          string   `json:"name"`
	Signature     string   `json:"signature"`
	Path          []string `json:"path,omitempty"`
}

type MIRTemplate struct {
	Name         string                  `json:"name"`
	TypeParam    string                  `json:"type_param"`
	Constraint   MIRTemplateConstraint   `json:"constraint"`
	Closure      []MIRClosureEntry       `json:"closure,omitempty"`
	Requirements []MIRRequirementBinding `json:"requirements,omitempty"`
	ReturnType   Type                    `json:"return_type"`
	Params       []MIRName               `json:"params,omitempty"`
	Operations   []MIROperation          `json:"operations,omitempty"`
	SourceSpan   Span                    `json:"source_span"`
}

type MIRInstance struct {
	ID                  string                  `json:"id"`
	TemplateName        string                  `json:"template_name"`
	ConcreteType        Type                    `json:"concrete_type"`
	GeneratedSymbol     string                  `json:"generated_symbol"`
	ConstraintConcept   string                  `json:"constraint_concept"`
	Closure             []MIRClosureEntry       `json:"closure,omitempty"`
	RequirementBindings []MIRRequirementBinding `json:"requirement_bindings,omitempty"`
	ReturnType          Type                    `json:"return_type"`
	Params              []MIRName               `json:"params,omitempty"`
	Operations          []MIROperation          `json:"operations,omitempty"`
	InvocationSpans     []Span                  `json:"invocation_spans,omitempty"`
	SourceSpan          Span                    `json:"source_span"`
}

type MIRFunction struct {
	Name             string                      `json:"name"`
	ReturnType       Type                        `json:"return_type"`
	Params           []MIRName                   `json:"params,omitempty"`
	ResultProvenance *MIRResultProvenanceSummary `json:"result_provenance,omitempty"`
	Operations       []MIROperation              `json:"operations"`
	TensorOperations []MIRTensorOperation        `json:"tensor_operations,omitempty"`
	Cleanups         []MIRCleanup                `json:"cleanups,omitempty"`
	SourceSpan       Span                        `json:"source_span"`
}

type MIRSemanticProof struct {
	ID              string                `json:"id,omitempty"`
	Concept         string                `json:"concept"`
	Analysis        string                `json:"analysis"`
	ConcreteType    string                `json:"concrete_type"`
	Subjects        []MIRSemanticSubject  `json:"subjects,omitempty"`
	Satisfied       bool                  `json:"satisfied"`
	Outcome         string                `json:"outcome,omitempty"`
	FactKind        SemanticFactKind      `json:"fact_kind,omitempty"`
	Parameters      []int                 `json:"parameters,omitempty"`
	Origin          SemanticFactOrigin    `json:"origin,omitempty"`
	RegionIDs       []string              `json:"supporting_region_ids,omitempty"`
	Evidence        *SemanticFactEvidence `json:"evidence,omitempty"`
	ProvenanceFacts []string              `json:"provenance_facts,omitempty"`
	SourceSpan      Span                  `json:"source_span"`
}

type MIRSemanticSubject struct {
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	Function       string `json:"function,omitempty"`
	ParameterIndex *int   `json:"parameter_index,omitempty"`
	Type           string `json:"type,omitempty"`
}

type MIRResultProvenanceSummary struct {
	Kind             string `json:"kind"`
	ParameterIndices []int  `json:"parameter_indices,omitempty"`
}

type MIRCleanup struct {
	Owner        string `json:"owner"`
	Type         string `json:"type"`
	DropFunction string `json:"drop_function"`
	State        string `json:"state"`
	Order        int    `json:"order"`
}

type MIROperation struct {
	ID                   string             `json:"id"`
	Kind                 string             `json:"kind"`
	Type                 string             `json:"type,omitempty"`
	Detail               string             `json:"detail,omitempty"`
	SourceStorageKind    StorageKind        `json:"source_storage_kind,omitempty"`
	TargetStorageKind    StorageKind        `json:"target_storage_kind,omitempty"`
	TargetRank           int                `json:"target_rank,omitempty"`
	TargetShape          []StorageDimension `json:"target_shape,omitempty"`
	CountCheck           string             `json:"count_check,omitempty"`
	Mutability           string             `json:"mutability,omitempty"`
	Provenance           string             `json:"provenance,omitempty"`
	NoCopy               bool               `json:"no_copy,omitempty"`
	NoAllocation         bool               `json:"no_allocation,omitempty"`
	NoOwnershipTransfer  bool               `json:"no_ownership_transfer,omitempty"`
	LayoutName           string             `json:"layout_name,omitempty"`
	RegionID             string             `json:"region_id,omitempty"`
	Offset               int                `json:"offset,omitempty"`
	ByteExtent           int                `json:"byte_extent,omitempty"`
	Alignment            int                `json:"alignment,omitempty"`
	SameBackingRegion    bool               `json:"same_backing_region,omitempty"`
	ElementType          *Type              `json:"element_type,omitempty"`
	BaseOffset           string             `json:"base_offset,omitempty"`
	Length               string             `json:"length,omitempty"`
	ByteExtentExpression string             `json:"byte_extent_expression,omitempty"`
	TensorBackingKind    TensorBackingKind  `json:"tensor_backing_kind,omitempty"`
	Contiguous           bool               `json:"contiguous,omitempty"`
	BoundsCheck          string             `json:"bounds_check,omitempty"`
	SourceSpan           Span               `json:"source_span"`
}

type semanticEnv struct {
	profile           *ProfileDefinition
	enums             map[string]EnumDecl
	structs           map[string]StructDecl
	layouts           map[string]LayoutDecl
	streams           map[string]StreamDecl
	effects           map[string]EffectDecl
	effectOrder       []string
	actuators         map[string]ActuatorDecl
	actuatorInfo      map[string]*evt1ActuatorInfo
	automata          map[string]AutomataDecl
	automataInfo      map[string]*evt1AutomataInfo
	functions         map[string][]FunctionDecl
	comptimeFunctions map[string]FunctionDecl
	templates         map[string]TemplateDecl
	concepts          map[string]ConceptDecl
	comptimeDecls     map[string]ComptimeDecl
	comptimeValues    map[string]Value
	fieldSets         map[string]map[string]Type
	escapedArmBinding map[string]Span
	copyableCache     map[string]bool
	templateInfos     map[string]*evt1TemplateInfo
	templateInstances map[string]*evt1TemplateInstance
	semanticProofs    []MIRSemanticProof
	resultProvenance  map[string]evt1ResultProvenanceSummary
}

const evt1AutomataDispatchOutcomeTypeName = "AutomataDispatchOutcome"

func evt1BuiltinAutomataDispatchOutcomeEnum() EnumDecl {
	return EnumDecl{
		Name: evt1AutomataDispatchOutcomeTypeName,
		Variants: []VariantDecl{
			{Name: "Transitioned", Tag: 0},
			{Name: "Unhandled", Tag: 1},
			{Name: "Ambiguous", Tag: 2},
			{Name: "Finished", Tag: 3},
			{Name: "AlreadyFinished", Tag: 4},
			{Name: "EffectBatchOccupied", Tag: 5},
		},
	}
}

func newSemanticEnv(profile *ProfileDefinition) *semanticEnv {
	enums := map[string]EnumDecl{}
	fieldSets := map[string]map[string]Type{}
	for _, enumDecl := range profile.BuiltinEnums {
		enums[enumDecl.Name] = enumDecl
	}
	for name, builtin := range profile.BuiltinTypes {
		if len(builtin.Fields) != 0 {
			fieldSets[name] = builtin.Fields
		}
	}
	return &semanticEnv{
		profile:           profile,
		enums:             enums,
		structs:           map[string]StructDecl{},
		layouts:           map[string]LayoutDecl{},
		streams:           map[string]StreamDecl{},
		effects:           map[string]EffectDecl{},
		effectOrder:       nil,
		actuators:         map[string]ActuatorDecl{},
		actuatorInfo:      map[string]*evt1ActuatorInfo{},
		automata:          map[string]AutomataDecl{},
		automataInfo:      map[string]*evt1AutomataInfo{},
		functions:         map[string][]FunctionDecl{},
		comptimeFunctions: map[string]FunctionDecl{},
		templates:         map[string]TemplateDecl{},
		concepts:          map[string]ConceptDecl{},
		comptimeDecls:     map[string]ComptimeDecl{},
		comptimeValues:    map[string]Value{},
		fieldSets:         fieldSets,
		escapedArmBinding: map[string]Span{},
		copyableCache:     map[string]bool{},
		templateInfos:     map[string]*evt1TemplateInfo{},
		templateInstances: map[string]*evt1TemplateInstance{},
		semanticProofs:    nil,
		resultProvenance:  map[string]evt1ResultProvenanceSummary{},
	}
}

type evt1TemplateRequirement struct {
	ID        string
	Path      []string
	Concept   string
	Operation OperationRequirement
}

type evt1TemplateClosureEntry struct {
	Concept string
	Path    []string
}

type evt1TemplateCallBinding struct {
	CallSpan    Span
	Requirement evt1TemplateRequirement
}

type evt1TemplateInfo struct {
	Decl         TemplateDecl
	Closure      []evt1TemplateClosureEntry
	Requirements []evt1TemplateRequirement
	CallBindings map[string]evt1TemplateCallBinding
}

type evt1InstanceRequirementBinding struct {
	Requirement evt1TemplateRequirement
	Function    FunctionDecl
}

type evt1TemplateInstance struct {
	Key                 string
	TemplateName        string
	ConcreteType        Type
	TypeIdentity        string
	GeneratedSymbol     string
	ConstraintConcept   string
	Closure             []evt1TemplateClosureEntry
	RequirementBindings []evt1InstanceRequirementBinding
	Function            FunctionDecl
	InvocationSpans     []Span
	SourceSpan          Span
}

type ValueKind string

const (
	ValueInt    ValueKind = "int"
	ValueBool   ValueKind = "bool"
	ValueString ValueKind = "string"
	ValueStruct ValueKind = "struct"
	ValueEnum   ValueKind = "enum"
	ValueArray  ValueKind = "array"
)

type Value struct {
	Kind        ValueKind
	Type        Type
	IntValue    int
	BoolValue   bool
	StringValue string
	StructName  string
	Fields      map[string]Value
	EnumName    string
	Variant     string
	Payload     []Value
	Elements    []Value
}

func (v Value) Render() string {
	switch v.Kind {
	case ValueInt:
		return fmt.Sprintf("%d", v.IntValue)
	case ValueBool:
		if v.BoolValue {
			return "true"
		}
		return "false"
	case ValueString:
		return fmt.Sprintf("%q", v.StringValue)
	case ValueStruct:
		structDeclName := v.StructName
		if structDeclName == "" {
			structDeclName = v.Type.Name
		}
		var parts []string
		for _, field := range sortedValueKeys(v.Fields) {
			parts = append(parts, fmt.Sprintf("%s=%s", field, v.Fields[field].Render()))
		}
		return structDeclName + "{" + strings.Join(parts, ", ") + "}"
	case ValueEnum:
		if len(v.Payload) == 0 {
			return v.EnumName + "::" + v.Variant
		}
		var parts []string
		for _, entry := range v.Payload {
			parts = append(parts, entry.Render())
		}
		return v.EnumName + "::" + v.Variant + "(" + strings.Join(parts, ", ") + ")"
	case ValueArray:
		var parts []string
		for _, entry := range v.Elements {
			parts = append(parts, entry.Render())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return "<invalid>"
	}
}

func evt1ArrayLengthString(t Type) string {
	if t.ArrayElem == nil {
		return ""
	}
	if t.ArrayLengthExpr != nil {
		return "?"
	}
	return fmt.Sprintf("%d", t.ArrayLength)
}

func sortedValueKeys(m map[string]Value) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func evt1Diagnostic(code, message string, span Span) error {
	return Diagnostic{Code: code, Message: message, Span: span}
}

func evt1MissingVariants(enumDecl EnumDecl, seen map[string]bool) []string {
	var missing []string
	for _, variant := range enumDecl.Variants {
		key := enumDecl.Name + "::" + variant.Name
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	return missing
}

func evt1LookupVariant(enumDecl EnumDecl, name string) (VariantDecl, bool) {
	for _, variant := range enumDecl.Variants {
		if variant.Name == name {
			return variant, true
		}
	}
	return VariantDecl{}, false
}

func evt1CName(name string) string {
	var out []byte
	lastLower := false
	for i := 0; i < len(name); i++ {
		c := name[i]
		isUpper := c >= 'A' && c <= 'Z'
		isLower := c >= 'a' && c <= 'z'
		if i > 0 && isUpper && lastLower {
			out = append(out, '_')
		}
		if isUpper {
			out = append(out, c+'a'-'A')
		} else {
			out = append(out, c)
		}
		lastLower = isLower
	}
	return "concept_" + string(out)
}

func evt1TagName(enumName, variantName string) string {
	return strings.ToUpper(evt1CName(enumName) + "_" + evt1CName(variantName)[len("concept_"):])
}

func evt1PayloadFieldName(variantName string) string {
	return evt1CName(variantName)[len("concept_"):]
}

func evt1Require(cond bool, code, message string, span Span) error {
	if cond {
		return nil
	}
	return evt1Diagnostic(code, message, span)
}

func evt1TypeNameOrDie(t Type) string {
	return t.String()
}

func evt1Unexpected(expr Expr) string {
	return fmt.Sprintf("%T", expr)
}
