package main

type UnaryExpr struct {
	Op   string // "*", "&"
	Expr Expression
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

//type CallExpr struct {
//   Func Expression
//  Args []Expression
//}

type CallExpr struct {
	FuncName string
	Args     []Expression
}

func (CallExpr) isExpr() {}

func parseCall(name string, tokens []Token, pos *int) Expression {
	expectType(tokens, pos, Delimiter.LParen)

	args := []Expression{}

	for tokens[*pos].Value != ")" {
		if tokens[*pos].Value == "," {
			*pos++
			continue
		}
		arg := parseExpr(tokens, pos)
		args = append(args, arg)
	}

	expectType(tokens, pos, Delimiter.RParen)
	return CallExpr{FuncName: name, Args: args}
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
	if tokens[pos].Type == Ident.Ident && tokens[pos+1].Type == Operator.Assign {
		return true
	}
	// IDENT := expr
	if tokens[pos].Type == Ident.Ident && tokens[pos+1].Type == Operator.Define {
		return true
	}
	return false
}
