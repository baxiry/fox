package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type RetSignsNode []string

func parseFunc(tokens []Token, pos *int) FuncNode {
	funcNode := FuncNode{}

	// read "func"
	// read func name
	expect(tokens, pos, "func")
	funcNode.Name = expectIdent(tokens, pos).Value

	// parse function's Params
	// read "("
	// parse params
	// read ")"
	expect(tokens, pos, "(")

	fmt.Println("   parse func params")
	for tokens[*pos].Value != ")" {
		if tokens[*pos].Value == "," {
			*pos++
		}
		name := expectIdent(tokens, pos).Value
		typ := expectIdent(tokens, pos).Value
		param := ParamNode{name, typ}
		funcNode.Params = append(funcNode.Params, param)
	}
	// close func param
	expect(tokens, pos, ")")

	fmt.Println("   parse returnsSign")
	parseRetSign(tokens, pos)

	fmt.Println("   parse budy func")
	expect(tokens, pos, "{")

	tok := tokens[*pos].Value
	for tok != "}" {
		expr := parseExpr(tokens, pos)
		funcNode.Body = append(funcNode.Body, ExprStatementNode{Expr: expr})
		time.Sleep(time.Second / 2)
	}

	return funcNode
}

func isAssign(tokens []Token, pos *int) bool {
	if *pos+1 >= len(tokens) {
		return false
	}
	return tokens[*pos].Type == "IDENT" && tokens[*pos+1].Value == ":="
}

func parsePrimary(tokens []Token, pos *int) ExpressionNode {
	tok := tokens[*pos]
	fmt.Println("tok", tok)
	time.Sleep(time.Second / 2)

	switch tok.Type {
	case "IDENT":
		*pos++
		return IdentExpr{Value: tok.Value}

	case "NUMBER":
		*pos++
		return NumberExpr{Value: tok.Value}

	default:
		panic(fmt.Sprintf(
			"expected expression at %d:%d, got '%s'",
			tok.Line, tok.Column, tok.Value,
		))
	}
}

// AST nodes
func astBuilder(tokens []Token) {
	p := 0
	pos := &p
	ast := &AST{}

	fmt.Println("len of tokens  : ", len(tokens))
	for *pos < len(tokens) {
		token := tokens[*pos]

		switch token.Value {
		case "package":
			ast.PackageName = parsePackage(tokens, pos)

		case "import":
			ast.Imports = parseImport(tokens, pos)

		case "type":
			ast.Structs = append(ast.Structs, parseStruct(tokens, pos))

		case "func":
			ast.Funcs = append(ast.Funcs, parseFunc(tokens, pos))

		default:
			// error or skip
			*pos++
		}
	}
	dump(ast)
}

func expectIdent(tokens []Token, pos *int) Token {
	if *pos >= len(tokens) {
		panic("unexpected end of input, expected identifier")
	}

	tok := tokens[*pos]
	fmt.Println("expectIdent: token is ", tok.Type, tok.Value)

	if tok.Type != "IDENT" {
		panic(fmt.Sprintf(
			"syntax error at line %d, col %d: expected IDENT, got '%s'",
			tok.Line, tok.Column, tok.Value,
		))
	}

	*pos++
	return tok
}

func expect(tokens []Token, pos *int, value string) {
	if *pos >= len(tokens) {
		panic("unexpected end of file, expected " + value)
	}

	tok := tokens[*pos]
	if tok.Value != value {
		// TODO add error posision, line number
		panic(fmt.Sprintf("syntax error At expect func: expected '%s', got '%s'", value, tok.Value))
	}
	*pos++
}

type ParseError struct {
	File   string
	Line   int
	Column int
	Msg    string
}

func dump(ast *AST) {
	data, err := json.MarshalIndent(ast, ".", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
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
