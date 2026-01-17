package main

import (
	"fmt"
	"time"
)

// ================= Expressions =================

// parse unary Operator: *p
func parseUnary(tokens []Token, pos *int) ExpressionNode {
	if tokens[*pos].Value == "*" {
		*pos++
		expr := parseUnary(tokens, pos)
		return expr // later: DerefExpr{Expr: expr}
	}
	return parsePrimary(tokens, pos)
}

// parse * and /
func parseMul(tokens []Token, pos *int) ExpressionNode {
	left := parseUnary(tokens, pos)
	for tokens[*pos].Value == "*" || tokens[*pos].Value == "/" {
		op := tokens[*pos].Value
		*pos++
		right := parseUnary(tokens, pos)
		left = BinaryExprNode{Left: left, Op: op, Right: right}
	}
	return left
}

// parse + and -
func parseAdd(tokens []Token, pos *int) ExpressionNode {
	left := parseMul(tokens, pos)
	for tokens[*pos].Value == "+" || tokens[*pos].Value == "-" {
		op := tokens[*pos].Value
		*pos++
		right := parseMul(tokens, pos)
		left = BinaryExprNode{Left: left, Op: op, Right: right}
	}
	return left
}

// top-level expression
func parseExpr(tokens []Token, pos *int) ExpressionNode {
	return parseAdd(tokens, pos)
}

// primary expressions
func parsePrimary(tokens []Token, pos *int) ExpressionNode {
	tok := tokens[*pos]

	switch tok.Type {
	case "IDENT":
		*pos++
		return IdentExpr{Value: tok.Value}

	case "NUMBER":
		*pos++
		return NumberExpr{Value: tok.Value}

	case "STRING":
		*pos++
		return StringExpr{Value: tok.Value}

	case "(":
		*pos++
		expr := parseExpr(tokens, pos)
		expect(tokens, pos, ")")
		return expr

	default:
		panic(fmt.Sprintf(
			"expected expression at %d:%d, got '%s'",
			tok.Line, tok.Column, tok.Value,
		))
	}
}

// ================= Functions =================

func parseFunc(tokens []Token, pos *int) FuncNode {
	funcNode := FuncNode{}

	expect(tokens, pos, "func")
	funcNode.Name = expectIdent(tokens, pos).Value

	expect(tokens, pos, "(")
	for tokens[*pos].Value != ")" {
		if tokens[*pos].Value == "," {
			*pos++
		}
		name := expectIdent(tokens, pos).Value

		typ := ""
		if tokens[*pos].Value == "*" {
			*pos++
			typ = "*" + expectIdent(tokens, pos).Value
		} else {
			typ = expectIdent(tokens, pos).Value
		}

		funcNode.Params = append(funcNode.Params, ParamNode{name, typ})
	}
	expect(tokens, pos, ")")

	parseRetSign(tokens, pos)

	expect(tokens, pos, "{")
	for tokens[*pos].Value != "}" {
		//expr := parseExpr(tokens, pos)
		stmt := parseStatement(tokens, pos)
		funcNode.Body = append(funcNode.Body, stmt)
		time.Sleep(time.Millisecond / 350)
	}
	expect(tokens, pos, "}")

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

// ================= Utilities =================

func expectIdent(tokens []Token, pos *int) Token {
	if *pos >= len(tokens) {
		panic("   unexpected end of input, expected identifier")
	}

	tok := tokens[*pos]
	fmt.Println("expectIdent: ", tok)

	if tok.Type != "IDENT" {
		panic(fmt.Sprintf(
			"   syntax error at line %d, col %d: expected IDENT, got '%s'\n\n",
			tok.Line, tok.Column, tok.Type,
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
	fmt.Println("expect:      ", tok)
	if tok.Value != value {
		panic(fmt.Sprintf("syntax error: expected '%s', got '%s'", value, tok.Value))
	}
	*pos++
}

func isAssign(tokens []Token, pos *int) bool {
	if *pos+1 >= len(tokens) {
		return false
	}
	return tokens[*pos].Type == "IDENT" && tokens[*pos+1].Value == ":="
}
