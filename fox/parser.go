package main

import (
	"fmt"
)

// ================= Expressions =================

// parse unary Operator: *p
func parseUnary(tokens []Token, pos *int) Expression {
	if tokens[*pos].Value == "*" {
		*pos++
		expr := parseUnary(tokens, pos)

		return UnaryExpr{Op: "*", Expr: expr}
	}
	return parsePrimary(tokens, pos)
}

// parse * and /
func parseMul(tokens []Token, pos *int) Expression {
	left := parseUnary(tokens, pos)
	for tokens[*pos].Value == "*" || tokens[*pos].Value == "/" {
		op := tokens[*pos]
		*pos++
		right := parseUnary(tokens, pos)
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

func parseEquality(tokens []Token, pos *int) Expression {
	left := parseAdd(tokens, pos)

	for tokens[*pos].Value == "==" || tokens[*pos].Value == "!=" {
		op := tokens[*pos]
		*pos++
		right := parseAdd(tokens, pos)

		left = BinaryExpr{
			Left:  left,
			Op:    op,
			Right: right,
		}
	}

	return left
}

// parse + and -
func parseAdd(tokens []Token, pos *int) Expression {
	left := parseMul(tokens, pos)
	for tokens[*pos].Value == "+" || tokens[*pos].Value == "-" {
		op := tokens[*pos]
		*pos++
		right := parseMul(tokens, pos)
		left = BinaryExpr{Left: left, Op: op, Right: right}
	}
	return left
}

// top-level expression
func parseExpr(tokens []Token, pos *int) Expression {
	// TODO: need && ||. ex: parseLogic()
	return parseEquality(tokens, pos)
}

// primary expressions
func parsePrimary(tokens []Token, pos *int) Expression {
	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing expression")
	}

	tok := tokens[*pos]

	switch tok.Type {

	case Ident.Ident:
		*pos++

		// function call: f(...)
		if *pos < len(tokens) && tokens[*pos].Type == Delimiter.LParen {
			return parseCall(tok.Value, tokens, pos)
		}
		return IdentExpr{Name: tok.Value}

	case NumericLiteral.Int, NumericLiteral.Float:
		*pos++
		return NumberExpr{Literal: tok.Value}

	case OtherLiteral.String:
		*pos++
		return StringExpr{Literal: tok.Value}

	case Delimiter.LParen: //TOKEN_LPAREN:
		*pos++
		expr := parseExpr(tokens, pos)
		expectType(tokens, pos, Delimiter.LParen)
		return expr

	default:
		panic(fmt.Sprintf(
			"expected expression at line %d, got %s (%q)",
			tok.Line, tok.Type, tok.Value,
		))
	}
}

// ================= Functions =================

func parseFunc(tokens []Token, pos *int) FuncDecl {
	funcNode := FuncDecl{}

	// func
	expectType(tokens, pos, keywords.Func)
	funcNode.Name = expectIdent(tokens, pos).Value

	// (
	expectType(tokens, pos, Delimiter.LParen)

	for tokens[*pos].Type != Delimiter.RParen {
		// skip comma
		if tokens[*pos].Type == Delimiter.Comma {
			*pos++
			continue
		}

		// param name
		name := expectIdent(tokens, pos).Value

		// param type
		typ := ""
		if tokens[*pos].Type == Operator.Star {
			*pos++
			typ = "*" + expectIdent(tokens, pos).Value
		} else {
			typ = expectIdent(tokens, pos).Value
		}

		funcNode.Params = append(funcNode.Params, ParamDecl{
			Name: name,
			Type: typ,
		})
	}

	// )
	expectType(tokens, pos, Delimiter.RParen)

	// return signature
	funcNode.Returns = parseRetSign(tokens, pos)

	// {
	expectType(tokens, pos, Delimiter.LBrace)

	for tokens[*pos].Type != Delimiter.RBrace {
		stmt := parseStatement(tokens, pos)
		funcNode.Body = append(funcNode.Body, stmt)
	}

	// }
	expectType(tokens, pos, Delimiter.RBrace)

	return funcNode
}

// ================= AST Builder =================

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
			*pos++
		}
	}
	dump(ast)
}

// ===== Top-Level Parsers =====

func parsePackage(tokens []Token, pos *int) string {
	expectType(tokens, pos, keywords.Package)
	pkg := tokens[*pos].Value
	*pos++
	return pkg
}

func parseImport(tokens []Token, pos *int) []string {
	expectType(tokens, pos, keywords.Import)
	expectType(tokens, pos, Delimiter.LParen)

	libs := []string{}
	for tokens[*pos].Value != ")" {
		pkg := expectIdent(tokens, pos)
		libs = append(libs, pkg.Value)
	}
	expectType(tokens, pos, Delimiter.RParen)
	return libs
}

func parseStruct(tokens []Token, pos *int) StructDecl {
	expectType(tokens, pos, keywords.Type)
	name := expectIdent(tokens, pos)
	expectType(tokens, pos, keywords.Struct)
	expectType(tokens, pos, Delimiter.LBrace)

	fields := []FieldDecl{}
	for tokens[*pos].Value != "}" {
		fields = append(fields, parseField(tokens, pos))
	}
	expectType(tokens, pos, Delimiter.RBrace)

	return StructDecl{Name: name.Value, Fields: fields}
}

func parseField(tokens []Token, pos *int) FieldDecl {
	nameTok := expectIdent(tokens, pos)
	typeTok := expectIdent(tokens, pos)
	return FieldDecl{Name: nameTok.Value, Type: typeTok.Value}
}
