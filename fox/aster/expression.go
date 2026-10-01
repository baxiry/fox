package aster

import "fox/symbols"

type UnaryExpr struct {
	Op   string // like "*", "&"
	Expr Expression
	Line int
}

// UnaryExpr
func (*UnaryExpr) isExpr()        {}
func (e *UnaryExpr) GetLine() int { return e.Line }

// ZeroValueExpr represents an explicit zero/nil initialization expression during expansion
type ZeroValueExpr struct {
	Type *symbols.Type
	Line int
}

func (*ZeroValueExpr) isExpr()        {}
func (e *ZeroValueExpr) GetLine() int { return e.Line }

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

func (e *StringExpr) GetLine() int { return e.Line }
func (*StringExpr) isExpr()        {}

// int expression
type IntExpr struct {
	Literal string
	Value   int
	Line    int
}

func (*IntExpr) isExpr() {}
func (n *IntExpr) GetLine() int {
	return n.Line
}

type FloatExpr struct {
	Literal string
	value   float64
	Line    int
}

func (*FloatExpr) isExpr() {}
func (e *FloatExpr) GetLine() int {
	return e.Line
}

// BoolExpr
type BoolExpr struct {
	Literal string
	Value   bool
	Line    int
}

func (*BoolExpr) isExpr() {}
func (e *BoolExpr) GetLine() int {
	return e.Line
}

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

func (*BinaryExpr) isExpr() {}
func (e *BinaryExpr) GetLine() int {
	return e.Line
}

// Callee Expression
type CallExpr struct {
	Callee      Expression
	Args        []Expression
	Line        int
	UnwrapPanic bool
}

func (*CallExpr) isExpr() {}
func (e *CallExpr) GetLine() int {
	return e.Line
}

// Struct Literal
type StructLiteral struct {
	Type   *symbols.Type
	Fields []FieldInit
	Line   int
}

func (*StructLiteral) isExpr() {}
func (e *StructLiteral) GetLine() int {
	return e.Line
}

// Field Init
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

				Targets: []Expression{expr},
				Op:      tok.Lexeme,
				Value:   value, // Now a single expression
				Line:    tok.Line,
			}
		}

		// 5. Handle Normal Assignment (=)
		return &Assign{
			Targets: []Expression{expr},
			Op:      tok.Lexeme,
			Value:   value, // Now a single expression
			Line:    tok.Line,
		}
	}

	// 6. If no operator, it's an Expression Statement (e.g., function call)
	return &ExprStmt{
		Expr: expr,
		Line: tok.Line,
	}
}
