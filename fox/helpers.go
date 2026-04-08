package main

import (
	"fmt"
)

// ================= Utilities =================

func expectIdent(tokens []Token, pos *int) Token {

	if *pos >= len(tokens) {
		panic("unexpected end of input, expected identifier")
	}

	tok := tokens[*pos]

	if tok.Type != IDENT {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected IDENT, got %s",
			tok.Line, tok.Type,
		))
	}

	*pos++
	return tok
}

func expectValue(tokens []Token, pos *int, value string) Token {
	if *pos >= len(tokens) {
		panic("unexpected end of input, expected " + value)
	}
	tok := tokens[*pos]

	if tok.Lexeme != value {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected '%s', got '%s'",
			tok.Line, value, tok.Lexeme,
		))
	}

	*pos++
	return tok
}

func expectType(tokens []Token, pos *int, expected TokenType) Token {

	if *pos >= len(tokens) {
		panic("unexpected end of input")
	}
	tok := tokens[*pos]

	if tok.Type != expected {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type,
		))
	}
	*pos++
	return tok
}

func isAssign(tokens []Token, pos *int) bool {
	if *pos+1 >= len(tokens) {
		return false
	}
	return tokens[*pos].Is(IDENT) && (tokens[*pos+1].Is(ASSIGN) || tokens[*pos+1].Is(DEFINE))
}

func (t Token) Is(tt TokenType) bool {
	return t.Type == tt
}

func (t Token) IsValue(val string) bool {
	return t.Lexeme == val
}
