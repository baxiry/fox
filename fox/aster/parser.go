package aster

import (
	"fmt"
	"strconv"
)

type Parser struct {
	tokens      []Token
	pos         int
	Errors      []string
	inCondition bool
	debug       bool
}

// Inside your parser where you handle StringExpr
func (p *Parser) parseStringExpr() *StringExpr {
	tok := p.tokens[p.pos]
	raw := tok.Lexeme

	// Process: remove leading/trailing quotes
	processed := ""
	if len(raw) >= 2 {
		processed = raw[1 : len(raw)-1]
	}

	return &StringExpr{
		Literal: raw,       // The raw "hello"
		Value:   processed, // The actual hello
		Line:    tok.Line,
	}
}

// New Parser
func NewParser() *Parser {
	return &Parser{
		tokens: make([]Token, 0),
		Errors: []string{},
		pos:    0,
	}
}

func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == NEW_LINE {
		p.pos++
	}
}

// skipe some tockens after invalid tocken
func (p *Parser) synchronize() {
	// Skip the token that caused the error
	p.pos++

	for p.pos < len(p.tokens) {
		// If we hit a semicolon (end of statement), we are likely safe to resume
		if p.tokens[p.pos-1].Type == SEMICOLON || p.tokens[p.pos-1].Type == NEW_LINE {
			return
		}

		// Check if the current token is a major keyword that starts a new block/statement
		switch p.tokens[p.pos].Type {
		case FUNC, TYPE, IF, FOR, RETURN, VAR:
			return
		}

		// Keep skipping if we haven't found a safe spot
		p.pos++
	}
}

func (p *Parser) parseUnary() Expression {

	if p.pos < len(p.tokens) {

		switch p.tokens[p.pos].Type {

		case NOT, MINUS, STAR, AMP:
			op := p.tokens[p.pos]
			p.pos++
			return &UnaryExpr{Op: op.Lexeme, Expr: p.parseUnary(), Line: op.Line}
		}
	}

	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Expression {
	p.skip()

	// IMPORTANT: start from primary ONLY (no unary call here)
	expr := p.parsePrimary()

	for p.pos < len(p.tokens) {
		switch p.tokens[p.pos].Type {

		case OPN_BRACK: // [
			p.pos++ // consume '['
			index := p.parseExpr()
			p.expectType(CLS_BRACK) // expect and consume ']'

			expr = &IndexExpr{
				Target: expr,
				Index:  index,
				Line:   p.tokens[p.pos-1].Line,
			}

		case DOT:
			p.pos++
			p.skip()
			field := p.expectIdent()

			if field.Type == ERROR {
				p.synchronize() // Jump to the next safe statement

				return &BadStmt{Line: p.currentToken().Line}
			}
			expr = &FieldAccessExpr{Object: expr, Field: field.Lexeme, Line: field.Line}

		case OPN_BRACE:

			// Follow Go's rule: Struct literals must start the brace on the same line.
			// If the brace is on a new line, it belongs to a new block (if, for, etc.)
			if p.inCondition {
				return expr
			}

			if p.tokens[p.pos].Line > p.tokens[p.pos-1].Line {
				return expr
			}

			// Struct literals are only valid if the preceding expression is an identifier (the type name)
			if ident, ok := expr.(*IdentExpr); ok {
				peekPos := p.pos + 1
				isStructLiteral := true
				if peekPos < len(p.tokens) {
					peekTok := p.tokens[peekPos]
					// If we see a keyword inside the brace immediately, it's likely a block, not a struct
					if peekTok.Type == IF || peekTok.Type == FOR || peekTok.Type == RETURN ||
						peekTok.Type == BREAK || peekTok.Type == CONTINUE {
						isStructLiteral = false
					}
					// Empty braces 'A{}' are valid struct literals
					if peekTok.Type == CLS_BRACE {
						isStructLiteral = true
					}
				}
				if isStructLiteral {
					expr = p.parseStructLiteral(ident.Name)
				} else {
					// It's a block, let the calling function (like parseIf) handle it
					return expr
				}
			} else {
				// Expression is not an identifier, so '{' must be a block
				return expr
			}
		default:
			return expr
		}
	}

	return expr
}

func (p *Parser) parseBinary(minPrec int) Expression {

	left := p.parseUnary()

	for p.pos < len(p.tokens) {

		tok := p.currentToken()

		prec, ok := precedence(tok.Type)
		if !ok || prec < minPrec {
			break
		}

		p.pos++ // consume operator

		right := p.parseBinary(prec + 1)

		left = &BinaryExpr{
			Left:  left,
			Op:    tok.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}

	return left
}

func precedence(t TokenType) (int, bool) {
	switch t {
	case OR:
		return 1, true
	case AND:
		return 2, true
	case EQ, NEQ:
		return 3, true
	case LT, GT, LTE, GTE:
		return 4, true
	case PLUS, MINUS:
		return 4, true
	case STAR, SLASH:
		return 5, true
	// AMP not correct here!.

	default:
		return 0, false
	}
}

func isUnaryStart(tok Token) bool {
	return tok.Type == AMP ||
		tok.Type == STAR ||
		tok.Type == NOT ||
		tok.Type == MINUS
}

func (p *Parser) parseComparison() Expression {
	// Start with higher precedence (addition/subtraction)
	expr := p.parseAdd()

	// Check for comparison operators: >, <, >=, <=
	for p.currentToken().Type == GT || p.currentToken().Type == LT ||
		p.currentToken().Type == GTE || p.currentToken().Type == LTE {

		tok := p.currentToken()
		p.pos++ // Advance to next token

		right := p.parseAdd()

		expr = &BinaryExpr{
			Left:  expr,
			Op:    tok.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}

	return expr
}
func (p *Parser) parsePrimary() Expression {

	tok := p.currentToken()

	switch tok.Type {
	case BOOL:
		val := p.currentToken().Lexeme
		line := p.currentToken().Line
		p.pos++
		return &BoolExpr{Literal: val, Line: line}

	case IDENT:
		p.pos++
		if p.currentToken().Type == OPN_PAREN {
			return p.parseCall(tok.Lexeme)
		}
		return &IdentExpr{Name: tok.Lexeme, Line: tok.Line}

	case BLANK:
		p.pos++
		return &IdentExpr{Name: "_"}

	case INT, FLOAT:
		p.pos++
		val, _ := strconv.Atoi(tok.Lexeme)

		return &IntExpr{Value: val, Line: tok.Line}

	case STRING:
		p.pos++
		return &StringExpr{Literal: tok.Lexeme, Line: tok.Line}

	case OPN_PAREN:
		p.pos++
		expr := p.parseExpr()

		p.expectType(CLS_PAREN)
		return expr

	case EOF:
		p.appendErrorf("unexpected end of input while parsing expression", tok.Line)
		return &BadStmt{Line: p.currentToken().Line}

	default:
		//
		if tok.Type == STAR {
			return p.parseUnary()
		}
		p.appendErrorf("expected expression, but found %s (%q)",
			tok.Line, tok.Type, tok.Lexeme)
		p.synchronize()
		return &BadStmt{Line: p.currentToken().Line}
	}
}

func (p *Parser) parseType() Type {

	if p.pos >= len(p.tokens) {
		p.appendErrorf("unexpected end of input while parsing type", p.currentToken().Line)
	}

	isArr := false
	size := 0
	if p.currentToken().Type == OPN_BRACK {
		p.pos++ // consume '['
		if IsNumericLiteral(p.currentToken()) {
			size, _ = strconv.Atoi(p.currentToken().Lexeme)
			p.pos++
		}
		p.expectType(CLS_BRACK) // consume ']'
		isArr = true
	}

	if p.tokens[p.pos].Type == AMP { // &
		p.appendErrorf(
			"syntax error at line %d, column %d: cannot use & in parameter signature",
			p.tokens[p.pos].Line, p.tokens[p.pos].Column)
	}

	ptrDepth := 0
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == STAR {
		ptrDepth++
		p.pos++
	}

	if p.pos >= len(p.tokens) {
		p.appendErrorf("unexpected end of input while parsing type after pointers", p.currentToken().Line)
	}

	name := p.expectIdent()

	if name.Type == ERROR {

		p.synchronize() // Jump to the next safe statement
		return Type{Line: name.Line, IsArray: isArr}
	}

	return Type{
		Name:     name.Lexeme,
		PtrDepth: ptrDepth,
		Line:     name.Line,
		IsArray:  isArr,
		Size:     size,
	}
}

// parse * and /
func (p *Parser) parseMul() Expression {
	left := p.parseUnary()

	tok := p.currentToken()
	for tok.Type == STAR || tok.Type == SLASH {
		op := tok
		p.pos++
		right := p.parseUnary()
		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}
	return left
}

// Bitwise AND &
func (p *Parser) parseBitwiseAnd() Expression {
	left := p.parseAdd()

	tok := p.currentToken()
	for tok.Lexeme == "&" {
		op := tok
		p.pos++
		right := p.parseAdd()

		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}

	return left
}

// Equality == & !=

func (p *Parser) parseEquality() Expression {
	// Call the new comparison level instead of parseAdd
	expr := p.parseComparison()

	for p.currentToken().Type == EQ || p.currentToken().Type == NEQ {
		tok := p.currentToken()
		p.pos++

		right := p.parseComparison()

		expr = &BinaryExpr{
			Left:  expr,
			Op:    tok.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}

	return expr
}

/// TODO   parseTerm

// parse + and -
func (p *Parser) parseAdd() Expression {
	left := p.parseMul()
	tok := p.currentToken()
	for tok.Lexeme == "+" || tok.Lexeme == "-" {
		p.pos++
		right := p.parseMul()
		left = &BinaryExpr{
			Left:  left,
			Op:    tok.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}
	return left
}

// top-level expression
func (p *Parser) parseExpr() Expression {

	return p.parseBinary(0)
}

// Functions
func (p *Parser) parseFunc() *Func {
	funcNode := Func{}

	// parse func word then name func
	p.expectType(FUNC)
	funcNode.FuncName = p.expectIdent().Lexeme
	if funcNode.FuncName == "ERROR" {

	}

	// parse open paren ( then params
	p.expectType(OPN_PAREN)

	for p.tokens[p.pos].Type != CLS_PAREN {
		// skip comma
		if p.tokens[p.pos].Type == COMMA {
			p.pos++
			continue
		}

		// param name
		name := p.expectIdent().Lexeme

		typ := p.parseType()

		funcNode.Params = append(funcNode.Params, Param{
			Name: name,
			Type: typ,
			Line: typ.Line,
		})
	}

	// )
	p.expectType(CLS_PAREN)

	// return signature
	funcNode.Return = p.parseRetSign()

	funcNode.Body = p.parseBlock()

	return &funcNode
}

// Block Parsing
func (p *Parser) parseBlock() *FrameBlock {
	p.expectType(OPN_BRACE)
	frameBlock := &FrameBlock{Stmts: []Statement{}}

	p.skip()
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type != CLS_BRACE {

		p.skip()

		frameBlock.Stmts = append(frameBlock.Stmts, p.parseStatement())

		p.skip()
	}
	p.expectType(CLS_BRACE)

	return frameBlock
}

// Top-Level Parsers

// parse package
func (p *Parser) parsePackage() string {
	p.expectType(PACKAGE)
	tok := p.currentToken()
	pkg := tok.Lexeme
	p.pos++
	return pkg
}

// import
func (p *Parser) parseImport() []string {
	p.expectType(IMPORT)
	p.expectType(OPN_PAREN)

	libs := []string{}

	for {
		p.skipNewlines()

		if p.currentToken().Type == CLS_PAREN {
			break
		}

		pkg := p.expectIdent()
		libs = append(libs, pkg.Lexeme)
	}

	p.expectType(CLS_PAREN)
	return libs
}

func (p *Parser) parseStruct() *Struct {

	p.expectType(TYPE)
	name := p.expectIdent()
	p.expectType(STRUCT)
	p.expectType(OPN_BRACE)

	fields := []Field{}
	for {
		p.skip()

		if p.currentToken().Type == CLS_BRACE {
			break
		}

		tok := p.currentToken()
		if tok.Type != IDENT {
			p.appendErrorf("expected field name, got %v", tok.Line, tok)
		}
		fields = append(fields, p.parseField())
	}

	p.expectType(CLS_BRACE)
	return &Struct{Name: name.Lexeme, Fields: fields, Line: p.currentToken().Line}
}

func (p *Parser) parseField() Field {
	nameTok := p.expectIdent()

	fieldType := p.parseType()

	return Field{
		Name: nameTok.Lexeme,
		Type: fieldType,
		Line: nameTok.Line,
	}
}

func (p *Parser) parseFieldAssign() Field {
	nameTok := p.expectIdent()
	typeTok := p.expectIdent()

	// Same here: Wrap the string Lexeme into the aster.Type struct
	fieldType := Type{
		Name:     typeTok.Lexeme,
		PtrDepth: 0,
		IsArray:  false,
		Line:     typeTok.Line,
	}

	return Field{
		Name: nameTok.Lexeme,
		Type: fieldType,
		Line: typeTok.Line,
	}
}

func (p *Parser) parseVarDecl() *VarDeclar {

	p.pos++ // 1. Consume 'var'
	name := p.expectIdent()

	var typeNode *Type = nil
	var value Expression = nil

	// 2. Updated Check: Type can start with IDENT or OPN_BRACK
	curr := p.currentToken().Type
	if curr == IDENT || curr == OPN_BRACK || curr == STAR {
		// Use your parseType logic here to keep it contained
		tmp := p.parseType()
		typeNode = &tmp
	}

	// 3. Check for Assignment (=)
	if p.currentToken().Type == ASSIGN {
		p.pos++
		value = p.parseExpr()
	}

	//	p.skipNewlines()
	return &VarDeclar{
		Name:  name.Lexeme,
		Type:  typeNode,
		Value: value,
		Line:  name.Line,
	}
}

func (p *Parser) parseStructLiteral(typeName string) Expression {
	p.expectType(OPN_BRACE) // Consumes '{'
	fields := []FieldInit{}

	for p.pos < len(p.tokens) && p.tokens[p.pos].Type != CLS_BRACE {
		var value Expression
		line := p.currentToken().Line

		// Check for named field (e.g., name: "adam")
		// Peek to see if current is IDENT and next is COLON
		if p.currentToken().Type == IDENT && p.peekToken().Type == COLON {
			nameTok := p.expectIdent() // Consume field name
			p.expectType(COLON)        // Consume ':'
			value = p.parseExpr()      // Consume the value

			fields = append(fields, FieldInit{
				Name:  nameTok.Lexeme,
				Value: value,
				Line:  line,
			})
		} else {
			// Handle positional value or unknown expression
			value = p.parseExpr()
			if value == nil {
				// Prevent infinite loop: if we can't parse, skip this token
				p.advanceToken()
				continue
			}
			fields = append(fields, FieldInit{
				Name:  "", // No name for positional
				Value: value,
				Line:  line,
			})
		}

		// Consume optional comma after field
		if p.currentToken().Type == COMMA {
			p.advanceToken()
		}

		// Safety break if we hit EOF
		if p.currentToken().Type == EOF {
			break
		}
	}

	p.expectType(CLS_BRACE) // Consumes '}'
	return &StructLiteral{
		Type:   Type{Name: typeName, Line: p.currentToken().Line},
		Fields: fields,
		Line:   p.currentToken().Line,
	}
}

// AST Builder
func (p *Parser) Builder(data []byte) *AST {
	p.tokens = Lexer(string(data))

	ast := &AST{
		Decls: make([]Decl, 0),
	}

	for p.pos < len(p.tokens) {
		token := p.tokens[p.pos]

		switch token.Type {
		case PACKAGE:
			ast.Package.Name = p.parsePackage()
		case STRUCT:

		case IMPORT:
			ast.Imports = p.parseImport()

		case TYPE:
			st := p.parseStruct()
			ast.Decls = append(ast.Decls, st)

		case FUNC:
			ast.Decls = append(ast.Decls, p.parseFunc())

		case VAR:
			ast.Decls = append(ast.Decls, p.parseVarDecl())

			//case IDENT:
			//	p.errors = append(p.errors, "cant use identifier at level package")
			//	println("catch error")
			//  p.pos++

		default:
			if token.Type == IDENT || token.Type == DEFINE || token.Type == ASSIGN {
				p.Errors = append(p.Errors, fmt.Sprintf(
					"line %d: non-declaration statement outside function body", token.Line))
				p.synchronize()
			} else {
				p.pos++
			}
		}
	}

	return ast
}

// TODO: Implement explicit memory casting for system programming.
// Goal: Support direct memory addressing safely (e.g., var reg *int = (*int)(0x40021000))
// 1. Parser: Add support for Type Casting expressions (Type)(Expr).
// 2. TypeChecker: Ensure casting to pointers is only allowed from integer types.
// 3. TypeChecker: Enforce PtrDepth <= 1 during the cast to maintain Fox's shallow pointer rule.
// 4. Codegen: Emit standard C-style casts for the generated code.
// end
