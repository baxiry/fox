package main

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenKind
type TokenKind int

const (
	// Special
	SpecialKind TokenKind = iota
	IdentKind
	NumericLiteralKind
	OtherLiteralKind
	KeywordKind
	OperatorKind
	DelimiterKind
)

// Token
type Token struct {
	Kind   TokenKind
	Type   string
	Value  string
	Line   int
	Column int
}

// Specials
type Specials struct {
	EOF, Illegal string
}

type Idents struct {
	Ident string
}

type Numerics struct {
	Int, Float string
}

type Literals struct {
	String, Bool string
}

type Keywords struct {
	Package, Import, Type, Struct, Func, Var, Const, If, Else, For,
	Continue, Break, Return string
}

type Operators struct {
	Plus, Minus, Star, Slash, Assign, Define, Eq, Neq, Lt, Gt, Lte, Gte, And, Or, Not string
}

type Delimiters struct {
	LParen, RParen, LBrace, RBrace, LBrack, RBrack, Comma, Semic string
}

// Values
var Special = Specials{
	EOF:     "EOF",
	Illegal: "ILLEGAL",
}

var Ident = Idents{
	Ident: "IDENT",
}

var NumericLiteral = Numerics{
	Int:   "INT",
	Float: "FLOAT",
}

var OtherLiteral = Literals{
	String: "STRING",
	Bool:   "BOOL",
}

var Keyword = Keywords{
	Package:  "package",
	Import:   "import",
	Const:    "const",
	Type:     "type",
	Struct:   "struct",
	Func:     "func",
	Var:      "var",
	If:       "if",
	Else:     "else",
	For:      "for",
	Break:    "break",
	Continue: "continue",
	Return:   "return",
}

var Operator = Operators{
	Plus:   "+",
	Minus:  "-",
	Star:   "*",
	Slash:  "/",
	Assign: "=",
	Define: ":=",
	Eq:     "==",
	Neq:    "!=",
	Lt:     "<",
	Gt:     ">",
	Lte:    "<=",
	Gte:    ">=",
	And:    "&&",
	Or:     "||",
	Not:    "!",
}

var Delimiter = Delimiters{
	LParen: "(",
	RParen: ")",
	LBrace: "{",
	RBrace: "}",
	LBrack: "[",
	RBrack: "]",
	Comma:  ",",
	Semic:  ";",
}

//  Lexer

func tokenize(input string) []Token {
	var tokens []Token
	var current strings.Builder
	line, col := 1, 0

	addToken := func() {
		if current.Len() == 0 {
			return
		}
		val := current.String()
		current.Reset()

		switch val {
		case "package":
			tokens = append(tokens, Token{Type: Keyword.Package, Value: val, Line: line, Column: col})
			return
		case "type":
			tokens = append(tokens, Token{Type: Keyword.Type, Value: val, Line: line, Column: col})
			return
		case "struct":
			tokens = append(tokens, Token{Type: Keyword.Struct, Value: val, Line: line, Column: col})
			return
		case "func":
			tokens = append(tokens, Token{Type: Keyword.Func, Value: val, Line: line, Column: col})
			return
		case "return":
			tokens = append(tokens, Token{Type: Keyword.Return, Value: val, Line: line, Column: col})
			return
		case "var":
			tokens = append(tokens, Token{Type: Keyword.Var, Value: val, Line: line, Column: col})
			return
		case "const":
			tokens = append(tokens, Token{Type: Keyword.Const, Value: val, Line: line, Column: col})
			return
		case "if":
			tokens = append(tokens, Token{Type: Keyword.If, Value: val, Line: line, Column: col})
			return
		case "for":
			tokens = append(tokens, Token{Type: Keyword.For, Value: val, Line: line, Column: col})
			return
		case "import":
			tokens = append(tokens, Token{Type: Keyword.Import, Value: val, Line: line, Column: col})
			return
		}

		if isInt(val) {
			tokens = append(tokens, Token{Type: NumericLiteral.Int, Value: val, Line: line, Column: col})
			return
		}
		if isFloat(val) {
			tokens = append(tokens, Token{Type: NumericLiteral.Float, Value: val, Line: line, Column: col})
			return
		}

		tokens = append(tokens, Token{Type: Ident.Ident, Value: val, Line: line, Column: col})
	}

	i := 0
	for i < len(input) {
		r := rune(input[i])
		col++

		if r == '\n' {
			addToken()
			line++
			col = 0
			i++
			continue
		}

		if unicode.IsSpace(r) {
			addToken()
			i++
			continue
		}

		if i+1 < len(input) {
			two := input[i : i+2]
			switch two {
			case ":=":
				addToken()
				tokens = append(tokens, Token{Type: Operator.Define, Value: ":=", Line: line, Column: col})
				i += 2
				col++
				continue
			case "==":
				addToken()
				tokens = append(tokens, Token{Type: Operator.Eq, Value: "==", Line: line, Column: col})
				i += 2
				col++
				continue
			case "!=":
				addToken()
				tokens = append(tokens, Token{Type: Operator.Neq, Value: "!=", Line: line, Column: col})
				i += 2
				col++
				continue
			}
		}

		switch r {
		case '=':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Assign, Value: "=", Line: line, Column: col})
			i++
			continue
		case '+':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Plus, Value: "+", Line: line, Column: col})
			i++
			continue
		case '-':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Minus, Value: "-", Line: line, Column: col})
			i++
			continue
		case '*':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Star, Value: "*", Line: line, Column: col})
			i++
			continue
		case '/':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Slash, Value: "/", Line: line, Column: col})
			i++
			continue
		case '(':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.LParen, Value: "(", Line: line, Column: col})
			i++
			continue
		case ')':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.RParen, Value: ")", Line: line, Column: col})
			i++
			continue
		case '{':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.LBrace, Value: "{", Line: line, Column: col})
			i++
			continue
		case '}':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.RBrace, Value: "}", Line: line, Column: col})
			i++
			continue
		case ',':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.Comma, Value: ",", Line: line, Column: col})
			i++
			continue
		case ';':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.Semic, Value: ";", Line: line, Column: col})
			i++
			continue
		case '"':
			addToken()
			i++
			startCol := col
			var s strings.Builder
			for i < len(input) && input[i] != '"' {
				s.WriteByte(input[i])
				i++
				col++
			}
			i++
			tokens = append(tokens, Token{Type: OtherLiteral.String, Value: s.String(), Line: line, Column: startCol})
			continue
		}

		current.WriteRune(r)
		i++
	}

	addToken()
	return tokens
}

// ================= Helpers =================

func IsLiteral(tok Token) bool {
	return tok.Kind == NumericLiteralKind || tok.Kind == OtherLiteralKind
}

func IsNumericLiteral(tok Token) bool {
	return tok.Kind == NumericLiteralKind
}

func IsKeyword(tok Token) bool {
	return tok.Kind == KeywordKind
}

func IsOperator(tok Token) bool {
	return tok.Kind == OperatorKind
}

func IsDelimiter(tok Token) bool {
	return tok.Kind == DelimiterKind
}
func (t Token) String() string {
	return fmt.Sprintf(
		"{Type: %s, Value: '%s', File: %s, Line: %d, Column: %d}",
		t.Type, t.Value, "test.fox", t.Line, t.Column,
	)
}

// ================= Utils =================

func isInt(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && (r == '+' || r == '-') {
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isFloat(s string) bool {
	if s == "" {
		return false
	}
	dotSeen := false
	for i, r := range s {
		if i == 0 && (r == '+' || r == '-') {
			continue
		}
		if r == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return dotSeen
}
