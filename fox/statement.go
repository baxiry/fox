package main

import "fmt"

type StatementNode interface {
	isStatement()
}

type ExprStatementNode struct {
	Expr ExpressionNode
}

type AssignNode struct {
	Name  string
	Op    string // "=", ":=", ...
	Value ExpressionNode
}

func (ExprStatementNode) isStatement() {}
func (AssignNode) isStatement()        {}
func (ReturnNode) isStatement()        {}

func parseStatement(tokens []Token, pos *int) StatementNode {
	switch tokens[*pos].Value {
	case "return":
		return parseReturn(tokens, pos)
	case "if":
		return parseIf(tokens, pos)
	case "for":
		return parseFor(tokens, pos)
	default:
		if tokens[*pos+1].Value == "=" {
			return parseAssign(tokens, pos)
		}
		return parseExprStatement(tokens, pos)
	}
}

func parseIf(tokens []Token, pos *int) ReturnNode {
	expect(tokens, pos, "if")
	val := parseExpr(tokens, pos)
	return ReturnNode{Value: val}
}
func parseFor(tokens []Token, pos *int) ReturnNode {
	expect(tokens, pos, "for")
	val := parseExpr(tokens, pos)
	return ReturnNode{Value: val}
}
func parseReturn(tokens []Token, pos *int) ReturnNode {
	expect(tokens, pos, "return")
	val := parseExpr(tokens, pos)
	return ReturnNode{Value: val}
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
func parseRetSign(tokens []Token, pos *int) []RetSignsNode {

	var retSigns = []RetSignsNode{}

	for tokens[*pos].Value != "{" {
		tok := expectIdent(tokens, pos)
		fmt.Println(" tok", tok)
		retSigns = append(retSigns, RetSignsNode{tok.Value, tok.Type})
	}
	return retSigns

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
