package aster

import (
	"fmt"
	"strings"
	"unicode"
)

type TokenType int

const (
	INVALID TokenType = iota

	// Identifiers & literals
	IDENT
	INT
	FLOAT
	STRING
	BOOL

	// Keywords
	PACKAGE
	IMPORT
	STRUCT
	TYPE
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

	ENUM
	SWITCH
	MATCH
	CASE
	//END_KEYWORDS

	// Operators
	PLUS
	MINUS
	STAR
	SLASH
	ASSIGN
	DEFINE
	PLUS_ASSIGN
	PLUS_DEFINE
	PLUS_PLUS
	EQ
	NEQ
	LT
	GT
	LTE
	GTE
	AND
	AMP
	OR
	EXCLAM
	DOT
	PIPE

	// Delimiters
	OPN_PAREN
	CLS_PAREN
	OPN_BRACE
	CLS_BRACE
	OPN_BRACK
	CLS_BRACK
	COMMA
	SEMICOLON
	COLON
	NEW_LINE
	COMMENT

	TRUE
	FALSE
	SPAWN
	BLANK

	// Special
	EOF
	ILLEGAL
	ERROR
)

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}

var keywords = map[string]TokenType{
	"package":     PACKAGE,
	"struct":      STRUCT,
	"var":         VAR,
	"fn":          FUNC,
	"goto":        GOTO,
	"return":      RETURN,
	"const":       CONST,
	"if":          IF,
	"fallthrough": FALLTHROUGH,
	"else":        ELSE,
	"for":         FOR,
	"import":      IMPORT,
	"break":       BREAK,
	"continue":    CONTINUE,
	"spawn":       SPAWN,
	"true":        BOOL,
	"false":       BOOL,
	"enum":        ENUM,
	"switch":      SWITCH,
	"match":       MATCH,
	"case":        CASE,
}

// LexerState holds the clean isolated environment for lexical analysis
type LexerState struct {
	tokens  []Token
	current strings.Builder
	line    int
	col     int
}

// addToken is now completely isolated as a method of LexerState
func (ls *LexerState) addToken() {
	if ls.current.Len() == 0 {
		return
	}
	val := ls.current.String()
	ls.current.Reset()

	// Keywords Lookup
	if tt, ok := keywords[val]; ok {
		ls.tokens = append(ls.tokens, Token{Type: tt, Lexeme: val, Line: ls.line, Column: ls.col})
		return
	}

	if isInt(val) {
		ls.tokens = append(ls.tokens, Token{Type: INT, Lexeme: val, Line: ls.line, Column: ls.col})
		return
	}
	if isFloat(val) {
		ls.tokens = append(ls.tokens, Token{Type: FLOAT, Lexeme: val, Line: ls.line, Column: ls.col})
		return
	}

	// Split identifiers by dot cleanly
	parts := strings.Split(val, ".")
	for i, p := range parts {
		if p == "" {
			continue
		}
		ls.tokens = append(ls.tokens, Token{Type: IDENT, Lexeme: p, Line: ls.line, Column: ls.col})
		if i < len(parts)-1 {
			ls.tokens = append(ls.tokens, Token{Type: DOT, Lexeme: ".", Line: ls.line, Column: ls.col})
		}
	}
}

// Lexer function remains the public entry point but uses the isolated state engine
func Lexer(input string) []Token {
	ls := &LexerState{
		tokens: make([]Token, 0),
		line:   1,
		col:    0,
	}

	i := 0
	for i < len(input) {
		r := rune(input[i])
		ls.col++

		if r == '\n' {
			ls.addToken()

			if len(ls.tokens) == 0 || ls.tokens[len(ls.tokens)-1].Type != NEW_LINE {
				ls.tokens = append(ls.tokens, Token{
					Type:   NEW_LINE,
					Lexeme: "\n",
					Line:   ls.line,
					Column: ls.col,
				})
			}

			ls.line++
			ls.col = 0
			i++
			continue
		}
		if unicode.IsSpace(r) {
			ls.addToken()
			i++
			continue
		}

		// Multi-char operators logic
		r = rune(input[i])
		if i+1 < len(input) {
			next := input[i+1]
			matched := true

			switch {
			case r == '/' && next == '/':
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue

			case r == '+' && next == '+':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: PLUS_PLUS, Lexeme: "++", Line: ls.line, Column: ls.col})

			case r == '>' && next == '=':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: GTE, Lexeme: ">=", Line: ls.line, Column: ls.col})

			case r == '<' && next == '=':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: LTE, Lexeme: "<=", Line: ls.line, Column: ls.col})

			case r == ':' && next == '=':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: DEFINE, Lexeme: ":=", Line: ls.line, Column: ls.col})

			case r == '=' && next == '=':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: EQ, Lexeme: "==", Line: ls.line, Column: ls.col})

			case r == '!' && next == '=':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: NEQ, Lexeme: "!=", Line: ls.line, Column: ls.col})

			case r == '&' && next == '&':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: AND, Lexeme: "&&", Line: ls.line, Column: ls.col})

			case r == '|' && next == '|':
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: OR, Lexeme: "||", Line: ls.line, Column: ls.col})

			default:
				matched = false
			}

			if matched {
				i += 2
				ls.col += 2
				continue
			}
		}

		tkn := string(r)

		switch r {

		case '&':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: AMP, Lexeme: string(r), Line: ls.line, Column: ls.col})
			i++
			continue

		case '|':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: PIPE, Lexeme: string(r), Line: ls.line, Column: ls.col})
			i++
			continue

		case '=':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: ASSIGN, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '+', '-':
			if ls.current.Len() > 0 {
				val := ls.current.String()
				ls.current.Reset()

				if isInt(val) {
					ls.tokens = append(ls.tokens, Token{Type: INT, Lexeme: val, Line: ls.line, Column: ls.col})
				} else if isFloat(val) {
					ls.tokens = append(ls.tokens, Token{Type: FLOAT, Lexeme: val, Line: ls.line, Column: ls.col})
				} else {
					ls.tokens = append(ls.tokens, Token{Type: IDENT, Lexeme: val, Line: ls.line, Column: ls.col})
				}
			}

			if i+1 < len(input) && unicode.IsDigit(rune(input[i+1])) &&
				(len(ls.tokens) == 0 || ls.tokens[len(ls.tokens)-1].Type != IDENT) {
				ls.current.WriteRune(r)
				i++
				ls.col++
				continue
			}

			if r == '+' {
				ls.tokens = append(ls.tokens, Token{Type: PLUS, Lexeme: tkn, Line: ls.line, Column: ls.col})
			} else {
				ls.tokens = append(ls.tokens, Token{Type: MINUS, Lexeme: tkn, Line: ls.line, Column: ls.col})
			}
			i++
			continue

		case '*':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: STAR, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '!':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: EXCLAM, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue
		case '/':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: SLASH, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '(':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: OPN_PAREN, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case ')':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: CLS_PAREN, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '{':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: OPN_BRACE, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '}':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: CLS_BRACE, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '[':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: OPN_BRACK, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case ']':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: CLS_BRACK, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case ',':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: COMMA, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case ';':
			j := i + 1
			for j < len(input) && (input[j] == ' ' || input[j] == '\t') {
				j++
			}
			if j < len(input) && input[j] != '\n' && input[j] != '\r' {
				ls.addToken()
				ls.tokens = append(ls.tokens, Token{Type: SEMICOLON, Lexeme: tkn, Line: ls.line, Column: ls.col})
			}
			i++
			continue

		case '<':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: LT, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '>':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: GT, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case ':':
			ls.addToken()
			ls.tokens = append(ls.tokens, Token{Type: COLON, Lexeme: tkn, Line: ls.line, Column: ls.col})
			i++
			continue

		case '"':
			ls.addToken()
			i++
			startCol := ls.col
			var s strings.Builder
			for i < len(input) && input[i] != '"' {
				s.WriteByte(input[i])
				i++
				ls.col++
			}
			i++
			ls.tokens = append(ls.tokens, Token{Type: STRING, Lexeme: s.String(), Line: ls.line, Column: startCol})
			continue
		}

		ls.current.WriteRune(r)
		i++
	}

	ls.addToken()
	ls.tokens = append(ls.tokens, Token{
		Type:   EOF,
		Lexeme: "EOF",
		Line:   ls.line,
		Column: ls.col,
	})

	//for k, v := range tokens {fmt.Print(k, ": ", v.Lexeme, ", ")}
	return ls.tokens
}

func (t TokenType) String() string {
	switch t {
	case INVALID:
		return "INVALID"
	case IDENT:
		return "IDENT"
	case INT:
		return "INT"
	case FLOAT:
		return "FLOAT"
	case STRING:
		return "STRING"
	case BOOL:
		return "BOOL"
	case PACKAGE:
		return "PACKAGE"
	case IMPORT:
		return "IMPORT"
	case STRUCT:
		return "STRUCT"
	case TYPE:
		return "TYPE"
	case ENUM:
		return "ENUM"
	case FUNC:
		return "FUNC"
	case VAR:
		return "VAR"
	case CONST:
		return "CONST"
	case IF:
		return "IF"
	case ELSE:
		return "ELSE"
	case FOR:
		return "FOR"
	case BREAK:
		return "BREAK"
	case CONTINUE:
		return "CONTINUE"
	case RETURN:
		return "RETURN"
	case FALLTHROUGH:
		return "FALLTHROUGH"
	case GOTO:
		return "GOTO"
	case SWITCH:
		return "SWITCH"
	case MATCH:
		return "MATCH"
	case CASE:
		return "CASE"
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
	case PLUS_PLUS:
		return "PLUS_PLUS"
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
	case EXCLAM:
		return "EXCLAM"
	case DOT:
		return "DOT"
	case PIPE:
		return "PIPE"
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
	case COLON:
		return "COLON"
	case NEW_LINE:
		return "NEW_LINE"
	case EOF:
		return "EOF"
	default:
		return "UNKNOWN"
	}
}

func (t Token) String() string {
	return fmt.Sprintf(
		"{Type: %s, Value: '%s', File: %s, Line: %d, Column: %d}",
		t.Type, t.Lexeme, "test.fox", t.Line, t.Column,
	)
}

/////////// Helpers //////////////////

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
	case PACKAGE, IMPORT, STRUCT, ENUM, TYPE, FUNC, VAR, CONST,
		IF, ELSE, FOR, CONTINUE, BREAK, RETURN, FALLTHROUGH, GOTO:
		return true
	}
	return false
}

func IsOperator(tok Token) bool {
	switch tok.Type {
	case PLUS, PLUS_PLUS, MINUS, STAR,
		ASSIGN, DEFINE, EQ, NEQ, PIPE,
		SLASH, LT, GT, LTE, GTE,
		AND, AMP, OR, EXCLAM, DOT:
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

// END
