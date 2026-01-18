package main

import "fmt"

type Statement interface {
	isStatement()
}

type ReturnStmt struct {
	Values []Expression
}

func (ReturnStmt) isStatement() {}

type IfStmt struct {
	Cond Expression
	Then []Statement
	Else []Statement
}

func (IfStmt) isStatement() {}

type ForStmt struct {
	Init Statement
	Cond Expression
	Post Statement
	Body []Statement
}

func (ForStmt) isStatement() {}

type AssignStmt struct {
	Name  string
	Op    string // "=" or ":="
	Value Expression
}

func (AssignStmt) isStatement() {}

type DefineStmt struct {
	Name  string
	Value Expression
}

func (DefineStmt) isStatement() {}

type ExprStmt struct {
	Expr Expression
}

func (ExprStmt) isStatement() {}
func parseExprStatement(tokens []Token, pos *int) Statement {
	fmt.Println("parseExprStatement.")
	expr := parseExpr(tokens, pos)
	return ExprStmt{Expr: expr}
}

func parseStatement(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Value {
	case "return":
		return parseReturn(tokens, pos)
	case "if":
		return parseIf(tokens, pos)
	case "for":
		return parseFor(tokens, pos)

	default:
		if *pos+1 >= len(tokens) {
			return parseExprStatement(tokens, pos)
		}

		op := tokens[*pos+1].Value
		if op == "=" {
			return parseAssign(tokens, pos)
		}

		if op == ":=" {
			return parseDefine(tokens, pos)
		}

		return parseExprStatement(tokens, pos)
	}
}

func parseIf(tokens []Token, pos *int) Statement {
	expect(tokens, pos, "if")
	return parseStatement(tokens, pos)
}

func parseFor(tokens []Token, pos *int) Statement {
	expect(tokens, pos, "for")
	return parseStatement(tokens, pos)
}

func parseReturn(tokens []Token, pos *int) Statement {
	expect(tokens, pos, "return")
	values := []Expression{}
	values = append(values, parseExpr(tokens, pos))
	for tokens[*pos].Value == "," {
		*pos++ // consume ','
		values = append(values, parseExpr(tokens, pos))
	}

	return ReturnStmt{Values: values}
}

func parsePackage(tokens []Token, pos *int) string {
	// read "package"
	// read package name
	expect(tokens, pos, "package")
	pkg := tokens[*pos].Value
	return pkg
}

func parseImport(tokens []Token, pos *int) []string {
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

	var libs = []string{}

	for tokens[*pos].Value != ")" {
		pkg := expectIdent(tokens, pos)
		libs = append(libs, pkg.Value)
	}

	expect(tokens, pos, ")") // consume closing brace

	return libs
}

func parseStruct(tokens []Token, pos *int) StructDecl {
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

	fields := []FieldDecl{}
	for tokens[*pos].Value != "}" {
		field := parseField(tokens, pos)
		fields = append(fields, field)
	}

	expect(tokens, pos, "}") // consume closing brace

	return StructDecl{
		Name:   name.Value,
		Fields: fields,
	}
}
func parseRetSign(tokens []Token, pos *int) []ReturnSig {

	var retSigns = []ReturnSig{}

	for tokens[*pos].Value != "{" {

		if tokens[*pos].Value == "," && tokens[*pos+1].Value != "{" {
			*pos++
		}
		tok := expectIdent(tokens, pos)

		retSigns = append(retSigns, ReturnSig{tok.Value, tok.Type})
	}
	return retSigns

}

func parseField(tokens []Token, pos *int) FieldDecl {
	if *pos >= len(tokens) {
		panic("unexpected end of file, expected Ident")
	}
	//ex: a int
	nameTok := expectIdent(tokens, pos)
	typeTok := expectIdent(tokens, pos)
	return FieldDecl{
		Name: nameTok.Value,
		Type: typeTok.Value,
	}
}
