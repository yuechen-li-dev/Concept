package concept

import "testing"

func TestStructuralTypeSubstitutionTraversesEveryCompositeChild(t *testing.T) {
	parameter := Type{Name: "T", Kind: TypeConceptParam}
	errorParameter := Type{Name: "E", Kind: TypeConceptParam}
	callableResult := Type{Name: "Result", Kind: TypeApplied, TypeArgs: []Type{
		{Name: "Owner", Kind: TypeApplied, TypeArgs: []Type{{Name: "Storage", Kind: TypeTypedStorage, TypeArgs: []Type{parameter}}}},
		errorParameter,
	}}
	typeUnderTest := Type{
		Name:           "callback",
		Kind:           TypeCallback,
		Ownership:      "ref",
		Const:          true,
		Scoped:         true,
		CallableParams: []Type{{Name: "TArray", Kind: TypeArray, ArrayElem: &parameter, Shape: []StorageDimension{{Extent: 4}}}},
		CallableResult: &callableResult,
	}

	got := evt1SubstituteType(typeUnderTest, "T", Type{Name: "Widget", Kind: TypeStruct})
	got = evt1SubstituteType(got, "E", Type{Name: "Error", Kind: TypeEnum})
	if got.Ownership != "ref" || !got.Const || !got.Scoped {
		t.Fatalf("qualifiers changed during substitution: %#v", got)
	}
	if got.CallableParams[0].ArrayElem.Name != "Widget" {
		t.Fatalf("array element remained open: %s", got.CallableParams[0].ArrayElem)
	}
	want := "Result<Owner<Storage<Widget>>, Error>"
	if got.CallableResult.String() != want {
		t.Fatalf("nested callable result = %s, want %s", got.CallableResult.String(), want)
	}
	if evt1TypeContainsConceptParameter(got) {
		t.Fatalf("closed structural type retains a template parameter: %s", got.String())
	}
}

func TestStructuralTypeSubstitutionHandlesTypeAndValueParameters(t *testing.T) {
	parameter := Type{Name: "T", Kind: TypeConceptParam}
	array := Type{
		Name:      "T[]",
		Kind:      TypeArray,
		ArrayElem: &parameter,
		Shape:     []StorageDimension{{Expr: &NameExpr{Name: "N"}, Expression: "N"}},
		TypeArgs:  []Type{{Name: "N", Kind: TypeTemplateValue}},
	}
	array = evt1SubstituteType(array, "T", Type{Name: "Widget", Kind: TypeStruct})
	params := []GenericParameter{{Name: "T", Kind: "type"}, {Name: "N", Kind: "value"}}
	args := []Type{{Name: "Widget", Kind: TypeStruct}, {Name: "128", Kind: TypeTemplateValue}}
	array = evt1SubstituteGenericValueExtents(array, params, args)
	if array.ArrayElem.Name != "Widget" || array.TypeArgs[0].Name != "128" {
		t.Fatalf("mixed substitution did not close array: %#v", array)
	}
	if extent, ok := array.Shape[0].Expr.(*IntLiteral); !ok || extent.Value != 128 {
		t.Fatalf("array extent was not structurally substituted: %#v", array.Shape[0])
	}
}

func TestGenericValueSubstitutionClosesNestedApplicationArgument(t *testing.T) {
	nested := Type{Name: "Inner", Kind: TypeApplied, TypeArgs: []Type{
		{Name: "T", Kind: TypeBuiltin},
		{Name: "N", Kind: TypeBuiltin},
	}}
	params := []GenericParameter{{Name: "T", Kind: "type"}, {Name: "N", Kind: "value"}}
	args := []Type{{Name: "Widget", Kind: TypeStruct}, {Name: "32", Kind: TypeTemplateValue}}
	nested = evt1SubstituteType(nested, "T", args[0])
	nested = evt1SubstituteGenericValueExtents(nested, params, args)
	if nested.TypeArgs[0].Name != "Widget" || nested.TypeArgs[1].Name != "32" || nested.TypeArgs[1].Kind != TypeTemplateValue {
		t.Fatalf("nested generic value argument remained open: %#v", nested)
	}
}

func TestClosedInstantiationInvariantRejectsReachableParameter(t *testing.T) {
	open := Type{Name: "Result", Kind: TypeApplied, TypeArgs: []Type{{Name: "T", Kind: TypeConceptParam}, {Name: "Error", Kind: TypeEnum}}}
	if err := evt1RequireClosedType(open, "test", Span{}); err == nil {
		t.Fatal("closed-instantiation invariant accepted a reachable template parameter")
	}
}

func TestOpenTypeRecognizesSymbolicNonTypeArguments(t *testing.T) {
	open := Type{Name: "Buffer", Kind: TypeApplied, TypeArgs: []Type{
		{Name: "Widget", Kind: TypeStruct},
		{Name: "Capacity", Kind: TypeTemplateValue},
	}}
	if !evt1TypeContainsConceptParameter(open) {
		t.Fatal("symbolic non-type argument must keep a nested generic application open")
	}
	closed := open
	closed.TypeArgs = append([]Type(nil), open.TypeArgs...)
	closed.TypeArgs[1].Name = "8"
	if evt1TypeContainsConceptParameter(closed) {
		t.Fatal("concrete non-type argument must close a nested generic application")
	}
}

func TestCallableExpressionSubstitutionClosesEnvironmentAndBodyTypes(t *testing.T) {
	parameter := Type{Name: "T", Kind: TypeConceptParam}
	callable := &CallableExpr{
		Params:     []Param{{Name: "value", Type: parameter}},
		Captures:   []CaptureBinding{{Name: "saved", Kind: CaptureMove, Type: Type{Name: "Owner", Kind: TypeApplied, TypeArgs: []Type{parameter}}, Source: &NameExpr{Name: "saved"}}},
		ResultType: Type{Name: "Async", Kind: TypeAsync, TypeArgs: []Type{{Name: "Option", Kind: TypeApplied, TypeArgs: []Type{parameter}}}},
		Body:       Block{Statements: []Statement{&VarDecl{Name: "copy", Type: parameter, Value: &NameExpr{Name: "value"}}, &ReturnStmt{Value: &NameExpr{Name: "copy"}}}},
	}
	substituted, err := evt1SubstituteExpr(callable, "T", Type{Name: "Widget", Kind: TypeStruct})
	if err != nil {
		t.Fatal(err)
	}
	closed := substituted.(*CallableExpr)
	if closed.Params[0].Type.Name != "Widget" || closed.Captures[0].Type.String() != "Owner<Widget>" || closed.ResultType.String() != "Async<Option<Widget>>" {
		t.Fatalf("callable signature/environment remained open: %#v", closed)
	}
	if local := closed.Body.Statements[0].(*VarDecl); local.Type.Name != "Widget" {
		t.Fatalf("callable body remained open: %s", local.Type.String())
	}
}

func TestGenericStructConstructionRetainsStructuralType(t *testing.T) {
	module, err := parseSyntaxModule("generic_construct.concept", `profile Core;
template <typename T>
struct Box { T value; };
template <typename T>
Box<T> Make(T value) { Box<T> box = Box<T>{move value}; return move box; }
`)
	if err != nil {
		t.Fatal(err)
	}
	decl := module.Templates[0]
	variable := decl.Body.Statements[0].(*VarDecl)
	construct := variable.Value.(*StructConstructExpr)
	if construct.StructType.Name != "Box" || len(construct.StructType.TypeArgs) != 1 || construct.StructType.TypeArgs[0].Name != "T" {
		t.Fatalf("generic construction lost structural type: %#v", construct.StructType)
	}
	substituted, err := evt1SubstituteExpr(construct, "T", Type{Name: "int", Kind: TypeBuiltin})
	if err != nil {
		t.Fatal(err)
	}
	closed := substituted.(*StructConstructExpr)
	if closed.StructName != "Box<int>" {
		t.Fatalf("substituted construction name = %q, want Box<int>", closed.StructName)
	}
}
