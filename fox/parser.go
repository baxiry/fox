package main

import "fmt"

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

type ImportNode []string

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
		panic(fmt.Sprintf(
			"syntax error at line %d, col %d: expected '%s', got '%s'",
			tok.Line, tok.Column, value, tok.Value,
		))
	}
	*pos++
}

type AST struct {
	PackageName string
	Imports     []string
	Structs     []StructNode
	Funcs       []FuncNode
}

type StructNode struct {
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	Name string
	Type string
}

type FuncNode struct {
	Name    string
	Params  []ParamNode
	Returns []ReturnNode
	Body    string // keep body as raw string
}

type ParamNode struct {
	Name string
	Type string
}

func parsePackage(tokens []Token, pos *int) string {
	// read "package"
	// read package name
	*pos++
	pkg := tokens[*pos].Value
	return pkg
}
