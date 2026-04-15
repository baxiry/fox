package aster

import (
	"fmt"
)

// Utilities

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

func skip(tokens []Token, pos *int) {
	for *pos < len(tokens) &&
		(tokens[*pos].Type == NEW_LINE || tokens[*pos].Type == COMMENT) {
		*pos++
	}
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
