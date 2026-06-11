package aster

import "fox/symbols"

type UnaryExpr struct {
	Op   string // like "*", "&"
	Expr Expression
	Line int
	// Col int later
}

// UnaryExpr
func (e *UnaryExpr) GetLine() int { return e.Line }

// CallExpr
func (e *CallExpr) GetLine() int { return e.Line }

// StringExpr
func (e *StringExpr) GetLine() int { return e.Line }

// StructLiteral
func (e *StructLiteral) GetLine() int { return e.Line }

// IdentExpr (Don't forget this one)
func (e *IdentExpr) GetLine() int { return e.Line }

type TypeExpr struct {
	Name string
}

func (TypeExpr) isExpr() {}

// StringExpr represents a string literal.
type StringExpr struct {
	Literal string
	Value   string
	Line    int
}

func (*StringExpr) isExpr() {}

type IntExpr struct {
	Literal string
	Value   int
	Line    int
}

func (*IntExpr) isExpr()        {}
func (n *IntExpr) GetLine() int { return n.Line }

type FloatExpr struct {
	Literal string
	value   float64
	Line    int
}

func (e *FloatExpr) GetLine() int { return e.Line }

func (*FloatExpr) isExpr() {}

type BoolExpr struct {
	Literal string
	value   bool
	Line    int
}

func (e *BoolExpr) GetLine() int { return e.Line }

func (*BoolExpr) isExpr() {}

type LiteralExpr struct {
	Value string
	Line  int
}

func (*LiteralExpr) isExpr() {}

type IdentExpr struct {
	Type *symbols.Type
	Name string
	Line int
}

func (*IdentExpr) isExpr() {}

// BinaryExpr
type BinaryExpr struct {
	Op    string
	Left  Expression
	Right Expression
	Line  int
}

func (*BinaryExpr) isExpr()        {}
func (e *BinaryExpr) GetLine() int { return e.Line }

// Callee Expression
type CallExpr struct {
	Callee      Expression
	Args        []Expression
	Line        int
	UnwrapPanic bool
}

func (*UnaryExpr) isExpr() {}

func (*CallExpr) isExpr() {}

type StructLiteral struct {
	Type   *symbols.Type
	Fields []FieldInit
	Line   int
}

func (*StructLiteral) isExpr() {}

type FieldInit struct {
	Name  string
	Value Expression
	Line  int
}

func (p *Parser) parseCall(name string) *CallExpr {
	p.expectType(OPN_PAREN)

	args := []Expression{}

	for p.currentToken().Type != CLS_PAREN {
		if p.tokens[p.pos].Type == COMMA {
			p.pos++
			continue
		}
		arg := p.parseExpr()
		args = append(args, arg)
	}

	p.expectType(CLS_PAREN)

	unwrapPanic := false
	if p.currentToken().Type == EXCLAM {
		p.pos++
		unwrapPanic = true
	}

	return &CallExpr{
		Callee:      &IdentExpr{Name: name, Line: p.currentToken().Line},
		Args:        args,
		Line:        p.currentToken().Line,
		UnwrapPanic: unwrapPanic,
	}
}

func (p *Parser) parseExprOrAssign() Statement {
	// 1. Parse exactly one expression for the left-hand side
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}

	// 2. Check for assignment or definition operators
	tok := p.currentToken()
	if tok.Type == DEFINE || tok.Type == ASSIGN {
		p.pos++ // consume := or =

		// 3. Parse exactly one expression for the RHS
		value := p.parseExpr()
		if value == nil {
			p.appendErrorf("expected expression after %s", tok.Line, tok.Lexeme)
			return nil
		}

		// 4. Handle Short Declaration (:=)
		if tok.Type == DEFINE {
			// Validate that the target is a valid name (IdentExpr)
			if !p.isValidDefineTarget(expr) {
				p.appendErrorf("non-name on left side of := Line:%d", tok.Line, tok.Line)
			}

			return &Declar{
				Name:  expr, // Now a single expression
				Op:    tok.Lexeme,
				Value: value, // Now a single expression
				Line:  tok.Line,
			}
		}

		// 5. Handle Normal Assignment (=)
		return &Assign{
			Target: expr, // Now a single expression
			Op:     tok.Lexeme,
			Value:  value, // Now a single expression
			Line:   tok.Line,
		}
	}

	// 6. If no operator, it's an Expression Statement (e.g., function call)
	return &ExprStmt{
		Expr: expr,
		Line: tok.Line,
	}
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
