package main

type UnaryExpr struct {
	Op   string // like "*", "&"
	Expr Expression
	Line int
	// Col int later
}

type Expression interface {
	isExpr()
}

type NumberExpr struct {
	Literal string
}

func (NumberExpr) isExpr() {}

type StringExpr struct {
	Literal string
}

func (StringExpr) isExpr() {}

type LiteralExpr struct {
	Value string
}

func (LiteralExpr) isExpr() {}

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

func (UnaryExpr) isExpr() {}

type CallExpr struct {
	Callee Expression
	Args   []Expression
}

func (CallExpr) isExpr() {}

func parseCall(name string, tokens []Token, pos *int) Expression {
	expectType(tokens, pos, OPN_PAREN)

	args := []Expression{}

	for tokens[*pos].Type != CLS_PAREN {
		if tokens[*pos].Type == COMMA {
			*pos++
			continue
		}
		arg := parseExpr(tokens, pos)
		args = append(args, arg)
	}

	expectType(tokens, pos, CLS_PAREN)
	return CallExpr{
		Callee: IdentExpr{Name: name},
		Args:   args,
	}
}

func parseExprOrAssign(tokens []Token, pos *int) Statement {
	if lookAheadIsAssign(tokens, *pos) {
		return parseAssign(tokens, pos)
	}
	expr := parseExpr(tokens, pos)
	return ExprStmt{Expr: expr}
}

func lookAheadIsAssign(tokens []Token, pos int) bool {
	if pos+1 >= len(tokens) {
		return false
	}
	// IDENT = expr
	if tokens[pos].Type == IDENT && tokens[pos+1].Type == ASSIGN {
		return true
	}
	// IDENT := expr
	if tokens[pos].Type == IDENT && tokens[pos+1].Type == DEFINE {
		return true
	}
	return false
}
