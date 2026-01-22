package main

import "fmt"

// ================= Utilities =================

func expectIdent(tokens []Token, pos *int) Token {
	if *pos >= len(tokens) {
		panic("unexpected end of input, expected identifier")
	}

	tok := tokens[*pos]

	if tok.Type != Ident.Ident {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected IDENT, got %s",
			tok.Line, tok.Type,
		))
	}

	*pos++
	return tok
}

func expectValue(tokens []Token, pos *int, value string) {
	if *pos >= len(tokens) {
		panic("unexpected end of file, expected " + value)
	}
	tok := tokens[*pos]

	if tok.Value != value {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected '%s', got '%s'",
			tok.Line, value, tok.Value,
		))
	}
	*pos++
}

func expectType(tokens []Token, pos *int, expected string) Token {
	if *pos >= len(tokens) {
		panic("unexpected end of input")
	}
	tok := tokens[*pos]

	if tok.Type != expected {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected, tok.Type,
		))
	}
	*pos++
	return tok
}

func expectKind(tokens []Token, pos *int, kind TokenKind, expectedText string) Token {
	if *pos >= len(tokens) {
		panic("unexpected end of input")
	}
	tok := tokens[*pos]

	if tok.Kind != kind {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expectedText, tok.Type,
		))
	}
	*pos++
	return tok
}
func isAssign(tokens []Token, pos *int) bool {
	if *pos+1 >= len(tokens) {
		return false
	}
	return tokens[*pos].Type == Ident.Ident &&
		(tokens[*pos+1].Type == Operator.Assign ||
			tokens[*pos+1].Type == Operator.Define)
}
