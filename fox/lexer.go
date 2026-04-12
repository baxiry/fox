package main

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	// Special
	EOF TokenType = iota
	ILLEGAL

	// Identifiers & literals
	START_IDENTIFIERS
	IDENT
	INT
	FLOAT
	STRING
	BOOL
	END_IDENTIF

	// Keywords
	START_KEYWORDS
	PACKAGE
	IMPORT
	TYPE
	STRUCT
	FUNC
	VAR
	CONST
	IF
	ELSE
	FOR
	BREAK
	CONTINUE
	RETURN
	FALLTHROUGH
	GOTO
	END_KEYWORDS

	// Operators
	START_OPERATORS
	PLUS
	MINUS
	STAR
	SLASH
	ASSIGN
	DEFINE
	PLUS_ASSIGN
	PLUS_DEFINE
	EQ
	NEQ
	LT
	GT
	LTE
	GTE
	AND
	AMP
	OR
	NOT
	DOT
	EDN_OPERATORS

	// Delimiters
	START_DELIMITERS
	OPN_PAREN
	CLS_PAREN
	OPN_BRACE
	CLS_BRACE
	OPN_BRACK
	CLS_BRACK
	COMMA
	SEMICOLON
	DUBLE_DOT
	NEW_LINE
	END_DELIMITERS

	TRUE
	FALSE

	SPAWN

	COMMENT
)

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}

var keywords = map[string]TokenType{
	"package":     PACKAGE,
	"type":        TYPE,
	"struct":      STRUCT,
	"fn":          FUNC,
	"goto":        GOTO,
	"return":      RETURN,
	"var":         VAR,
	"const":       CONST,
	"if":          IF,
	"fallthrough": FALLTHROUGH,
	"else":        ELSE,
	"for":         FOR,
	"import":      IMPORT,
	"break":       BREAK,
	"continue":    CONTINUE,
	"spawn":       SPAWN,
}

func Lexer(input string) []Token {
	var tokens []Token
	var current strings.Builder
	line, col := 1, 0

	addToken := func() {
		if current.Len() == 0 {
			return
		}
		val := current.String()
		fmt.Println("currnet: ", val)
		current.Reset()

		// Keywords
		if tt, ok := keywords[val]; ok {
			tokens = append(tokens, Token{Type: tt, Lexeme: val, Line: line, Column: col})
			return
		}

		if isInt(val) {
			tokens = append(tokens, Token{Type: INT, Lexeme: val, Line: line, Column: col})
			return
		}
		if isFloat(val) {
			tokens = append(tokens, Token{Type: FLOAT, Lexeme: val, Line: line, Column: col})
			return
		}

		// Split identifiers by dot
		parts := strings.Split(val, ".")
		for i, p := range parts {
			if p == "" {
				continue
			}
			tokens = append(tokens, Token{Type: IDENT, Lexeme: p, Line: line, Column: col})
			if i < len(parts)-1 {
				tokens = append(tokens, Token{Type: DOT, Lexeme: ".", Line: line, Column: col})
			}
		}
	}

	i := 0
	for i < len(input) {
		r := rune(input[i])
		col++

		if r == '\n' {
			addToken()

			if len(tokens) == 0 || tokens[len(tokens)-1].Type != NEW_LINE {
				tokens = append(tokens, Token{
					Type:   NEW_LINE,
					Lexeme: "\n",
					Line:   line,
					Column: col,
				})
			}

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

		// Two-char operators
		if i+1 < len(input) {
			two := input[i : i+2]
			switch two {
			case ":=":
				addToken()
				tokens = append(tokens, Token{Type: DEFINE, Lexeme: ":=", Line: line, Column: col})
				i += 2
				col++
				continue
			case "==":
				addToken()
				tokens = append(tokens, Token{Type: EQ, Lexeme: "==", Line: line, Column: col})
				i += 2
				col++
				continue
			case "!=":
				addToken()
				tokens = append(tokens, Token{Type: NEQ, Lexeme: "!=", Line: line, Column: col})
				i += 2
				col++
				continue
			case "&&":
				addToken()
				tokens = append(tokens, Token{Type: AND, Lexeme: "&&", Line: line, Column: col})
				i += 2
				col++
				continue
			case "||":
				addToken()
				tokens = append(tokens, Token{Type: OR, Lexeme: "||", Line: line, Column: col})
				i += 2
				col++
				continue

			case "//":
				addToken()
				tokens = append(tokens, Token{Type: COMMENT, Lexeme: "//", Line: line, Column: col})
				i += 2
				col++
				continue

			}
		}

		tkn := string(r)

		switch r {

		case '&':
			addToken()
			tokens = append(tokens, Token{Type: AMP, Lexeme: string(r), Line: line, Column: col})
			i++
			continue

		case '=':
			addToken()
			tokens = append(tokens, Token{Type: ASSIGN, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '+', '-':
			if current.Len() > 0 {
				val := current.String()
				current.Reset()

				if isInt(val) {
					tokens = append(tokens, Token{Type: INT, Lexeme: val, Line: line, Column: col})
				} else if isFloat(val) {
					tokens = append(tokens, Token{Type: FLOAT, Lexeme: val, Line: line, Column: col})
				} else {
					tokens = append(tokens, Token{Type: IDENT, Lexeme: val, Line: line, Column: col})
				}
			}

			// unary sign detection
			if i+1 < len(input) && unicode.IsDigit(rune(input[i+1])) &&
				(len(tokens) == 0 || tokens[len(tokens)-1].Type != IDENT) {
				current.WriteRune(r)
				i++
				col++
				continue
			}

			if r == '+' {
				tokens = append(tokens, Token{Type: PLUS, Lexeme: tkn, Line: line, Column: col})
			} else {
				tokens = append(tokens, Token{Type: MINUS, Lexeme: tkn, Line: line, Column: col})
			}
			i++
			continue

		case '*':
			addToken()
			tokens = append(tokens, Token{Type: STAR, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '!':
			addToken()
			tokens = append(tokens, Token{Type: NOT, Lexeme: tkn, Line: line, Column: col})
			i++
			continue
		case '/':
			addToken()
			tokens = append(tokens, Token{Type: SLASH, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '(':
			addToken()
			tokens = append(tokens, Token{Type: OPN_PAREN, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case ')':
			addToken()
			tokens = append(tokens, Token{Type: CLS_PAREN, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '{':
			addToken()
			tokens = append(tokens, Token{Type: OPN_BRACE, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '}':
			addToken()
			tokens = append(tokens, Token{Type: CLS_BRACE, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '[':
			addToken()
			tokens = append(tokens, Token{Type: OPN_BRACK, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case ']':
			addToken()
			tokens = append(tokens, Token{Type: CLS_BRACK, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case ',':
			addToken()
			tokens = append(tokens, Token{Type: COMMA, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case ';':
			j := i + 1
			for j < len(input) && (input[j] == ' ' || input[j] == '\t') {
				j++
			}
			if j < len(input) && input[j] != '\n' && input[j] != '\r' {
				addToken()
				tokens = append(tokens, Token{Type: SEMICOLON, Lexeme: tkn, Line: line, Column: col})
			}
			i++
			continue

		case '<':
			addToken()
			tokens = append(tokens, Token{Type: LT, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case '>':
			addToken()
			tokens = append(tokens, Token{Type: GT, Lexeme: tkn, Line: line, Column: col})
			i++
			continue

		case ':':
			addToken()
			tokens = append(tokens, Token{Type: DUBLE_DOT, Lexeme: tkn, Line: line, Column: col})
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
			tokens = append(tokens, Token{Type: STRING, Lexeme: s.String(), Line: line, Column: startCol})
			continue
		}

		current.WriteRune(r)
		i++
	}

	addToken()
	return tokens
}

// Helpers
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

func IsLiteral(tok Token) bool {
	switch tok.Type {
	case INT, FLOAT, STRING, BOOL:
		return true
	}
	return false
}

func IsNumericLiteral(tok Token) bool {
	return tok.Type == INT || tok.Type == FLOAT
}

func IsKeyword(tok Token) bool {
	switch tok.Type {
	case PACKAGE, IMPORT, TYPE, STRUCT, FUNC, VAR, CONST,
		IF, ELSE, FOR, CONTINUE, BREAK, RETURN, FALLTHROUGH, GOTO:
		return true
	}
	return false
}

func IsOperator(tok Token) bool {
	switch tok.Type {
	case PLUS, MINUS, STAR, SLASH,
		ASSIGN, DEFINE, EQ, NEQ,
		LT, GT, LTE, GTE,
		AND, AMP, OR, NOT, DOT:
		return true
	}
	return false
}

func IsDelimiter(tok Token) bool {
	switch tok.Type {
	case OPN_PAREN, CLS_PAREN,
		OPN_BRACE, CLS_BRACE,
		OPN_BRACK, CLS_BRACK,
		COMMA, SEMICOLON:
		return true
	}
	return false
}

func (t Token) String() string {
	return fmt.Sprintf(
		"{Type: %s, Value: '%s', File: %s, Line: %d, Column: %d}",
		t.Type, t.Lexeme, "test.fox", t.Line, t.Column,
	)
}

func (tt TokenType) String() string {
	switch tt {
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"

	case FOR:
		return "FOR"
	case IF:
		return "IF"

		// Delimiters
	case OPN_PAREN:
		return "OPN_PAREN"
	case CLS_PAREN:
		return "CLS_PAREN"
	case OPN_BRACE:
		return "OPN_BRACE"
	case CLS_BRACE:
		return "CLS_BRACE"
	case OPN_BRACK:
		return "OPN_BRACK"
	case CLS_BRACK:
		return "CLS_BRACK"
	case COMMA:
		return "COMMA"
	case SEMICOLON:
		return "SEMICOLON"
	case NEW_LINE:
		return "NEW_LINE"

	// Operators
	case PLUS:
		return "PLUS"
	case MINUS:
		return "MINUS"
	case STAR:
		return "STAR"
	case SLASH:
		return "SLASH"
	case ASSIGN:
		return "ASSIGN"
	case DEFINE:
		return "DEFINE"
	case EQ:
		return "EQ"
	case NEQ:
		return "NEQ"
	case LT:
		return "LT"
	case GT:
		return "GT"
	case LTE:
		return "LTE"
	case GTE:
		return "GTE"
	case AND:
		return "AND"
	case AMP:
		return "AMP"
	case OR:
		return "OR"
	case NOT:
		return "NOT"
	case DOT:
		return "DOT"
	case DUBLE_DOT:
		return "DUBLE_DOT"
	case COMMENT:
		return "COMMENT"

	default:
		return "UNKNOWN"
	}
}

// END
