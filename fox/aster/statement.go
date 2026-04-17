package aster

import (
	"fmt"
)

type Statement interface {
	isStat()
}

type BreakNode struct {
	Tok Token
}

type ContinueNode struct {
	Tok Token
}

type ReturnStmt struct {
	Results []Expression
}

type IfStmt struct {
	Cond    Expression
	Then    *FrameBlock
	Else    Statement
	HasElse bool
}

type ForStmt struct {
	Init Statement
	Cond Expression
	Post Statement
	Body *FrameBlock
}

type Assign struct {
	Targets []Expression
	Op      string
	Values  []Expression
	Line    int
}

type VarDeclar struct {
	Name  string
	Type  *Type
	Value Expression
	Line  int
}

type Declar struct {
	Names  []Expression
	Op     string
	Values []Expression
	Line   int
}

type ExprStmt struct {
	Expr Expression
}

type FieldValue struct {
	Field IdentExpr  // something like 'obj'
	Value Expression // NumberExpr{10}
}

type SpawnStmt struct {
	Call Expression
}
type BadStmt struct{ Msg string }

func (BadStmt) isStat()      {}
func (VarDeclar) isStat()    {}
func (ReturnStmt) isStat()   {}
func (ContinueNode) isStat() {}
func (IfStmt) isStat()       {}
func (BreakNode) isStat()    {}
func (ForStmt) isStat()      {}
func (Assign) isStat()       {}
func (Declar) isStat()       {}
func (ExprStmt) isStat()     {}
func (*SpawnStmt) isStat()   {}

// Parsing Helpers
func isTypeStart(tok Token) bool {
	switch tok.Type {
	case IDENT: // Obj, X, User ...
		return true
	case STAR: // *Obj
		return true
	default:
		return false
	}
}

func (p *Parser) parseVarDeclar() Statement {
	// var obj Obj
	// var i int
	// var i int = 10
	// var i = 10 + 10

	// consome var
	p.expectType(VAR)

	// consume variable name
	nameTok := p.expectType(IDENT)

	var typ *Type = nil
	var value Expression = nil

	// case one & three, type
	if isTypeStart(p.tokens[p.pos]) {
		tmp := p.parseType()
		typ = &tmp
	}

	// case 3 & 4 =
	if p.tokens[p.pos].Type == ASSIGN {
		p.pos++
		value = p.parseExpr()
	}

	return &VarDeclar{
		Name:  nameTok.Lexeme,
		Type:  typ,
		Value: value,
	}
}

// Statement Parsers
func (p *Parser) parseSpawn() Statement {

	p.expectType(SPAWN)
	t := p.tokens[p.pos]
	expr := p.parseExpr()
	call, ok := expr.(CallExpr)
	if !ok {
		return BadStmt{
			Msg: fmt.Sprintf("spawn expects function call at %d:%d", t.Line, t.Column),
		}
	}

	return &SpawnStmt{Call: call}
}

func (p *Parser) current() string {
	return p.tokens[p.pos].Lexeme
}

func (p *Parser) parseIf() Statement {

	p.expectType(IF)
	p.inCondition = true
	cond := p.parseExpr()
	p.inCondition = false

	// Skip newlines/whitespace after the condition to find the '{'
	p.skip()

	thenBlock := p.parseBlock()

	var elseStmt Statement = nil

	// Skip potential newlines before checking for ELSE
	p.skip()

	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == ELSE {
		p.pos++

		// Skip newlines after ELSE to see if an IF or a '{' follows
		p.skip()

		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == IF {
			elseStmt = p.parseIf()
		} else {
			elseStmt = p.parseBlock()
		}
	}

	return &IfStmt{
		Cond: cond,
		Then: thenBlock,
		Else: elseStmt,
	}
}

func (p *Parser) parseExprUntil(stop string) Expression {
	expr := p.parseExpr()

	for p.pos < len(p.tokens) && p.tokens[p.pos].Lexeme != stop {
		op := p.tokens[p.pos]
		if !op.IsOperator() {
			break
		}
		p.pos++

		right := p.parseExpr()
		expr = &BinaryExpr{
			Op:    op.Lexeme,
			Left:  expr,
			Right: right,
		}
	}
	return expr
}

func (p *Parser) parseFor() Statement {
	p.expectType(FOR)

	// Enable condition mode to prevent parsePostfix from misinterpreting
	// the loop's '{' as a struct literal.
	p.inCondition = true

	forStmt := ForStmt{}

	// 1. Infinite loop: for {}
	if p.tokens[p.pos].Type == OPN_BRACE {
		p.inCondition = false // Must disable before calling parseBlock
		forStmt.Body = p.parseBlock()
		return forStmt
	}

	// Helper function to scan ahead for assignment operators (= or :=)
	isAssignOrDef := func() bool {
		ps := p.pos
		for ps < len(p.tokens) && p.tokens[ps].Type != SEMICOLON && p.tokens[ps].Type != OPN_BRACE {
			if p.tokens[ps].Type == ASSIGN || p.tokens[ps].Type == DEFINE {
				return true
			}
			ps++
		}
		return false
	}

	// 2. Case: for condition {} (While-style)
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		ps := p.pos
		hasSemicolon := false
		for ps < len(p.tokens) && p.tokens[ps].Type != OPN_BRACE {
			if p.tokens[ps].Type == SEMICOLON {
				hasSemicolon = true
				break
			}
			ps++
		}

		if !hasSemicolon && !isAssignOrDef() {
			forStmt.Cond = p.parseExpr()
			p.inCondition = false // Header finished
			forStmt.Body = p.parseBlock()
			return forStmt
		}
	}

	// 3. Case: for init; cond; post {} (C-style)

	// --- INIT ---
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		if isAssignOrDef() {
			forStmt.Init = p.parseDefOrAssign()
		} else {
			forStmt.Init = p.parseExprStatement()
		}
	}

	p.expectType(SEMICOLON)

	// --- CONDITION ---
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		forStmt.Cond = p.parseExpr()
	}

	p.expectType(SEMICOLON)

	// --- POST ---
	if p.tokens[p.pos].Type != OPN_BRACE {
		if isAssignOrDef() {
			forStmt.Post = p.parseDefOrAssign()
		} else {
			forStmt.Post = p.parseExprStatement()
		}
	}

	// Disable condition mode before parsing the loop body
	p.inCondition = false

	// --- BODY ---
	forStmt.Body = p.parseBlock()
	return forStmt
}

// .
func isExprStart(tok Token) bool {
	switch tok.Type {

	case IDENT,
		INT, FLOAT, STRING,
		TRUE, FALSE,

		OPN_PAREN, // (a + b)
		AMP,       // &a
		STAR,      // *a
		NOT,       // !a
		MINUS:     // -a

		return true
	}

	return false
}

func (p *Parser) parseReturn() Statement {
	p.expectType(RETURN)

	if !isExprStart(p.tokens[p.pos]) {
		return ReturnStmt{
			Results: nil,
		}
	}
	results := []Expression{}

	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != CLS_BRACE {
		results = append(results, p.parseExpr())
		for p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
			p.pos++
			results = append(results, p.parseExpr())
		}
	}

	return ReturnStmt{Results: results}
}

func (p *Parser) parseRetSign() []ReturnSig {

	var retSigns []ReturnSig

	for p.pos < len(p.tokens) && p.tokens[p.pos].Type != OPN_BRACE {

		// skip commas
		if p.tokens[p.pos].Type == COMMA {
			(p.pos)++
			continue
		}

		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type == OPN_BRACE {
			break
		}

		var name string

		// case: named return value  (name type)
		if p.pos+1 < len(p.tokens) &&
			p.tokens[p.pos].Type == IDENT &&
			p.tokens[p.pos+1].Type == IDENT {

			name = p.expectIdent().Lexeme
		}

		// parse type (this handles Obj, *Obj, etc)
		typ := p.parseType()

		retSigns = append(retSigns, ReturnSig{
			Name: name,
			Type: typ,
		})
	}

	return retSigns
}
func (p *Parser) parseExprStatement() Statement {

	p.skip()

	expr := p.parseExpr()
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == SEMICOLON {
		p.pos++
	}

	return ExprStmt{Expr: expr}
}

func (p *Parser) parseAssign() Statement {
	// 1. Parse the first target from the current position
	// This will read "i" or "data.x"
	firstTarget := p.parsePostfix()

	// 2. Collect additional targets if there's a comma (for u, o = ...)
	targets := []Expression{firstTarget}
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
		p.pos++ // consume ','
		targets = append(targets, p.parsePostfix())
	}

	// 3. Look for the assignment operator '='
	if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != ASSIGN {
		// Here we handle the panic with more context
		line := 0
		if p.pos < len(p.tokens) {
			line = p.tokens[p.pos].Line
		}
		panic(fmt.Sprintf("line %d: expected '=' for assignment, but found %s",
			line, p.tokens[p.pos].Lexeme))
	}

	opTok := p.tokens[p.pos]
	p.pos++ // consume '='

	// 4. Parse the right-hand side values
	values := []Expression{p.parseExpr()}
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
		p.pos++ // consume ','
		values = append(values, p.parseExpr())
	}

	return Assign{
		Targets: targets,
		Op:      opTok.Lexeme,
		Values:  values,
		Line:    opTok.Line,
	}
}

func (p *Parser) parseDefine() Statement {
	target := p.parsePostfix()

	switch target.(type) {
	case IdentExpr, *IdentExpr, FieldAccessExpr, *FieldAccessExpr:
	default:
		panic("left-hand side of define must be an identifier or field access")
	}

	opTok := p.tokens[p.pos]
	if opTok.Type != DEFINE {
		panic("expected := for definition")
	}
	p.pos++

	value := p.parseExpr()

	line := p.tokens[p.pos].Line
	return Declar{
		Names:  []Expression{target},
		Op:     opTok.Lexeme,
		Values: []Expression{value},
		Line:   line,
	}
}

// For init/post
func (p *Parser) parseDefOrAssign() Statement {
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == DEFINE {
		return p.parseDefine()
	}
	return p.parseAssign()
}

func (t Token) IsOperator() bool {
	switch t.Type {
	case PLUS, MINUS, STAR, SLASH, ASSIGN, DEFINE,
		EQ, NEQ, LT, GT, LTE, GTE, AND, OR, NOT, DOT:
		return true
	default:
		return false
	}
}

func (p *Parser) parseStatement() Statement {
	tok := p.tokens[p.pos]

	switch tok.Type {
	case VAR:
		return p.parseVarDeclar()
	case RETURN:
		return p.parseReturn()
	case IF:
		return p.parseIf()
	case FOR:
		return p.parseFor()
	case BREAK:
		p.pos++
		return BreakNode{Tok: tok}

	case CONTINUE:
		p.pos++
		return ContinueNode{Tok: tok}
	case SPAWN:
		return p.parseSpawn()

	default:
		return p.parseExprOrAssign()
	}
}

// end
