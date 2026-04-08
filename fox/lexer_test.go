package main

import "testing"

func Test_TokenizeOperators(t *testing.T) {
	src := "& * + - / := = =="
	tokens := Lexer(src)

	expectedTypes := []TokenType{
		AND, STAR, PLUS, MINUS,
		SLASH, DEFINE, ASSIGN, EQ,
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

	expectedTypes := []TokenType{
		FUNC, RETURN, VAR, CONST,
		IF, ELSE, FOR, BREAK, CONTINUE,
		PACKAGE, IMPORT, TYPE, STRUCT,
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

	expectedTypes := []TokenType{
		INT, FLOAT,
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
		if tok.Type != STRING {
			t.Errorf("expected string literal, got %s", tok.Type)
		}
	}
}

func Test_TokenizeDelimiters(t *testing.T) {
	src := "( ) { } [ ] , ;"
	tokens := Lexer(src)

	expectedTypes := []TokenType{
		OPN_PAREN, CLS_PAREN,
		OPN_BRACE, CLS_BRACE,
		OPN_BRACK, CLS_BRACK,
		COMMA, SEMICOLON,
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
		if tok.Type != IDENT {
			t.Errorf("unexpected token type for illegal char: %s", tok.Type)
		}
	}
}

type TokCase struct {
	Type  TokenType
	Value string
}

func Test_TokenizeSimpleArithmetic(t *testing.T) {
	tests := []struct {
		src      string
		expected []TokCase
	}{
		{"i+1", []TokCase{{IDENT, "i"}, {PLUS, "+"}, {INT, "1"}}},
		{"i + 1", []TokCase{{IDENT, "i"}, {PLUS, "+"}, {INT, "1"}}},
		{"i-1", []TokCase{{IDENT, "i"}, {MINUS, "-"}, {INT, "1"}}},
		{"i - 1", []TokCase{{IDENT, "i"}, {MINUS, "-"}, {INT, "1"}}},
		{"i=i+1", []TokCase{{IDENT, "i"}, {ASSIGN, "="}, {IDENT, "i"}, {PLUS, "+"}, {INT, "1"}}},
		{"i=i-1", []TokCase{{IDENT, "i"}, {ASSIGN, "="}, {IDENT, "i"}, {MINUS, "-"}, {INT, "1"}}},
		{"+1", []TokCase{{INT, "+1"}}},
		{"-1", []TokCase{{INT, "-1"}}},
	}

	for _, tt := range tests {
		tokens := Lexer(tt.src)
		if len(tokens) != len(tt.expected) {
			t.Fatalf("input=%q expected %d tokens, got %d", tt.src, len(tt.expected), len(tokens))
		}
		for i, tok := range tokens {
			if tok.Type != tt.expected[i].Type || tok.Lexeme != tt.expected[i].Value {
				t.Errorf("input=%q token %d: expected (%s,%s), got (%s,%s)",
					tt.src, i, tt.expected[i].Type, tt.expected[i].Value, tok.Type, tok.Lexeme)
			}
		}
	}
}
