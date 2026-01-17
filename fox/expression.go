package main

import "fmt"

type ExpressionNode interface {
	isExpr()
}

type NumberNode struct {
	Value int
}

type StringExpr struct {
	Value string
}

type IdentNode struct {
	Name string
}

type BinaryExprNode struct {
	Op    string
	Left  ExpressionNode
	Right ExpressionNode
}

type CallExprNode struct {
	Name string
	Args []ExpressionNode
}

type IdentExpr struct {
	Value string
}

type NumberExpr struct {
	Value string
}

func (NumberExpr) isExpr() {}
func (StringExpr) isExpr() {}

func (NumberNode) isExpr()     {}
func (CallExprNode) isExpr()   {}
func (BinaryExprNode) isExpr() {}
func (IdentNode) isExpr()      {}
func (IdentExpr) isExpr()      {}

func parseExprStatement(tokens []Token, pos *int) StatementNode {
	fmt.Println("parseExprStatement.")
	expr := parseExpr(tokens, pos)
	return ExprStatementNode{Expr: expr}
}

func parseExprOrAssign(tokens []Token, pos *int) StatementNode {

	if lookAheadIsAssign(tokens, *pos) {
		return parseAssign(tokens, pos)
	}

	expr := parseExpr(tokens, pos)
	return ExprStatementNode{Expr: expr}
}

func parseAssign(tokens []Token, pos *int) StatementNode {
	// ex : x = expr
	name := expectIdent(tokens, pos).Value
	op := tokens[*pos].Value //
	*pos++

	value := parseExpr(tokens, pos)

	return AssignNode{
		Name:  name,
		Op:    op,
		Value: value,
	}
}

func lookAheadIsAssign(tokens []Token, pos int) bool {
	if pos+1 >= len(tokens) {
		return false
	}
	// ex: IDENT "="
	if tokens[pos].Type == "IDENT" && tokens[pos+1].Value == "=" {
		return true
	}
	// ex: IDENT ":="
	if tokens[pos].Type == "IDENT" && tokens[pos+1].Value == ":=" {
		return true
	}
	return false
}
