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
	Plus, Minus, Ref, Star, Slash, Assign, Define, Eq, Neq, Lt, Gt, Lte, Gte, And, Or, Not string
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

var keywords = Keywords{
	Package:  "package",
	Import:   "import",
	Const:    "const",
	Type:     "type",
	Struct:   "struct",
	Func:     "fn",
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
		case keywords.Package:
			// TODO use keyType like : PACKAGE instead keywords.Package
			tokens = append(tokens, Token{Type: keywords.Package, Value: val, Line: line, Column: col})
			return
		case keywords.Type:
			tokens = append(tokens, Token{Type: keywords.Type, Value: val, Line: line, Column: col})
			return
		case keywords.Struct:
			tokens = append(tokens, Token{Type: keywords.Struct, Value: val, Line: line, Column: col})
			return
		case keywords.Func:
			tokens = append(tokens, Token{Type: keywords.Func, Value: val, Line: line, Column: col})
			return
		case keywords.Return:
			tokens = append(tokens, Token{Type: keywords.Return, Value: val, Line: line, Column: col})
			return
		case keywords.Var:
			tokens = append(tokens, Token{Type: keywords.Var, Value: val, Line: line, Column: col})
			return
		case keywords.Const:
			tokens = append(tokens, Token{Type: keywords.Const, Value: val, Line: line, Column: col})
			return
		case keywords.If:
			tokens = append(tokens, Token{Type: keywords.If, Value: val, Line: line, Column: col})
			return
		case keywords.Else:
			tokens = append(tokens, Token{Type: keywords.Else, Value: val, Line: line, Column: col})
			return
		case keywords.For:
			tokens = append(tokens, Token{Type: keywords.For, Value: val, Line: line, Column: col})
			return
		case keywords.Import:
			tokens = append(tokens, Token{Type: keywords.Import, Value: val, Line: line, Column: col})
			return
		case keywords.Break:
			tokens = append(tokens, Token{
				Kind:   KeywordKind,
				Type:   keywords.Break,
				Value:  val,
				Line:   line,
				Column: col,
			})
			return
		case keywords.Continue:
			tokens = append(tokens, Token{
				Kind:   KeywordKind,
				Type:   keywords.Continue,
				Value:  val,
				Line:   line,
				Column: col,
			})
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

		tkn := string(r)
		switch r {

		case '&':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Ref, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '=':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Assign, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '+':
			// إذا كان بعدها رقم فهي جزء من الرقم
			if i+1 < len(input) && unicode.IsDigit(rune(input[i+1])) {
				current.WriteRune(r)
				i++
				col++
				continue
			}
			addToken()
			tokens = append(tokens, Token{Type: Operator.Plus, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '-':
			if i+1 < len(input) && unicode.IsDigit(rune(input[i+1])) {
				current.WriteRune(r)
				i++
				col++
				continue
			}
			addToken()
			tokens = append(tokens, Token{Type: Operator.Minus, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '*':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Star, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '/':
			addToken()
			tokens = append(tokens, Token{Type: Operator.Slash, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '(':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.LParen, Value: tkn, Line: line, Column: col})
			i++
			continue

		case ')':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.RParen, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '{':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.LBrace, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '}':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.RBrace, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '[':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.LBrack, Value: tkn, Line: line, Column: col})
			i++
			continue

		case ']':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.RBrack, Value: tkn, Line: line, Column: col})
			i++
			continue

		case ',':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.Comma, Value: tkn, Line: line, Column: col})
			i++
			continue

		case ';':
			addToken()
			tokens = append(tokens, Token{Type: Delimiter.Semic, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '<':
			addToken()
			tokens = append(tokens, Token{Kind: OperatorKind, Type: Operator.Lt, Value: tkn, Line: line, Column: col})
			i++
			continue

		case '>':
			addToken()
			tokens = append(tokens, Token{Kind: OperatorKind, Type: Operator.Gt, Value: tkn, Line: line, Column: col})
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
