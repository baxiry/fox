package aster

import "fmt"

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
	Target Expression
	Op     string
	Value  Expression
}

type VarDeclar struct {
	Name  string
	Type  *Type
	Value Expression
}

type Declar struct {
	Name  Expression
	Op    string
	Value Expression
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

func (p *Parser) parseIf() Statement {
	p.expectType(IF)
	cond := p.parseExpr()
	thenBlock := p.parseBlock()

	var elseStmt Statement = nil // Using Statement interface for flexibility

	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == ELSE {
		p.pos++

		// If another IF follows ELSE, it's an 'else if'
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == IF {
			// Recursive call to build the chain
			elseStmt = p.parseIf()
		} else {
			// Normal ELSE block
			elseStmt = p.parseBlock()
		}
	}

	// Always return a pointer to satisfy the Statement interface
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

	forStmt := ForStmt{}

	// for {}
	if p.tokens[p.pos].Type == OPN_BRACE {
		forStmt.Body = p.parseBlock()
		return forStmt
	}

	// for condition {} (NO semicolons ahead)
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

		if !hasSemicolon {
			forStmt.Cond = p.parseExpr()
			forStmt.Body = p.parseBlock()
			return forStmt
		}
	}

	// INIT
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {

		if p.pos+1 < len(p.tokens) &&
			(p.tokens[p.pos+1].Type == ASSIGN || p.tokens[p.pos+1].Type == DEFINE) {

			forStmt.Init = p.parseDefOrAssign()
		} else {
			forStmt.Init = p.parseExprStatement()
		}
	}

	p.expectType(SEMICOLON)

	// CONDITION
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		forStmt.Cond = p.parseExpr() // parseExprUntil( ";")
	}

	p.expectType(SEMICOLON)

	// POST
	if p.tokens[p.pos].Type != OPN_BRACE && p.tokens[p.pos].Type != CLS_BRACE {

		if p.pos+1 < len(p.tokens) &&
			(p.tokens[p.pos+1].Type == ASSIGN || p.tokens[p.pos+1].Type == DEFINE) {

			forStmt.Post = p.parseDefOrAssign()
		} else {
			forStmt.Post = p.parseExprStatement()
		}
	}

	// BODY
	if p.tokens[p.pos].Type == OPN_BRACE {
		forStmt.Body = p.parseBlock()
		return forStmt
	}

	// fallback safety
	forStmt.Body = p.parseBlock()
	return forStmt
}

func (p *Parser) parseFor_Old() Statement {
	p.expectType(FOR)

	forStmt := ForStmt{}

	// for {}
	if p.tokens[p.pos].Type == OPN_BRACE {
		forStmt.Body = p.parseBlock()
		return forStmt
	}

	// for condition {}
	if p.tokens[p.pos].Type == IDENT {
		ps := p.pos
		composite := false
		for p.tokens[ps].Type != OPN_BRACE {
			if p.tokens[ps].Type == SEMICOLON {
				composite = true
				break
			}
			ps++
		}
		if !composite {
			forStmt.Cond = p.parseExpr()
			return forStmt
		}
	}

	//  INIT
	// check ";" "{" befor init
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		if p.pos+1 < len(p.tokens) && (p.tokens[p.pos+1].Type == ASSIGN || p.tokens[p.pos+1].Type == DEFINE) {
			forStmt.Init = p.parseDefOrAssign()
		} else {
			forStmt.Init = p.parseExprStatement()
		}
	}
	p.expectType(SEMICOLON) // use ;

	// CONDITION
	if p.tokens[p.pos].Type != SEMICOLON && p.tokens[p.pos].Type != OPN_BRACE {
		forStmt.Cond = p.parseExprUntil(";")
	}
	p.expectType(SEMICOLON) // use ;

	// POST
	if p.tokens[p.pos].Type != OPN_BRACE {
		if p.pos+1 < len(p.tokens) && (p.tokens[p.pos+1].Type == ASSIGN || p.tokens[p.pos+1].Type == DEFINE) {
			forStmt.Post = p.parseDefOrAssign()
		} else if p.tokens[p.pos].Type != CLS_BRACE {
			forStmt.Post = p.parseExprStatement()
		}
	}

	// for {}
	if p.tokens[p.pos].Type == OPN_BRACE {
		forStmt.Body = p.parseBlock()
		return forStmt
	}

	//  BODY

	forStmt.Body = p.parseBlock()
	return forStmt
}
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
	target := p.parsePostfix()

	switch target.(type) {
	case IdentExpr, *IdentExpr, FieldAccessExpr, *FieldAccessExpr:
	default:
		panic("left-hand side of assignment must be an identifier or field access")
	}

	opTok := p.tokens[p.pos]
	if opTok.Type != ASSIGN {
		panic("expected = for assignment")
	}
	p.pos++

	value := p.parseExpr()

	return Assign{
		Target: target,
		Op:     opTok.Lexeme,
		Value:  value,
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

	return Declar{
		Name:  target,
		Op:    opTok.Lexeme,
		Value: value,
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
