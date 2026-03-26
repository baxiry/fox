package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
)

type ParseError struct {
	File   string
	Line   int
	Column int
	Msg    string
}

func Dump(v interface{}) {
	dumpValue(reflect.ValueOf(v), 0)
}

func dumpValue(val reflect.Value, indent int) {
	ind := strings.Repeat("  ", indent)

	if !val.IsValid() {
		fmt.Print("nil")
		return
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			fmt.Print("nil")
			return
		}
		dumpValue(val.Elem(), indent)

	case reflect.Struct:
		typ := val.Type()
		if val.NumField() == 0 {
			fmt.Print(typ.Name(), " {}")
			return
		}
		// Decide inline: small struct with all simple fields
		inline := val.NumField() <= 2
		if inline {
			fmt.Print(typ.Name(), " { ")
			for i := 0; i < val.NumField(); i++ {
				field := typ.Field(i)
				fmt.Print(field.Name, ": ")
				dumpValue(val.Field(i), indent)
				if i < val.NumField()-1 {
					fmt.Print(", ")
				}
			}
			fmt.Print(" }")
		} else {
			fmt.Println(typ.Name(), "{")
			for i := 0; i < val.NumField(); i++ {
				field := typ.Field(i)
				fmt.Print(strings.Repeat("  ", indent+1), field.Name, ": ")
				dumpValue(val.Field(i), indent+1)
				fmt.Println()
			}
			fmt.Print(ind, "}")
		}

	case reflect.Slice:
		if val.Len() == 0 {
			fmt.Print("[]")
			return
		}
		fmt.Println("[") // non-empty slice → new line
		for i := 0; i < val.Len(); i++ {
			fmt.Print(strings.Repeat("  ", indent+1))
			dumpValue(val.Index(i), indent+1)
			fmt.Println()
		}
		fmt.Print(ind, "]")

	case reflect.String:
		fmt.Print(val.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Print(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Print(val.Uint())
	case reflect.Bool:
		fmt.Print(val.Bool())
	default:
		fmt.Print(val.Interface())
	}
}

var enc = json.NewEncoder(os.Stdout)

func dmp(ast *AST) {
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(ast)
	if err != nil {
		panic(err)
	}
	//fmt.Println(ast)
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
