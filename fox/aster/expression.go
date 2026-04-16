package aster

type UnaryExpr struct {
	Op   string // like "*", "&"
	Expr Expression
	Line int
	// Col int later
}

type Expression interface {
	isExpr()
}

type TypeExpr struct {
	Name string
}

type NumberExpr struct {
	Literal string
}

type StringExpr struct {
	Literal string
}

type LiteralExpr struct {
	Value string
}

type IdentExpr struct {
	Name string
}

type BinaryExpr struct {
	Op    string
	Left  Expression
	Right Expression
}

type CallExpr struct {
	Callee Expression
	Args   []Expression
}

func (CallExpr) isExpr() {}

func (TypeExpr) isExpr()    {}
func (NumberExpr) isExpr()  {}
func (StringExpr) isExpr()  {}
func (LiteralExpr) isExpr() {}
func (IdentExpr) isExpr()   {}
func (BinaryExpr) isExpr()  {}
func (UnaryExpr) isExpr()   {}

func (p *Parser) parseCall(name string) CallExpr {
	p.expectType(OPN_PAREN)

	args := []Expression{}

	for p.tokens[p.pos].Type != CLS_PAREN {
		if p.tokens[p.pos].Type == COMMA {
			p.pos++
			continue
		}
		arg := p.parseExpr()
		args = append(args, arg)
	}

	p.expectType(CLS_PAREN)
	return CallExpr{
		Callee: IdentExpr{Name: name},
		Args:   args,
	}
}

func (p *Parser) parseExprOrAssign() Statement {
	target := p.parseBinary(0)

	if p.pos < len(p.tokens) {
		switch p.tokens[p.pos].Type {

		case ASSIGN:
			p.pos++
			value := p.parseExpr()
			return Assign{
				Target: target,
				Op:     "=",
				Value:  value,
			}

		case DEFINE:
			p.pos++
			ident, ok := target.(IdentExpr)
			if !ok {
				panic("left side of := must be identifier")
			}
			value := p.parseExpr()
			return Declar{
				Name:  ident,
				Value: value,
				Op:    DEFINE.String(),
			}
		}
	}

	return ExprStmt{Expr: target}
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
