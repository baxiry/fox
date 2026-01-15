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
		fmt.Println("default")
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
