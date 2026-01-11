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
			parseImport(tokens, pos)
		case "struct":
			ast.Structs = append(ast.Structs, parseStruct(tokens, pos))

		case "func":
			ast.Funcs = append(ast.Funcs, parseFunc(tokens, pos))

		default:
			// error or skip
			*pos++
		}
	}
	fmt.Printf("%#v\n", ast)
	fmt.Println(ast)
}

type AST struct {
	PackageName string
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
	Name   string
	Params []ParamNode
	Body   string // keep body as raw string
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

func parseImport(tokens []Token, pos *int) string {
	// read "package"
	// read package name

	*pos++
	return "fmt"
}

func parseStruct(tokens []Token, pos *int) StructNode {
	// read "struct"
	// read struct name
	// read "{"
	// parse fields
	// read "}"

	*pos++
	return StructNode{}
}

func parseFunc(tokens []Token, pos *int) FuncNode {
	// read "func"
	// read func name
	// read "("
	// parse params
	// read ")"
	// read "{"
	// parse body
	// read "}"

	*pos++
	return FuncNode{}
}
func parseStatement(tokens []Token, pos *int) StatementNode {
	// detect statement type by lookahead

	*pos++
	return StatementNode{}
}

type StatementNode struct{}
