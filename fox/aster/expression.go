package aster

type UnaryExpr struct {
	Op   string // like "*", "&"
	Expr Expression
	Line int
	// Col int later
}

type Expression interface {
	GetLine() int
	isExpr()
}

// UnaryExpr
func (e UnaryExpr) GetLine() int { return e.Line }

// FieldAccessExpr
func (e FieldAccessExpr) GetLine() int { return e.Line }

// CallExpr
func (e CallExpr) GetLine() int { return e.Line }

// NumberExpr
func (e NumberExpr) GetLine() int { return e.Line }

// StringExpr
func (e StringExpr) GetLine() int { return e.Line }

// StructLiteral
func (e StructLiteral) GetLine() int { return e.Line }

// IdentExpr (Don't forget this one)
func (e IdentExpr) GetLine() int { return e.Line }

// BinaryExpr
func (e BinaryExpr) GetLine() int { return e.Line }

type TypeExpr struct {
	Name string
}

func (TypeExpr) isExpr() {}

type NumberExpr struct {
	Literal string
	Line    int
}

func (NumberExpr) isExpr() {}

type StringExpr struct {
	Literal string
	Line    int
}

func (StringExpr) isExpr() {}

type IntExpr struct {
	Literal string
}

func (IntExpr) isExpr() {}

type FloatExpr struct {
	Literal string
	Line    int
}

func (e FloatExpr) GetLine() int { return e.Line }
func (FloatExpr) isExpr()        {}

type BoolExpr struct {
	Literal string
	Line    int
}

func (e BoolExpr) GetLine() int { return e.Line }

func (BoolExpr) isExpr() {}

type LiteralExpr struct {
	Value string
	Line  int
}

func (LiteralExpr) isExpr() {}

type IdentExpr struct {
	Name string
	Line int
}

func (IdentExpr) isExpr() {}

type BinaryExpr struct {
	Op    string
	Left  Expression
	Right Expression
	Line  int
}

func (BinaryExpr) isExpr() {}

type CallExpr struct {
	Callee Expression
	Args   []Expression
	Line   int
}

func (UnaryExpr) isExpr() {}

func (CallExpr) isExpr() {}

type StructLiteral struct {
	Type   Type
	Fields []FieldInit
	Line   int
}

func (StructLiteral) isExpr() {}

type FieldInit struct {
	Name  string
	Value Expression
}

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
	// 1. Parse the left-hand side as a list (e.g., u, o or x.y)
	exprs := p.parseExprList()

	// 2. Check for assignment or definition operators using currentToken()
	tok := p.currentToken()
	if tok.Type == DEFINE || tok.Type == ASSIGN {
		p.pos++ // consume := or =

		// 3. Parse the RHS as a list
		values := p.parseExprList()

		// 4. Handle Short Declaration (:=)
		if tok.Type == DEFINE {
			for _, target := range exprs {
				if !p.isValidDefineTarget(target) {
					p.appendErrorf("non-name on left side of := Line:%d", tok.Line)
				}
			}
			return &Declar{ // Return as pointer
				Names:  exprs,
				Op:     tok.Lexeme,
				Values: values,
				Line:   tok.Line,
			}
		}

		// 5. Handle Normal Assignment (=)
		return &Assign{ // Return as pointer
			Targets: exprs,
			Op:      tok.Lexeme,
			Values:  values,
			Line:    tok.Line,
		}
	}

	// 6. If no operator, it's just an Expression Statement (like a function call)
	// Make sure we have at least one expression
	if len(exprs) == 0 {
		return nil
	}
	return &ExprStmt{Expr: exprs[0], Line: tok.Line}
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
