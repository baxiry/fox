package main

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
	Cond Expression
	Then *FrameBlock
	Else *FrameBlock
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
	Type  Type
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
func (SpawnStmt) isStat()    {}

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

func parseVarDeclar(tokens []Token, pos *int) Statement {
	// var obj Obj
	// var i int
	// var i int = 10
	// var i = 10 + 10

	// consome var
	expectType(tokens, pos, VAR)

	// consume variable name
	nameTok := expectType(tokens, pos, IDENT)

	var typ *Type = nil
	var value Expression = nil

	// case one & three, type
	if isTypeStart(tokens[*pos]) {
		t := parseType(tokens, pos)
		typ = &t
	}

	// case 3 & 4 =
	if tokens[*pos].Type == ASSIGN {
		*pos++
		value = parseExpr(tokens, pos)
	}

	return VarDeclar{
		Name:  nameTok.Lexeme,
		Type:  *typ,
		Value: value,
	}
}

// Statement Parsers
func parseSpawn(tokens []Token, pos *int) Statement {

	expectType(tokens, pos, SPAWN)
	t := tokens[*pos]
	expr := parseExpr(tokens, pos)
	call, ok := expr.(CallExpr)
	if !ok {
		return BadStmt{
			Msg: fmt.Sprintf("spawn expects function call at %d:%d", t.Line, t.Column),
		}
	}

	return SpawnStmt{Call: call}
}

func parseIf(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, IF)
	cond := parseExpr(tokens, pos)
	thenBlock := parseBlock(tokens, pos)

	var elseBlock *FrameBlock
	if *pos < len(tokens) && tokens[*pos].Type == ELSE {
		*pos++
		elseBlock = parseBlock(tokens, pos)
	}

	return IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}
}

func parseExprUntil(tokens []Token, pos *int, stop string) Expression {
	expr := parseExpr(tokens, pos)

	for *pos < len(tokens) && tokens[*pos].Lexeme != stop {
		op := tokens[*pos]
		if !op.IsOperator() {
			break
		}
		*pos++

		right := parseExpr(tokens, pos)
		expr = &BinaryExpr{
			Op:    op.Lexeme,
			Left:  expr,
			Right: right,
		}
	}
	return expr
}

func parseFor(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, FOR)

	forStmt := ForStmt{}

	// for {}
	if tokens[*pos].Type == OPN_BRACE {
		forStmt.Body = parseBlock(tokens, pos)
		return forStmt
	}

	// for condition {} (NO semicolons ahead)
	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != OPN_BRACE {

		ps := *pos
		hasSemicolon := false

		for ps < len(tokens) && tokens[ps].Type != OPN_BRACE {
			if tokens[ps].Type == SEMICOLON {
				hasSemicolon = true
				break
			}
			ps++
		}

		if !hasSemicolon {
			forStmt.Cond = parseExpr(tokens, pos)
			forStmt.Body = parseBlock(tokens, pos)
			return forStmt
		}
	}

	// INIT
	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != OPN_BRACE {

		if *pos+1 < len(tokens) &&
			(tokens[*pos+1].Type == ASSIGN || tokens[*pos+1].Type == DEFINE) {

			forStmt.Init = parseDefOrAssign(tokens, pos)
		} else {
			forStmt.Init = parseExprStatement(tokens, pos)
		}
	}

	expectType(tokens, pos, SEMICOLON)

	// CONDITION
	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != OPN_BRACE {
		forStmt.Cond = parseExprUntil(tokens, pos, ";")
	}

	expectType(tokens, pos, SEMICOLON)

	// POST
	if tokens[*pos].Type != OPN_BRACE && tokens[*pos].Type != CLS_BRACE {

		if *pos+1 < len(tokens) &&
			(tokens[*pos+1].Type == ASSIGN || tokens[*pos+1].Type == DEFINE) {

			forStmt.Post = parseDefOrAssign(tokens, pos)
		} else {
			forStmt.Post = parseExprStatement(tokens, pos)
		}
	}

	// BODY
	if tokens[*pos].Type == OPN_BRACE {
		forStmt.Body = parseBlock(tokens, pos)
		return forStmt
	}

	// fallback safety
	forStmt.Body = parseBlock(tokens, pos)
	return forStmt
}

func parseFor_Old(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, FOR)

	forStmt := ForStmt{}

	// for {}
	if tokens[*pos].Type == OPN_BRACE {
		forStmt.Body = parseBlock(tokens, pos)
		return forStmt
	}

	// for condition {}
	if tokens[*pos].Type == IDENT {
		ps := *pos
		composite := false
		for tokens[ps].Type != OPN_BRACE {
			if tokens[ps].Type == SEMICOLON {
				composite = true
				break
			}
			ps++
		}
		if !composite {
			forStmt.Cond = parseExpr(tokens, pos)
			return forStmt
		}
	}

	//  INIT
	// check ";" "{" befor init
	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != OPN_BRACE {
		if *pos+1 < len(tokens) && (tokens[*pos+1].Type == ASSIGN || tokens[*pos+1].Type == DEFINE) {
			forStmt.Init = parseDefOrAssign(tokens, pos)
		} else {
			forStmt.Init = parseExprStatement(tokens, pos)
		}
	}
	expectType(tokens, pos, SEMICOLON) // use ;

	// CONDITION
	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != OPN_BRACE {
		forStmt.Cond = parseExprUntil(tokens, pos, ";")
	}
	expectType(tokens, pos, SEMICOLON) // use ;

	// POST
	if tokens[*pos].Type != OPN_BRACE {
		if *pos+1 < len(tokens) && (tokens[*pos+1].Type == ASSIGN || tokens[*pos+1].Type == DEFINE) {
			forStmt.Post = parseDefOrAssign(tokens, pos)
		} else if tokens[*pos].Type != CLS_BRACE {
			forStmt.Post = parseExprStatement(tokens, pos)
		}
	}

	// for {}
	if tokens[*pos].Type == OPN_BRACE {
		forStmt.Body = parseBlock(tokens, pos)
		return forStmt
	}

	//  BODY

	forStmt.Body = parseBlock(tokens, pos)
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

func parseReturn(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, RETURN)

	if !isExprStart(tokens[*pos]) {
		return ReturnStmt{
			Results: nil,
		}
	}
	results := []Expression{}

	if tokens[*pos].Type != SEMICOLON && tokens[*pos].Type != CLS_BRACE {
		results = append(results, parseExpr(tokens, pos))
		for *pos < len(tokens) && tokens[*pos].Type == COMMA {
			*pos++
			results = append(results, parseExpr(tokens, pos))
		}
	}

	return ReturnStmt{Results: results}
}

func parseRetSign(tokens []Token, pos *int) []ReturnSig {

	var retSigns []ReturnSig

	for *pos < len(tokens) && tokens[*pos].Type != OPN_BRACE {

		// skip commas
		if tokens[*pos].Type == COMMA {
			(*pos)++
			continue
		}

		if *pos >= len(tokens) || tokens[*pos].Type == OPN_BRACE {
			break
		}

		var name string

		// case: named return value  (name type)
		if *pos+1 < len(tokens) &&
			tokens[*pos].Type == IDENT &&
			tokens[*pos+1].Type == IDENT {

			name = expectIdent(tokens, pos).Lexeme
		}

		// parse type (this handles Obj, *Obj, etc)
		typ := parseType(tokens, pos)

		retSigns = append(retSigns, ReturnSig{
			Name: name,
			Type: typ,
		})
	}

	return retSigns
}
func parseExprStatement(tokens []Token, pos *int) Statement {

	skip(tokens, pos)

	expr := parseExpr(tokens, pos)
	if *pos < len(tokens) && tokens[*pos].Type == SEMICOLON {
		*pos++
	}

	return ExprStmt{Expr: expr}
}

func parseAssign(tokens []Token, pos *int) Statement {
	target := parsePrimary(tokens, pos)

	switch target.(type) {
	case IdentExpr, *IdentExpr, FieldAccessExpr, *FieldAccessExpr:
	default:
		panic("left-hand side of assignment must be an identifier or field access")
	}

	opTok := tokens[*pos]
	if opTok.Type != ASSIGN {
		panic("expected = for assignment")
	}
	*pos++

	value := parseExpr(tokens, pos)

	return Assign{
		Target: target,
		Op:     opTok.Lexeme,
		Value:  value,
	}
}

func parseDefine(tokens []Token, pos *int) Statement {
	target := parsePostfix(tokens, pos)

	switch target.(type) {
	case IdentExpr, *IdentExpr, FieldAccessExpr, *FieldAccessExpr:
	default:
		panic("left-hand side of define must be an identifier or field access")
	}

	opTok := tokens[*pos]
	if opTok.Type != DEFINE {
		panic("expected := for definition")
	}
	*pos++

	value := parseExpr(tokens, pos)

	return Declar{
		Name:  target,
		Op:    opTok.Lexeme,
		Value: value,
	}
}

// For init/post
func parseDefOrAssign(tokens []Token, pos *int) Statement {
	if *pos+1 < len(tokens) && tokens[*pos+1].Type == DEFINE {
		return parseDefine(tokens, pos)
	}
	return parseAssign(tokens, pos)
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

func parseStatement(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Type {
	case VAR:
		return parseVarDeclar(tokens, pos)
	case RETURN:
		return parseReturn(tokens, pos)
	case IF:
		return parseIf(tokens, pos)
	case FOR:
		return parseFor(tokens, pos)
	case BREAK:
		*pos++
		return BreakNode{Tok: tok}

	case CONTINUE:
		*pos++
		return ContinueNode{Tok: tok}
	case SPAWN:
		return parseSpawn(tokens, pos)

	default:
		return parseExprOrAssign(tokens, pos)
	}
}

// end
