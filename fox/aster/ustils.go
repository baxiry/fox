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
func (p *Parser) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	finalMsg := fmt.Sprintf("line %d: %s", line, msg)
	p.Errors = append(p.Errors, finalMsg)
}

func (p *Parser) expectIdent() Token {

	if p.pos >= len(p.tokens) {
		p.Errors = append(p.Errors, strf(
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

func (p *Parser) expectType(expected TokenType) Token {

	if p.pos >= len(p.tokens) {
		lastLine := 0
		if len(p.tokens) > 0 {
			lastLine = p.tokens[len(p.tokens)-1].Line
		}
		p.Errors = append(p.Errors, fmt.Sprintf("line %d: unexpected end of file", lastLine))
		return Token{Type: EOF, Line: lastLine}
	}

	tok := p.currentToken()

	if tok.Type != expected {

		p.Errors = append(p.Errors, fmt.Sprintf(
			"syntax error at line %d: expected %s, got %s",
			tok.Line, expected.String(), tok.Type.String(),
		))

		p.pos++
		return tok
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

// current get the current token
func (p *Parser) currentToken() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF, Lexeme: "EOF", Line: p.currentToken().Line}
	}
	return p.tokens[p.pos]
}

// peekToken peek next token
func (p *Parser) peekToken() Token {
	if p.pos+1 >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos+1]
}

// advance pushes the course forward by one step
func (p *Parser) advanceToken() {
	if p.pos < len(p.tokens) {
		p.pos++
	}

	if p.debug {
		curr := p.currentToken()
		fmt.Printf("[DEBUG] Moving to Pos: %d | Token: %s | Lexeme: '%s'\n",
			p.pos, curr.Type, curr.Lexeme)
	}
}

func (p *Parser) makeType(name string, ptrDepth int) *Type {
	return &Type{
		Name:     name,
		PtrDepth: ptrDepth,
	}
}

// end
