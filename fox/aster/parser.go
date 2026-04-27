package aster

import (
	"fmt"
)

type Parser struct {
	tokens      []Token
	pos         int
	inCondition bool
	Errors      []string
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
			return UnaryExpr{Op: op.Lexeme, Expr: p.parseUnary(), Line: op.Line}
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

		case DOT:
			p.pos++
			p.skip()
			field := p.expectIdent()

			if field.Type == ERROR {
				p.synchronize() // Jump to the next safe statement
				return nil
			}
			expr = FieldAccessExpr{Object: expr, Field: field.Lexeme, Line: field.Line}

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
			if ident, ok := expr.(IdentExpr); ok {
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

		left = BinaryExpr{
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

func (p *Parser) parsePrimary() Expression {

	tok := p.currentToken()

	switch tok.Type {
	case BOOL:
		val := p.currentToken().Lexeme
		line := p.currentToken().Line
		p.pos++
		return BoolExpr{Literal: val, Line: line}

	case IDENT:
		p.pos++
		if p.currentToken().Type == OPN_PAREN {
			return p.parseCall(tok.Lexeme)
		}
		return IdentExpr{Name: tok.Lexeme, Line: tok.Line}

	case BLANK:
		p.pos++
		return IdentExpr{Name: "_"}

	case INT, FLOAT:
		p.pos++
		return NumberExpr{Literal: tok.Lexeme, Line: tok.Line}

	case STRING:
		p.pos++
		return StringExpr{Literal: tok.Lexeme, Line: tok.Line}

	case OPN_PAREN:
		p.pos++
		expr := p.parseExpr()

		p.expectType(CLS_PAREN)
		return expr

	case EOF:

		p.appendErrorf("unexpected end of input while parsing expression", tok.Line)
		return nil

	default:
		//
		if tok.Type == STAR {
			return p.parseUnary()
		}
		p.appendErrorf("expected expression, but found %s (%q)",
			tok.Line, tok.Type, tok.Lexeme)
		p.synchronize()
		return nil
	}
}

func (p *Parser) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	finalMsg := fmt.Sprintf("line %d: %s", line, msg)
	p.Errors = append(p.Errors, finalMsg)
}

func (p *Parser) parseType() Type {

	if p.pos >= len(p.tokens) {
		p.appendErrorf("unexpected end of input while parsing type", p.currentToken().Line)
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
		return Type{Line: name.Line}
	}

	return Type{
		Name:     name.Lexeme,
		PtrDepth: ptrDepth,
		Line:     name.Line,
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
	left := p.parseBitwiseAnd()
	tok := p.currentToken()
	for tok.Lexeme == "==" || tok.Lexeme == "!=" {
		op := p.tokens[p.pos]
		p.pos++
		right := p.parseBitwiseAnd()

		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
			Line:  tok.Line,
		}
	}

	return left
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

func (p *Parser) parseExprList() []Expression {
	exprs := []Expression{}

	// Parse the first expression
	exprs = append(exprs, p.parseExpr())

	// Keep parsing as long as there are commas
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
		p.pos++ // consume COMMA
		exprs = append(exprs, p.parseExpr())
	}

	return exprs
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
	funcNode.Returns = p.parseRetSign()

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
	typeTok := p.expectIdent()
	//line := p.currentToken().Line
	return Field{Name: nameTok.Lexeme, Type: typeTok.Lexeme, Line: typeTok.Line}
}

func (p *Parser) parseFieldAssign() Field {
	nameTok := p.expectIdent()
	typeTok := p.expectIdent()
	return Field{Name: nameTok.Lexeme, Type: typeTok.Lexeme, Line: typeTok.Line}
}

func (p *Parser) parseVarDecl() *VarDeclar {
	p.pos++ // 1. Consume 'var'

	name := p.expectIdent()

	var typeNode *Type = nil
	var value Expression = nil

	// 2. Check for Type (IDENT)
	// IMPORTANT: Check the CURRENT token here
	if p.currentToken().Type == IDENT {
		tok := p.currentToken()
		typeName := p.expectIdent()
		typeNode = &Type{Name: typeName.Lexeme, Line: tok.Line}
	}

	// 3. Check for Assignment (=)
	// IMPORTANT: Check the CURRENT token again
	if p.currentToken().Type == ASSIGN {
		p.pos++ // consume '='
		value = p.parseExpr()
	}

	return &VarDeclar{
		Name:  name.Lexeme,
		Type:  typeNode,
		Value: value,
		Line:  name.Line, // Use name.Line for consistency
	}
}

func (p *Parser) parseStructLiteral(typeName string) Expression {
	p.expectType(OPN_BRACE)
	fields := []FieldInit{}
	// Loop until we find the closing brace or reach end of tokens
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type != CLS_BRACE {
		// Check for named field: Key: value
		if p.pos+1 < len(p.tokens) && p.tokens[p.pos].Type == IDENT && p.tokens[p.pos+1].Type == COLON {
			nameTok := p.expectIdent()
			p.expectType(COLON)
			value := p.parseExpr()
			fields = append(fields, FieldInit{
				Name:  nameTok.Lexeme,
				Value: value,
				Line:  value.GetLine(),
			})
		} else {
			// Handle positional value or expressions
			value := p.parseExpr()
			fields = append(fields, FieldInit{
				Name:  "",
				Value: value,
				Line:  value.GetLine(),
			})
		}
		// Consume optional comma after field
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
			p.expectType(COMMA)
		}
	}
	p.expectType(CLS_BRACE)
	return StructLiteral{
		Type:   Type{Name: typeName, Line: p.currentToken().Line},
		Fields: fields,
		Line:   p.currentToken().Line,
	}
}

// AST Builder
func (p *Parser) Builder(data []byte) *AST {
	p.tokens = Lexer(string(data))

	ast := &AST{}

	for p.pos < len(p.tokens) {
		token := p.tokens[p.pos]

		switch token.Type {
		case PACKAGE:
			ast.PackageName = p.parsePackage()

		case IMPORT:
			ast.Imports = p.parseImport()

		case TYPE:
			ast.Structs = append(ast.Structs, p.parseStruct())

		case FUNC:
			ast.Funcs = append(ast.Funcs, p.parseFunc())

		case VAR:
			ast.Vars = append(ast.Vars, p.parseVarDecl())

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

// end
