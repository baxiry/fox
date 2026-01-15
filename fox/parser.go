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
		//stmt := parseExpr(tokens, pos)
		parseExpr(tokens, pos)
		//	funcNode.Body = append(funcNode.Body, stmt)
		time.Sleep(time.Second / 2)
	}

	return funcNode
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

func parsePackage(tokens []Token, pos *int) string {
	// read "package"
	// read package name
	expect(tokens, pos, "package")
	pkg := tokens[*pos].Value
	return pkg
}

func parseImport(tokens []Token, pos *int) ImportNode {
	/*
		read import
		read (
		read "fmt"
		my read ,
		read "io"
		read )
	*/
	expect(tokens, pos, "import")
	expect(tokens, pos, "(")

	var libs = ImportNode{}

	for tokens[*pos].Value != ")" {
		pkg := expectIdent(tokens, pos)
		libs = append(libs, pkg.Value)
	}

	expect(tokens, pos, ")") // consume closing brace

	return libs
}

func parseStruct(tokens []Token, pos *int) StructNode {
	/*
	   expect "type" or "struct"
	   read "struct"
	   read struct name
	   read "{"
	   parse fields
	   read "}"

	   type X struct {
	       a int
	       b int
	   }
	*/
	expect(tokens, pos, "type")

	name := expectIdent(tokens, pos)

	expect(tokens, pos, "struct")
	expect(tokens, pos, "{")

	fields := []FieldNode{}
	for tokens[*pos].Value != "}" {
		field := parseField(tokens, pos)
		fields = append(fields, field)
	}

	expect(tokens, pos, "}") // consume closing brace

	return StructNode{
		Name:   name.Value,
		Fields: fields,
	}
}

func parseField(tokens []Token, pos *int) FieldNode {
	if *pos >= len(tokens) {
		panic("unexpected end of file, expected Ident")
	}
	//ex: a int
	nameTok := expectIdent(tokens, pos)
	typeTok := expectIdent(tokens, pos)
	return FieldNode{
		Name: nameTok.Value,
		Type: typeTok.Value,
	}
}
func parseRetSign(tokens []Token, pos *int) []RetSignsNode {

	var retSigns = []RetSignsNode{}

	for tokens[*pos].Value != "{" {
		tok := expectIdent(tokens, pos)
		fmt.Println(" tok", tok)
		retSigns = append(retSigns, RetSignsNode{tok.Value, tok.Type})
	}
	return retSigns

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
