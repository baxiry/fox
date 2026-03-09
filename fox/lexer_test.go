package main

import "testing"

func Test_TokenizeOperators(t *testing.T) {
	src := "& * + - / := = =="
	tokens := Lexer(src)

	expectedTypes := []string{
		Operator.Ref, Operator.Star, Operator.Plus, Operator.Minus,
		Operator.Slash, Operator.Define, Operator.Assign, Operator.Eq,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("token %d: expected type %s, got %s", i, expectedTypes[i], tok.Type)
		}
	}
}

func Test_TokenizeKeywords(t *testing.T) {
	src := "fn return var const if else for break continue package import type struct"
	tokens := Lexer(src)

	expectedTypes := []string{
		keywords.Func, keywords.Return, keywords.Var, keywords.Const,
		keywords.If, keywords.Else, keywords.For, keywords.Break, keywords.Continue,
		keywords.Package, keywords.Import, keywords.Type, keywords.Struct,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("token %d: expected type %s, got %s", i, expectedTypes[i], tok.Type)
		}
	}
}

func Test_TokenizeNumbers(t *testing.T) {
	src := "42 -7 3.14 -0.5"
	tokens := Lexer(src)

	expectedTypes := []string{
		NumericLiteral.Int, NumericLiteral.Int,
		NumericLiteral.Float, NumericLiteral.Float,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("token %d: expected type %s, got %s", i, expectedTypes[i], tok.Type)
		}
	}
}

func Test_TokenizeStrings(t *testing.T) {
	src := `"hello" "world"`
	tokens := Lexer(src)

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	for _, tok := range tokens {
		if tok.Type != OtherLiteral.String {
			t.Errorf("expected string literal, got %s", tok.Type)
		}
	}
}

func Test_TokenizeDelimiters(t *testing.T) {
	src := "( ) { } [ ] , ;"
	tokens := Lexer(src)

	expectedTypes := []string{
		Delimiter.LParen, Delimiter.RParen,
		Delimiter.LBrace, Delimiter.RBrace,
		Delimiter.LBrack, Delimiter.RBrack,
		Delimiter.Comma, Delimiter.Semic,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, tok := range tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("token %d: expected type %s, got %s", i, expectedTypes[i], tok.Type)
		}
	}
}

func Test_TokenizeIllegal(t *testing.T) {
	src := "@ # $"
	tokens := Lexer(src)

	for _, tok := range tokens {
		if tok.Type != Ident.Ident {
			t.Errorf("unexpected token type for illegal char: %s", tok.Type)
		}
	}
}

func Test_TokenizeIncrementCases(t *testing.T) {
	tests := []struct {
		src      string
		expected []struct {
			Type  string
			Value string
		}
	}{
		{"i+1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},
		{"i +1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},
		{"i+ 1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},
		{"i=i+1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Assign, "="}, {Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},
		{"i=i +1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Assign, "="}, {Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},
		{"i=i + 1", []struct{ Type, Value string }{
			{Ident.Ident, "i"}, {Operator.Assign, "="}, {Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "1"},
		}},

		{"+1", []struct{ Type, Value string }{{NumericLiteral.Int, "+1"}}},
		{"-1", []struct{ Type, Value string }{{NumericLiteral.Int, "-1"}}},
		{"i+ -1", []struct{ Type, Value string }{{Ident.Ident, "i"}, {Operator.Plus, "+"}, {NumericLiteral.Int, "-1"}}},
		{"i- +1", []struct{ Type, Value string }{{Ident.Ident, "i"}, {Operator.Minus, "-"}, {NumericLiteral.Int, "+1"}}},
	}

	for _, tt := range tests {
		tokens := Lexer(tt.src)
		if len(tokens) != len(tt.expected) {
			t.Fatalf("input=%q expected %d tokens, got %d", tt.src, len(tt.expected), len(tokens))
		}
		for i, tok := range tokens {
			if tok.Type != tt.expected[i].Type || tok.Value != tt.expected[i].Value {
				t.Errorf("input=%q token %d: expected (%s,%s), got (%s,%s)",
					tt.src, i, tt.expected[i].Type, tt.expected[i].Value, tok.Type, tok.Value)
			}
		}
	}
}
