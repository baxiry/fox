package aster

import (
	"fmt"
)

// Utilities

func (p *Parser) expectIdent() Token {

	if p.pos >= len(p.tokens) {
		p.errors = append(p.errors, fmt.Sprintf("unexpected end of input, expected identifier"))
		return Token{}
	}

	tok := p.tokens[p.pos]

	if tok.Type != IDENT {

		p.errors = append(p.errors, fmt.Sprintf(
			"syntax error at line %d: expected IDENT, got %s",
			tok.Line, tok.Type))

		return Token{}
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

		p.errors = append(p.errors, "unexpected end of input")
		return Token{}
	}

	tok := p.tokens[p.pos]

	if tok.Type != expected {
		p.errors = append(p.errors, fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type,
		))
		return Token{}
	}
	p.pos++
	return tok
}
