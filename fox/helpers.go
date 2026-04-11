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
		fmt.Println("token: ", tok.Lexeme)
		panic(fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type,
		))
	}
	*pos++
	return tok
}

/*
func isAssign(tokens []Token, pos int) bool {
	for i := pos; i < len(tokens); i++ {
		if tokens[i].Type == ASSIGN {
			return true
		}
		if tokens[i].Type == SEMICOLON || tokens[i].Type == CLS_BRACE {
			return false
		}
	}
	return false
}
func isDefine(tokens []Token, pos int) bool {
	for i := pos; i < len(tokens); i++ {
		if tokens[i].Type == DEFINE {
			return true
		}
		if tokens[i].Type == SEMICOLON || tokens[i].Type == CLS_BRACE {
			return false
		}
	}
	return false
}
*/

func (t Token) Is(tt TokenType) bool {
	return t.Type == tt
}

func (t Token) IsValue(val string) bool {
	return t.Lexeme == val
}
