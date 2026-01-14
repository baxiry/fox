package main

import (
	"strings"
	"unicode"
)

// Token represents a simple token in Fox code
type Token struct {
	Type  string
	Value string

	file   string
	Line   int
	Column int
}

// tokenize splits the input into very simple tokens
func tokenize(input string) []Token {
	var tokens []Token
	var current strings.Builder

	addToken := func() []Token {
		if current.Len() > 0 {
			tokVal := current.String()
			tokType := "IDENT"
			if tokVal == "package" || tokVal == "type" || tokVal == "struct" || tokVal == "func" || tokVal == "return" {
				tokType = "KEYWORD"
			}
			tokens = append(tokens, Token{Type: tokType, Value: tokVal})
			current.Reset()
		}
		return tokens
	}

	for _, r := range input {
		if unicode.IsSpace(r) || r == '{' || r == '}' || r == '(' || r == ')' || r == ':' || r == ',' {
			addToken()
			if !unicode.IsSpace(r) {
				tokens = append(tokens, Token{Type: "SYMBOL", Value: string(r)})
			}
		} else {
			current.WriteRune(r)
		}
	}
	tokens = addToken()

	return tokens
}
