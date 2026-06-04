package aster

import (
	"fmt"
	"fox/symbols"
)

type BreakNode struct {
	//Tok Token
	Line int
}

type ContinueNode struct {
	//Tok Token
	Line int
}

func (ContinueNode) isStmt() {}

type ReturnStmt struct {
	Result Expression
	Line   int
}

type IfStmt struct {
	Cond    Expression
	Then    *FrameBlock
	Else    Statement
	HasElse bool
	Line    int
}

type ForStmt struct {
	Init Statement
	Cond Expression
	Post Statement
	Body *FrameBlock
	Line int
}

type Assign struct {
	Target Expression
	Op     string
	Value  Expression
	Line   int
}

type Declar struct {
	Name  Expression
	Op    string
	Value Expression
	Line  int
}

type ExprStmt struct {
	Expr Expression
	Line int
}

type FieldValue struct {
	Field IdentExpr  // something like 'obj'
	Value Expression // NumberExpr{10}
}

type SpawnStmt struct {
	Call Expression
	Line int
}

// BadStmt
type BadStmt struct {
	Msg  string
	Line int
}

func (s *BadStmt) GetLine() int { return s.Line }
func (s *BadStmt) isStmt()      {}
func (s *BadStmt) isExpr()      {}

// BreakNode
func (s *BreakNode) GetLine() int { return s.Line }
func (s *BreakNode) isStmt()      {}

// VarDeclar
func (s *VarDeclar) isStmt() {}

// expression too
func (*Declar) isExpr() {}

// SpawnStmt
func (s *SpawnStmt) GetLine() int { return s.Line }
func (s *SpawnStmt) isStmt()      {}

// FrameBlock (or Block)
func (s *FrameBlock) GetLine() int { return s.Line }
func (s *FrameBlock) isStmt()      {}

// IfStmt
func (s *IfStmt) GetLine() int { return s.Line }
func (s *IfStmt) isStmt()      {}

// ForStmt
func (s *ForStmt) GetLine() int { return s.Line }
func (s *ForStmt) isStmt()      {}

// ReturnStmt
func (s *ReturnStmt) GetLine() int { return s.Line }
func (s *ReturnStmt) isStmt()      {}
func (BadStmt) isStat()            {}
func (VarDeclar) isStat()          {}
func (ReturnStmt) isStat()         {}
func (ContinueNode) isStat()       {}
func (IfStmt) isStat()             {}
func (BreakNode) isStat()          {}
func (ForStmt) isStat()            {}
func (Assign) isStat()             {}
func (Declar) isStat()             {}
func (ExprStmt) isStat()           {}
func (*SpawnStmt) isStat()         {}

// ContinueNode
func (s *ContinueNode) GetLine() int { return s.Line }

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

// var obj Obj
// var i int
// var i int = 10
// var i = 10 + 10

func (p *Parser) parseVarDeclar() Statement {
	p.expectType(VAR)
	nameTok := p.expectIdent()

	var typ *symbols.Type = nil
	var value Expression = nil

	curr := p.currentToken()
	if curr.Type == IDENT || curr.Type == OPN_BRACK || curr.Type == STAR {
		t := p.parseType()
		typ = &t
	}

	if p.currentToken().Type == ASSIGN {
		p.pos++ // consume '='
		p.skip()

		if p.currentToken().Type == IDENT && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == DOT {
			parentNameTok := p.expectIdent()

			value = p.parseEnumLiteral(parentNameTok.Lexeme)
		} else {
			value = p.parseExpr()
		}
	}

	p.skipNewlines()

	return &VarDeclar{
		Name:  nameTok.Lexeme,
		Type:  typ,
		Value: value,
		Line:  nameTok.Line,
	}
}

// Statement Parsers
func (p *Parser) parseSpawn() Statement {

	p.expectType(SPAWN)
	t := p.tokens[p.pos]
	expr := p.parseExpr()
	call, ok := expr.(*CallExpr)
	if !ok {
		return &BadStmt{
			Msg: fmt.Sprintf("spawn expects function call at %d:%d", t.Line, t.Column),
		}
	}

	return &SpawnStmt{Call: call}
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
			Line:  expr.GetLine(),
		}
	}
	return expr
}

func (p *Parser) parseFor() Statement {
	p.expectType(FOR)

	// Enable condition mode to prevent misinterpreting '{' as a struct literal
	p.inCondition = true

	forStmt := ForStmt{}

	// 1. Case: Infinite loop -> for {}
	if p.tokens[p.pos].Type == OPN_BRACE {
		p.inCondition = false
		forStmt.Body = p.parseBlock()
		return &forStmt
	}

	// Helper to detect if we are in a 3-part loop (C-style)
	// by scanning for a SEMICOLON before the opening brace
	hasSemicolon := false
	ps := p.pos
	for ps < len(p.tokens) && p.tokens[ps].Type != OPN_BRACE {
		if p.tokens[ps].Type == SEMICOLON {
			hasSemicolon = true
			break
		}
		ps++
	}

	// 2. Case: for condition {} (While-style)
	if !hasSemicolon {
		forStmt.Cond = p.parseExpr()
		p.inCondition = false
		forStmt.Body = p.parseBlock()
		return &forStmt
	}

	// 3. Case: for init; cond; post {} (C-style)
	// Helper to distinguish between an assignment (i := 0) and an expression (a == 0)
	isPostAssign := func() bool {
		curr := p.pos
		for curr < len(p.tokens) && p.tokens[curr].Type != SEMICOLON {
			if p.tokens[curr].Type == ASSIGN || p.tokens[curr].Type == DEFINE {
				return true
			}
			curr++
		}
		return false
	}

	//INIT
	if p.tokens[p.pos].Type != SEMICOLON {
		// We use a modified check to see if we should call parseDefOrAssign
		if isPostAssign() {
			forStmt.Init = p.parseDefOrAssign()
		} else {
			// This is what will catch 'a == 0'
			// We use parseExprStatement but we DON'T consume the semicolon here
			expr := p.parseExpr()
			forStmt.Init = &ExprStmt{Expr: expr}
		}
	}
	p.expectType(SEMICOLON)

	// CONDITION
	if p.tokens[p.pos].Type != SEMICOLON {
		forStmt.Cond = p.parseExpr()
	}
	p.expectType(SEMICOLON)

	//POST
	if p.tokens[p.pos].Type != OPN_BRACE {
		/* 🔍 Look ahead directly to check if the upcoming pattern represents a shorthand increment step */
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == PLUS_PLUS {
			targetIdent := p.parsePostfix() /* Safely consume the variable identifier (e.g., 'i') */
			opTok := p.tokens[p.pos]        /* Capture the actual "++" token for accurate line logging */
			p.pos++                         /* Explicitly consume the "++" token to balance the parsing layout */

			/* Inject the structural sugar directly, transforming it into a standard i = i + 1 node */
			forStmt.Post = &Assign{
				Target: targetIdent,
				Op:     "=",
				Value: &BinaryExpr{
					Op:    "+",
					Left:  targetIdent,
					Right: &IntExpr{Literal: "1", Value: 1, Line: opTok.Line},
					Line:  opTok.Line,
				},
				Line: opTok.Line,
			}
		} else {
			/* Fallback to the original routing execution branches if standard expressions emerge */
			isPostAssign := func() bool {
				curr := p.pos
				for curr < len(p.tokens) && p.tokens[curr].Type != OPN_BRACE {
					if p.tokens[curr].Type == ASSIGN || p.tokens[curr].Type == DEFINE {
						return true
					}
					curr++
				}
				return false
			}

			if isPostAssign() {
				forStmt.Post = p.parseDefOrAssign()
			} else {
				forStmt.Post = p.parseExprStatement()
			}
		}
	}

	// Header is finished, disable condition mode
	p.inCondition = false

	// --- BODY PART ---
	forStmt.Body = p.parseBlock()
	return &forStmt
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
		EXCLAM,    // !a
		MINUS:     // -a

		return true
	}

	return false
}

func (p *Parser) parseReturn() Statement {
	line := p.currentToken().Line
	p.expectType(RETURN)

	// 1. If it's an empty return (void function)
	if p.currentToken().Type == SEMICOLON ||
		p.currentToken().Type == NEW_LINE ||
		p.currentToken().Type == CLS_BRACE {
		return &ReturnStmt{
			Result: nil,
			Line:   line,
		}
	}

	// 2. Parse exactly one expression
	result := p.parseExpr()
	if result == nil {
		p.appendErrorf("expected expression after return", line)
		return nil
	}

	// 3. Strict Check: If a comma follows, it's an error in Fox
	if p.currentToken().Type == COMMA {
		p.appendErrorf("multi-value return is not supported in Fox", p.currentToken().Line)
		// Error recovery: skip until we find a statement end
		p.synchronize()
	}

	return &ReturnStmt{
		Result: result,
		Line:   line,
	}
}

func (p *Parser) parseRetSign() *ReturnSig {
	if p.currentToken().Type == OPN_BRACE {
		return nil
	}

	line := p.currentToken().Line
	typ := p.parseType()

	isErrorUnion := false
	if p.currentToken().Type == EXCLAM {
		p.pos++
		isErrorUnion = true
	}

	if p.currentToken().Type == COMMA {
		p.appendErrorf("multi-value return types are not supported", p.currentToken().Line)
		for p.pos < len(p.tokens) && p.currentToken().Type != OPN_BRACE {
			p.pos++
		}
	}

	return &ReturnSig{
		Type:         &typ,
		IsErrorUnion: isErrorUnion,
		Line:         line,
	}
}

// Helper function to check if the return is named (e.g., 'res int')
func (p *Parser) isNamedReturn() bool {
	// A named return must have an identifier followed by a type start (IDENT, STAR, or OPN_BRACK)
	if p.currentToken().Type == IDENT {
		next := p.peekToken()
		return next.Type == IDENT || next.Type == STAR || next.Type == OPN_BRACK
	}
	return false
}

func (p *Parser) parseExprStatement() Statement {

	p.skip()

	expr := p.parseExpr()
	//if p.pos < len(p.tokens) && p.tokens[p.pos].Type == SEMICOLON {
	//	p.pos++
	//}

	return &ExprStmt{Expr: expr}
}

// ParseDefOrAssign processes variable initialization or inline target value mutations.
// It acts as a static syntactic transformer for shorthand arithmetic increment steps.
func (p *Parser) parseDefOrAssign() Statement {
	// 1. Parse exactly one target on the left-hand side
	target := p.parsePostfix()

	// Strict Check: Ensure no multiple targets are attempted
	if p.currentToken().Type == COMMA {
		p.appendErrorf("multiple assignment is not supported in Fox", p.currentToken().Line)
		p.synchronize()
		return nil
	}

	// 🔍 2. Check if this is a shorthand postfix increment operator (e.g., i++)
	if p.currentToken().Type == PLUS_PLUS {
		opTok := p.currentToken()
		p.pos++ // Consume the "++" token cleanly

		// Transform the shortcut operation internally into a standard semantic representation (i = i + 1)
		return &Assign{
			Target: target,
			Op:     "=",
			Value: &BinaryExpr{
				Op:    "+",
				Left:  target,
				Right: &IntExpr{Literal: "1", Value: 1, Line: opTok.Line},
				Line:  opTok.Line,
			},
			Line: opTok.Line,
		}
	}

	// 3. Standard Path: Identify the operator (must be '=' or ':=')
	opTok := p.currentToken()
	if opTok.Type != ASSIGN && opTok.Type != DEFINE {
		p.appendErrorf("expected '=' or ':=' after expression, but found %q", opTok.Line, opTok.Lexeme)
		p.synchronize()
		return nil
	}
	p.pos++ // consume the operator

	// 4. Parse exactly one expression on the right-hand side
	value := p.parseExpr()
	if value == nil {
		p.appendErrorf("expected expression on the right side of %s", opTok.Line, opTok.Lexeme)
		return nil
	}

	// Double Check: Ensure no multiple values follow
	if p.currentToken().Type == COMMA {
		p.appendErrorf("multiple values in assignment are not supported in Fox", p.currentToken().Line)
		p.synchronize()
	}

	// 5. Return the appropriate node based on the operator type
	if opTok.Type == DEFINE {
		if !p.isValidDefineTarget(target) {
			p.appendErrorf("non-name on left side of :=", opTok.Line)
		}
		return &Declar{
			Name:  target,
			Op:    opTok.Lexeme,
			Value: value,
			Line:  opTok.Line,
		}
	}

	// Default to a standard Assignment node (=)
	return &Assign{
		Target: target,
		Op:     opTok.Lexeme,
		Value:  value,
		Line:   opTok.Line,
	}
}

// isValidDefineTarget checks if the expression is a valid identifier for ':='
func (p *Parser) isValidDefineTarget(expr Expression) bool {
	switch expr.(type) {
	case *IdentExpr:
		// Only plain identifiers (like 'x' or '_') are allowed for definition
		return true
	default:
		// Complex expressions like FieldAccess (x.y) are not allowed for ':='
		return false
	}
}

func (t Token) IsOperator() bool {
	switch t.Type {
	case PLUS, MINUS, STAR, SLASH, ASSIGN, DEFINE,
		EQ, NEQ, LT, GT, LTE, GTE, AND, OR, EXCLAM, DOT:
		return true
	default:
		return false
	}
}

func (p *Parser) parseStatement() Statement {
	for p.currentToken().Type == NEW_LINE {
		p.pos++
	}

	tok := p.currentToken()
	var stmt Statement

	switch tok.Type {
	case VAR:
		stmt = p.parseVarDeclar()

	case RETURN:
		stmt = p.parseReturn()

	case IF:
		return p.parseIf()

	case FOR:
		return p.parseFor()

	case SWITCH:
		return p.parseSwitch()

	case MATCH:
		return p.parseMatch()

	case BREAK:
		p.pos++
		stmt = &BreakNode{Line: tok.Line}

	case CONTINUE:
		p.pos++
		stmt = &ContinueNode{Line: tok.Line}

	case SPAWN:
		stmt = p.parseSpawn()

	default:
		stmt = p.parseExprOrAssign()
	}

	if p.currentToken().Type == SEMICOLON {
		p.pos++
	}
	return stmt
}

// end
