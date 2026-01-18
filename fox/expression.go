package main

import "fmt"

type Expression interface {
	isExpr()
}

type NumberExpr struct {
	Value string
}

func (NumberExpr) isExpr() {}

type StringExpr struct {
	Value string
}

func (StringExpr) isExpr() {}

type IdentExpr struct {
	Name string
}

func (IdentExpr) isExpr() {}

type BinaryExpr struct {
	Op    string
	Left  Expression
	Right Expression
}

func (BinaryExpr) isExpr() {}

type CallExpr struct {
	Name string
	Args []Expression
}

func (CallExpr) isExpr() {}

func parseExprOrAssign(tokens []Token, pos *int) Statement {

	if lookAheadIsAssign(tokens, *pos) {
		return parseAssign(tokens, pos)
	}

	expr := parseExpr(tokens, pos)
	return ExprStmt{Expr: expr}
}

func parseAssign(tokens []Token, pos *int) Statement {
	// ex: x = expr
	name := expectIdent(tokens, pos).Value
	op := tokens[*pos].Value //
	*pos++

	value := parseExpr(tokens, pos)

	return AssignStmt{
		Name:  name,
		Op:    op,
		Value: value,
	}
}

func parseDefine(tokens []Token, pos *int) Statement {
	// ex: x = expr
	name := expectIdent(tokens, pos).Value
	op := tokens[*pos].Value //
	fmt.Println("op in parseDef", op)
	*pos++

	value := parseExpr(tokens, pos)

	return AssignStmt{
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
