package main

import (
	"strings"
	"unicode"
)

// TODO: disambiguate '*' between pointer and multiplication

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

	var line = 1

	addToken := func() {
		if current.Len() == 0 {
			return
		}
		val := current.String()
		typ := "IDENT"

		// NUMBER
		if isNumber(val) {
			typ = "NUMBER"
		}

		// KEYWORD
		switch val {
		case "package", "type", "struct", "func", "return", "var", "const", "if", "for":
			typ = "KEYWORD"
		}

		tokens = append(tokens, Token{Type: typ, Value: val, Line: line})
		current.Reset()
	}

	i := 0
	for i < len(input) {
		if string(input[i]) == "\n" {
			line++
		}
		r := rune(input[i])

		// Spaces
		if unicode.IsSpace(r) {
			addToken()
			i++
			continue
		}

		// OPERATOR
		if i+1 < len(input) {
			two := input[i : i+2]
			if two == ":=" || two == "+=" || two == "-=" || two == "*=" || two == "/=" || two == "==" || two == "!=" {
				addToken()
				tokens = append(tokens, Token{Type: "OPERATOR", Value: two, Line: line})
				i += 2
				continue
			}
		}

		// SYMBOLS
		switch r {
		case '{', '}', '(', ')', ',', ';':
			addToken()
			tokens = append(tokens, Token{Type: "SYMBOL", Value: string(r), Line: line})
			i++
			continue
		case '=', '+', '-', '*', '/', '<', '>', '!':
			addToken()
			tokens = append(tokens, Token{Type: "OPERATOR", Value: string(r), Line: line})
			i++
			continue
		case '"':
			// STRINGS
			addToken()
			i++
			var s strings.Builder
			for i < len(input) && input[i] != '"' {
				s.WriteByte(input[i])
				i++
			}
			i++ // skip this symbol: ".
			tokens = append(tokens, Token{Type: "STRING", Value: s.String(), Line: line})
			continue
		}

		// part of name or number
		current.WriteRune(r)
		i++
	}

	addToken()
	return tokens
}

func isNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}
