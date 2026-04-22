package aster

import (
	"fmt"
	"strings"
)

type ParseError struct {
	File   string
	Line   int
	Column int
	Msg    string
}

var strf = fmt.Sprintf

// Utilities

func (p *Parser) expectIdent() Token {

	if p.pos >= len(p.tokens) {
		p.Errors = append(p.Errors, fmt.Sprintf(
			"Line: %d unexpected end of input, expected identifier",
			p.currentToken().Line,
		))
		return Token{Line: p.currentToken().Line}
	}

	tok := p.tokens[p.pos]

	if tok.Type != IDENT {

		p.Errors = append(p.Errors, fmt.Sprintf(
			"syntax error at line %d: expected IDENT, got %s",
			tok.Line, tok.Type))

		return Token{Line: p.currentToken().Line}
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

func (p *Parser) currentToken() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF, Lexeme: "EOF", Line: p.currentToken().Line}
	}
	return p.tokens[p.pos]
}

func (p *Parser) expectType(expected TokenType) Token {

	if p.pos >= len(p.tokens) {

		p.Errors = append(p.Errors, fmt.Sprintf("Line %d unexpected end of file", p.currentToken().Line))
		return Token{Line: p.currentToken().Line}
	}

	tok := p.currentToken()

	if tok.Type != expected {
		p.Errors = append(p.Errors, fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type,
		))
		return Token{Line: p.currentToken().Line}
	}
	p.pos++
	return tok
}

func (p *Parser) readNumber(src string, pos *int) Token {
	start := *pos

	// integer part
	for *pos < len(src) && isDigit(src[*pos]) {
		*pos++
	}

	isFloat := false
	if *pos < len(src) && src[*pos] == '.' {
		isFloat = true
		*pos++
		for *pos < len(src) && isDigit(src[*pos]) {
			*pos++
		}
	}

	// suffix
	sufStart := *pos
	for *pos < len(src) && isLetterOrDigit(src[*pos]) {
		*pos++
	}

	value := src[start:sufStart]
	suffix := src[sufStart:*pos]

	//  new rule

	line := p.currentToken().Line
	if isFloat && (strings.HasPrefix(suffix, "i") || strings.HasPrefix(suffix, "u")) {
		p.appendErrorf("invalid numeric literal: float cannot have integer suffix: %s%s", line, value, suffix)
	}
	//

	return Token{
		Type:   FLOAT,
		Lexeme: value,
		Line:   line,
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isLetterOrDigit(b byte) bool {
	return isLetter(b) || isDigit(b)
}

// end
