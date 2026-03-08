package main

import (
	"testing"
)

func Test_ParseType_Success(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Star, Value: "*"},
		{Type: Operator.Star, Value: "*"},
		{Type: Ident.Ident, Value: "Obj"},
	}

	pos := 0
	typ := parseType(tokens, &pos)

	if typ.Name != "Obj" {
		t.Errorf("expected type name Obj, got %s", typ.Name)
	}
	if typ.PtrDepth != 2 {
		t.Errorf("expected pointer depth 2, got %d", typ.PtrDepth)
	}
}

func Test_ParseType_FailRef(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&"},
		{Type: Ident.Ident, Value: "Obj"},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when using & in type signature")
		}
	}()
	parseType(tokens, &pos)
}

func Test_ParseUnary_Success(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&"},
		{Type: Ident.Ident, Value: "x"},
	}

	pos := 0
	expr := parseUnary(tokens, &pos)

	u, ok := expr.(UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}

	if u.Op != "&" {
		t.Errorf("expected Op '&', got %s", u.Op)
	}

	id, ok := u.Expr.(IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr inside unary, got %T", u.Expr)
	}

	if id.Name != "x" {
		t.Errorf("expected name 'x', got %s", id.Name)
	}
}

func Test_ParseUnary_FailEnd(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&"},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic at end of input after unary operator")
		}
	}()
	parseUnary(tokens, &pos)
}

func Test_ParsePrimary_NumberStringIdent(t *testing.T) {
	tokens := []Token{
		{Type: NumericLiteral.Int, Value: "42"},
		{Type: OtherLiteral.String, Value: "hello"},
		{Type: Ident.Ident, Value: "foo"},
	}

	pos := 0
	num := parsePrimary(tokens, &pos)
	if n, ok := num.(NumberExpr); !ok || n.Literal != "42" {
		t.Errorf("expected NumberExpr 42, got %#v", num)
	}

	str := parsePrimary(tokens, &pos)
	if s, ok := str.(StringExpr); !ok || s.Literal != "hello" {
		t.Errorf("expected StringExpr 'hello', got %#v", str)
	}

	id := parsePrimary(tokens, &pos)
	if i, ok := id.(IdentExpr); !ok || i.Name != "foo" {
		t.Errorf("expected IdentExpr 'foo', got %#v", id)
	}
}

func Test_ParsePrimary_Fail(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Plus, Value: "+"},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid primary expression")
		}
	}()
	parsePrimary(tokens, &pos)
}

func Test_ParseFunc_Simple(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Func, Value: "fn"},
		{Type: Ident.Ident, Value: "make"},
		{Type: Delimiter.LParen, Value: "("},
		{Type: Delimiter.RParen, Value: ")"},
		{Type: Delimiter.LBrace, Value: "{"},
		{Type: Ident.Ident, Value: "x"},
		{Type: Operator.Assign, Value: "="},
		{Type: Ident.Ident, Value: "Obj"},
		{Type: Delimiter.RBrace, Value: "}"},
	}

	pos := 0
	fn := parseFunc(tokens, &pos)

	if fn.Name != "make" {
		t.Errorf("expected function name 'make', got %s", fn.Name)
	}
	if len(fn.Body) != 1 { // variable + expression
		t.Errorf("expected body length 2, got %d", len(fn.Body))
	}
}

func Test_ParseUnaryRef(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&"},
		{Type: Ident.Ident, Value: "x"},
	}

	pos := 0
	expr := parseUnary(tokens, &pos)

	unary, ok := expr.(UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}

	if unary.Op != "&" {
		t.Errorf("expected operator &, got %s", unary.Op)
	}

	ident, ok := unary.Expr.(IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr, got %T", unary.Expr)
	}

	if ident.Name != "x" {
		t.Errorf("expected identifier x, got %s", ident.Name)
	}
}

func Test_ParseUnaryMinus(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Minus, Value: "-"},
		{Type: Ident.Ident, Value: "x"},
	}

	pos := 0
	expr := parseUnary(tokens, &pos)

	unary, ok := expr.(UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}

	if unary.Op != "-" {
		t.Errorf("expected operator -, got %s", unary.Op)
	}

	ident, ok := unary.Expr.(IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr, got %T", unary.Expr)
	}

	if ident.Name != "x" {
		t.Errorf("expected identifier x, got %s", ident.Name)
	}
}

func Test_ParseUnaryNested(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Star, Value: "*"},
		{Type: Operator.Ref, Value: "&"},
		{Type: Ident.Ident, Value: "x"},
	}

	pos := 0
	expr := parseUnary(tokens, &pos)

	outer, ok := expr.(UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}

	if outer.Op != "*" {
		t.Errorf("expected outer operator *, got %s", outer.Op)
	}

	inner, ok := outer.Expr.(UnaryExpr)
	if !ok {
		t.Fatalf("expected inner UnaryExpr, got %T", outer.Expr)
	}

	if inner.Op != "&" {
		t.Errorf("expected inner operator &, got %s", inner.Op)
	}

	ident, ok := inner.Expr.(IdentExpr)
	if !ok {
		t.Fatalf("expected IdentExpr, got %T", inner.Expr)
	}

	if ident.Name != "x" {
		t.Errorf("expected identifier x, got %s", ident.Name)
	}
}

func Test_ParseType_Simple(t *testing.T) {
	tokens := []Token{
		{Type: Ident.Ident, Value: "int"},
	}
	pos := 0
	typ := parseType(tokens, &pos)

	if typ.Name != "int" {
		t.Errorf("expected type name 'int', got %s", typ.Name)
	}
	if typ.PtrDepth != 0 {
		t.Errorf("expected ptr depth 0, got %d", typ.PtrDepth)
	}
}

func Test_ParseType_Pointer(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Star, Value: "*"},
		{Type: Operator.Star, Value: "*"},
		{Type: Ident.Ident, Value: "float"},
	}
	pos := 0
	typ := parseType(tokens, &pos)

	if typ.Name != "float" {
		t.Errorf("expected type name 'float', got %s", typ.Name)
	}
	if typ.PtrDepth != 2 {
		t.Errorf("expected ptr depth 2, got %d", typ.PtrDepth)
	}
}

func Test_ParseType_InvalidRef(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&"},
		{Type: Ident.Ident, Value: "Obj"},
	}
	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic due to & in type, but did not panic")
		}
	}()
	_ = parseType(tokens, &pos)
}

func Test_ParseStruct_Simple(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Type, Value: "type"},
		{Type: Ident.Ident, Value: "Obj"},
		{Type: keywords.Struct, Value: "struct"},
		{Type: Delimiter.LBrace, Value: "{"},
		{Type: Ident.Ident, Value: "value"},
		{Type: Ident.Ident, Value: "int"},
		{Type: Delimiter.RBrace, Value: "}"},
	}

	pos := 0
	s := parseStruct(tokens, &pos)

	if s.Name != "Obj" {
		t.Errorf("expected struct name 'Obj', got %s", s.Name)
	}
	if len(s.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(s.Fields))
	}
	if s.Fields[0].Name != "value" || s.Fields[0].Type != "int" {
		t.Errorf("expected field 'value int', got %s %s", s.Fields[0].Name, s.Fields[0].Type)
	}
}

func Test_ParseStruct_MultipleFields(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Type, Value: "type"},
		{Type: Ident.Ident, Value: "Person"},
		{Type: keywords.Struct, Value: "struct"},
		{Type: Delimiter.LBrace, Value: "{"},
		{Type: Ident.Ident, Value: "name"},
		{Type: Ident.Ident, Value: "string"},
		{Type: Ident.Ident, Value: "age"},
		{Type: Ident.Ident, Value: "int"},
		{Type: Delimiter.RBrace, Value: "}"},
	}

	pos := 0
	s := parseStruct(tokens, &pos)

	if len(s.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(s.Fields))
	}
	if s.Fields[1].Name != "age" || s.Fields[1].Type != "int" {
		t.Errorf("expected second field 'age int', got %s %s", s.Fields[1].Name, s.Fields[1].Type)
	}
}

func Test_ParseField_Invalid(t *testing.T) {
	tokens := []Token{
		{Type: Ident.Ident, Value: "x"},
		{Type: Operator.Plus, Value: "+"}, // invalid type
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic due to invalid type token, but did not panic")
		}
	}()

	_ = parseField(tokens, &pos)
}

func Test_ParseFunc_NoParams(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Func, Value: "fn"},
		{Type: Ident.Ident, Value: "make"},
		{Type: Delimiter.LParen, Value: "("},
		{Type: Delimiter.RParen, Value: ")"},
		{Type: Delimiter.LBrace, Value: "{"},
		{Type: Ident.Ident, Value: "x"},
		{Type: Operator.Assign, Value: "="},
		{Type: Ident.Ident, Value: "Obj"},
		{Type: Delimiter.RBrace, Value: "}"},
	}

	pos := 0
	fn := parseFunc(tokens, &pos)

	if fn.Name != "make" {
		t.Errorf("expected function name 'make', got %s", fn.Name)
	}
	if len(fn.Params) != 0 {
		t.Errorf("expected 0 params, got %d", len(fn.Params))
	}
	if len(fn.Body) != 1 {
		t.Errorf("expected body length 2, got %d", len(fn.Body))
	}
}

func Test_ParseFunc_WithParamsAndReturn(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Func, Value: "fn"},
		{Type: Ident.Ident, Value: "add"},
		{Type: Delimiter.LParen, Value: "("},
		{Type: Ident.Ident, Value: "a"},
		{Type: Ident.Ident, Value: "int"},
		{Type: Delimiter.Comma, Value: ","},
		{Type: Ident.Ident, Value: "b"},
		{Type: Ident.Ident, Value: "int"},
		{Type: Delimiter.RParen, Value: ")"},
		{Type: Ident.Ident, Value: "int"}, // return type
		{Type: Delimiter.LBrace, Value: "{"},
		{Type: keywords.Return, Value: "return"},
		{Type: Ident.Ident, Value: "a"},
		{Type: Operator.Plus, Value: "+"},
		{Type: Ident.Ident, Value: "b"},
		{Type: Delimiter.RBrace, Value: "}"},
	}

	pos := 0
	fn := parseFunc(tokens, &pos)

	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
	if len(fn.Returns) != 1 {
		t.Errorf("expected 1 return, got %d", len(fn.Returns))
	} else if fn.Returns[0].Type.Name != "int" {
		t.Errorf("expected return type 'int', got %s", fn.Returns[0].Type.Name)
	}
	//if fn.Returns.Name != "int" {
	//	t.Errorf("expected return type 'int', got %s", fn.Returns.Name)
	//	}
}

func Test_ParseFunc_SyntaxError(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Func, Value: "fn"},
		{Type: Ident.Ident, Value: "bad"},
		{Type: Delimiter.LParen, Value: "("},
		// missing closing parenthesis
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic due to missing ')', but did not panic")
		}
	}()

	_ = parseFunc(tokens, &pos)
}
