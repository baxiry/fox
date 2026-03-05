package main

import (
	"testing"
)

func TestParseType_Success(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Star, Value: "*", Line: 1},
		{Type: Operator.Star, Value: "*", Line: 1},
		{Type: Ident.Ident, Value: "Obj", Line: 1},
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

func TestParseType_FailRef(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&", Line: 1},
		{Type: Ident.Ident, Value: "Obj", Line: 1},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when using & in type signature")
		}
	}()
	parseType(tokens, &pos)
}

func TestParseUnary_Success(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&", Line: 1},
		{Type: Ident.Ident, Value: "x", Line: 1},
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

func TestParseUnary_FailEnd(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Ref, Value: "&", Line: 1},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic at end of input after unary operator")
		}
	}()
	parseUnary(tokens, &pos)
}

func TestParsePrimary_NumberStringIdent(t *testing.T) {
	tokens := []Token{
		{Type: NumericLiteral.Int, Value: "42", Line: 1},
		{Type: OtherLiteral.String, Value: "hello", Line: 1},
		{Type: Ident.Ident, Value: "foo", Line: 1},
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

func TestParsePrimary_Fail(t *testing.T) {
	tokens := []Token{
		{Type: Operator.Plus, Value: "+", Line: 1},
	}

	pos := 0
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid primary expression")
		}
	}()
	parsePrimary(tokens, &pos)
}

func TestParseFunc_Simple(t *testing.T) {
	tokens := []Token{
		{Type: keywords.Func, Value: "fn", Line: 1},
		{Type: Ident.Ident, Value: "make", Line: 1},
		{Type: Delimiter.LParen, Value: "(", Line: 1},
		{Type: Delimiter.RParen, Value: ")", Line: 1},
		{Type: Delimiter.LBrace, Value: "{", Line: 1},
		{Type: Ident.Ident, Value: "x", Line: 1},
		{Type: Operator.Assign, Value: "=", Line: 1},
		{Type: Ident.Ident, Value: "Obj", Line: 1},
		{Type: Delimiter.RBrace, Value: "}", Line: 1},
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
