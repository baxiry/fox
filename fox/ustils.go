package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ParseError struct {
	File   string
	Line   int
	Column int
	Msg    string
}

func dump(ast *AST) {
	data, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
func readNumber(src string, pos *int) Token {
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

	// ---- new rule ----
	if isFloat && (strings.HasPrefix(suffix, "i") || strings.HasPrefix(suffix, "u")) {
		panic(fmt.Sprintf("invalid numeric literal: float cannot have integer suffix: %s%s", value, suffix))
	}
	// -------------------

	return Token{
		Type:  NumericLiteral.Float,
		Value: value,
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

/*
func TrackError() {
	if r := recover(); r != nil {
		if e, ok := r.(ParseError); ok {
			fmt.Printf("%s:%d:%d: %s\n", e.File, e.Line, e.Column, e.Msg)
		} else {
			panic(r)
		}
	}
}()
*/
