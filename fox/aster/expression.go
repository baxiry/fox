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

type StructLiteral struct {
	Type   Type
	Fields []FieldInit
}

type FieldInit struct {
	Name  string
	Value Expression
}

func (CallExpr) isExpr()      {}
func (StructLiteral) isExpr() {}
func (TypeExpr) isExpr()      {}
func (NumberExpr) isExpr()    {}
func (StringExpr) isExpr()    {}
func (LiteralExpr) isExpr()   {}
func (IdentExpr) isExpr()     {}
func (BinaryExpr) isExpr()    {}
func (UnaryExpr) isExpr()     {}

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
	// 1. Try to parse a list of expressions (LHS)
	exprs := p.parseExprList()

	// 2. Check for assignment or definition operators
	if p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		if tok.Type == DEFINE || tok.Type == ASSIGN {
			p.pos++ // consume := or =

			// 3. Parse the RHS as a list
			values := p.parseExprList()

			return Declar{
				Names:  exprs,
				Op:     tok.Lexeme,
				Values: values,
			}
		}
	}

	// 4. If no operator, return as an Expression Statement
	return ExprStmt{Expr: exprs[0]}
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
