package main

import "testing"

func TestTokenizeOperators(t *testing.T) {
	src := "& * + - / := = =="
	tokens := tokenize(src)

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

func TestTokenizeKeywords(t *testing.T) {
	src := "fn return var const if else for break continue package import type struct"
	tokens := tokenize(src)

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

func TestTokenizeNumbers(t *testing.T) {
	src := "42 -7 3.14 -0.5"
	tokens := tokenize(src)

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

func TestTokenizeStrings(t *testing.T) {
	src := `"hello" "world"`
	tokens := tokenize(src)

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	for _, tok := range tokens {
		if tok.Type != OtherLiteral.String {
			t.Errorf("expected string literal, got %s", tok.Type)
		}
	}
}

func TestTokenizeDelimiters(t *testing.T) {
	src := "( ) { } [ ] , ;"
	tokens := tokenize(src)

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

func TestTokenizeIllegal(t *testing.T) {
	src := "@ # $"
	tokens := tokenize(src)

	for _, tok := range tokens {
		if tok.Type != Ident.Ident {
			t.Errorf("unexpected token type for illegal char: %s", tok.Type)
		}
	}
}
