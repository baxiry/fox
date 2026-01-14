package main

type ExpressionNode interface {
	isExpr()
}

type StatementNode interface {
	isStatement()
}

type ReturnNode struct {
	Value ExpressionNode
}

func (ReturnNode) isStatement() {}

type NumberNode struct {
	Value int
}

func (NumberNode) isExpr() {}

func parseStatement(tokens []Token, pos *int) StatementNode {
	stmntNode := ReturnNode{}
	tok := tokens[*pos].Value
	if tok == "return" {
		stmntNode = parseReturn(tokens, pos)
		return stmntNode
	}

	return nil
}

func parseReturn(tokens []Token, pos *int) ReturnNode {
	expect(tokens, pos, "return")
	val := parseExpr(tokens, pos)
	return ReturnNode{Value: val}
}

func parseExpr(tokens []Token, pos *int) ExpressionNode {
	var exprNode ExpressionNode
	expect(tokens, pos, "return")
	for tokens[*pos].Value != "}" {

		exprNode = parseExpr(tokens, pos)
	}

	return exprNode
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

func (CallExprNode) isExpr()   {}
func (BinaryExprNode) isExpr() {}
func (IdentNode) isExpr()      {}
