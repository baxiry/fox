package aster

import (
	"fmt"
)

// Utilities

func (p *Parser) expectIdent() Token {

	if p.pos >= len(p.tokens) {
		panic("unexpected end of input, expected identifier")
	}

	tok := p.tokens[p.pos]

	if tok.Type != IDENT {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected IDENT, got %s",
			tok.Line, tok.Type,
		))
	}

	p.pos++
	return tok
}

func (p *Parser) skip() {
	for p.pos < len(p.tokens) &&
		(p.tokens[p.pos].Type == NEW_LINE || p.tokens[p.pos].Type == COMMENT) {
		p.pos++
	}
}

func (p *Parser) expectType(expected TokenType) Token {

	if p.pos >= len(p.tokens) {
		panic("unexpected end of input")
	}
	tok := p.tokens[p.pos]

	if tok.Type != expected {
		panic(fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type,
		))
	}
	p.pos++
	return tok
}
