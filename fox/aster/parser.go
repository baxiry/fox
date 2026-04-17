package aster

import (
	"fmt"
)

type Parser struct {
	tokens      []Token
	pos         int
	inCondition bool
}

// New Parser
func NewParser() *Parser {
	return &Parser{
		tokens: make([]Token, 0),
		pos:    0,
	}
}

func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == NEW_LINE {
		p.pos++
	}
}

func (p *Parser) parseUnary() Expression {

	if p.pos < len(p.tokens) {
		switch p.tokens[p.pos].Type {

		case NOT, MINUS, STAR, AMP:
			op := p.tokens[p.pos]
			p.pos++
			return UnaryExpr{Op: op.Lexeme, Expr: p.parseUnary()}
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
			field := p.expectIdent().Lexeme
			expr = FieldAccessExpr{Object: expr, Field: field}

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

		op := p.tokens[p.pos]

		prec, ok := precedence(op.Type)
		if !ok || prec < minPrec {
			break
		}

		p.pos++ // consume operator

		right := p.parseBinary(prec + 1)

		left = BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
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
	if p.pos >= len(p.tokens) {
		panic("unexpected end of input while parsing expression")
	}
	tok := p.tokens[p.pos]

	switch tok.Type {
	case IDENT:
		p.pos++
		// Check for function call: name()
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == OPN_PAREN {
			return p.parseCall(tok.Lexeme)
		}
		// Struct literals will be handled by parsePostfix() now
		return IdentExpr{Name: tok.Lexeme}

	case BLANK:
		p.pos++
		return IdentExpr{Name: "_"}

	case INT, FLOAT:
		p.pos++
		return NumberExpr{Literal: tok.Lexeme}

	case STRING:
		p.pos++
		return StringExpr{Literal: tok.Lexeme}

	case OPN_PAREN:
		p.pos++
		expr := p.parseExpr()
		p.expectType(CLS_PAREN)
		return expr

	default:
		if tok.Type == STAR {
			return p.parseUnary()
		}
		panic(fmt.Sprintf("expected expression at line %d, got %s (%q)", tok.Line, tok.Type, tok.Lexeme))
	}
}
func (p *Parser) parseType() Type {

	if p.pos >= len(p.tokens) {
		panic("unexpected end of input while parsing type")
	}

	if p.tokens[p.pos].Type == AMP { // &
		panic(fmt.Sprintf(
			"syntax error at line %d, column %d: cannot use & in parameter signature",
			p.tokens[p.pos].Line, p.tokens[p.pos].Column,
		))
	}

	ptrDepth := 0
	for p.pos < len(p.tokens) && p.tokens[p.pos].Type == STAR {
		ptrDepth++
		p.pos++
	}

	if p.pos >= len(p.tokens) {
		panic("unexpected end of input while parsing type after pointers")
	}

	name := p.expectIdent().Lexeme

	return Type{
		Name:     name,
		PtrDepth: ptrDepth,
	}
}

// parse * and /
func (p *Parser) parseMul() Expression {
	left := p.parseUnary()
	for p.tokens[p.pos].Type == STAR || p.tokens[p.pos].Type == SLASH {
		op := p.tokens[p.pos]
		p.pos++
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Op: op.Lexeme, Right: right}
	}
	return left
}

// Bitwise AND &
func (p *Parser) parseBitwiseAnd() Expression {
	left := p.parseAdd()

	for p.tokens[p.pos].Lexeme == "&" {
		op := p.tokens[p.pos]
		p.pos++
		right := p.parseAdd()

		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
		}
	}

	return left
}

// Equality == & !=
func (p *Parser) parseEquality() Expression {
	left := p.parseBitwiseAnd()

	for p.tokens[p.pos].Lexeme == "==" || p.tokens[p.pos].Lexeme == "!=" {
		op := p.tokens[p.pos]
		p.pos++
		right := p.parseBitwiseAnd()

		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
		}
	}

	return left
}

/// TODO   parseTerm

// parse + and -
func (p *Parser) parseAdd() Expression {
	left := p.parseMul()
	for p.tokens[p.pos].Lexeme == "+" || p.tokens[p.pos].Lexeme == "-" {
		op := p.tokens[p.pos]
		p.pos++
		right := p.parseMul()
		left = &BinaryExpr{Left: left, Op: op.Lexeme, Right: right}
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
func (p *Parser) parseFunc() Func {
	funcNode := Func{}

	// parse func word then name func
	p.expectType(FUNC)
	funcNode.FuncName = p.expectIdent().Lexeme

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
		})
	}

	// )
	p.expectType(CLS_PAREN)

	// return signature
	funcNode.Returns = p.parseRetSign()

	funcNode.Body = p.parseBlock()

	return funcNode
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
	pkg := p.tokens[p.pos].Lexeme
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

		if p.tokens[p.pos].Type == CLS_PAREN {
			break
		}

		pkg := p.expectIdent()
		libs = append(libs, pkg.Lexeme)
	}

	p.expectType(CLS_PAREN)
	return libs
}

func (p *Parser) parseStruct() Struct {

	p.expectType(TYPE)
	name := p.expectIdent()
	p.expectType(STRUCT)
	p.expectType(OPN_BRACE)

	fields := []Field{}
	for {
		p.skip()

		if p.tokens[p.pos].Type == CLS_BRACE {
			break
		}

		if p.tokens[p.pos].Type != IDENT {
			panic(fmt.Sprintf("expected field name, got %v", p.tokens[p.pos]))
		}
		fields = append(fields, p.parseField())
	}

	p.expectType(CLS_BRACE)
	return Struct{Name: name.Lexeme, Fields: fields}
}

func (p *Parser) parseField() Field {
	nameTok := p.expectIdent()
	typeTok := p.expectIdent()
	return Field{Name: nameTok.Lexeme, Type: typeTok.Lexeme}
}

func (p *Parser) parseFieldAssign() Field {
	nameTok := p.expectIdent()
	typeTok := p.expectIdent()
	return Field{Name: nameTok.Lexeme, Type: typeTok.Lexeme}
}

func (p *Parser) parseVarDecl() VarDeclar {
	// 1. Consume the 'var' keyword
	if p.tokens[p.pos].Type == VAR {
		p.pos++
	}

	// 2. Expect the variable name (e.g., 'obj' or 'i')
	name := p.expectIdent()

	var typeNode *Type = nil
	var value Expression = nil

	// 3. Optional: Check if the next token is a Type (IDENT)
	// In 'var i int', the second IDENT is the type
	if p.tokens[p.pos].Type == IDENT {
		typeName := p.expectIdent()
		typeNode = &Type{Name: typeName.Lexeme}
	}

	// 4. Optional: Check for assignment operator '='
	// Note: Use ASSIGN (=) here, not DEFINE (:=)
	if p.tokens[p.pos].Type == ASSIGN {
		p.pos++
		// Parse the expression on the right side
		value = p.parseExpr()
	}

	return VarDeclar{
		Name:  name.Lexeme,
		Type:  typeNode,
		Value: value,
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
			})
		} else {
			// Handle positional value or expressions
			value := p.parseExpr()
			fields = append(fields, FieldInit{
				Name:  "",
				Value: value,
			})
		}
		// Consume optional comma after field
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == COMMA {
			p.expectType(COMMA)
		}
	}
	p.expectType(CLS_BRACE)
	return StructLiteral{
		Type:   Type{Name: typeName},
		Fields: fields,
	}
}

// AST Builder
func (p *Parser) Builder(data []byte) *AST {
	p.tokens = Lexer(string(data))

	ast := &AST{}

	for p.pos < len(p.tokens) {
		token := p.tokens[p.pos]

		switch token.Lexeme {
		case "package":
			ast.PackageName = p.parsePackage()

		case "import":
			ast.Imports = p.parseImport()

		case "type":
			ast.Structs = append(ast.Structs, p.parseStruct())

		case "fn":
			ast.Funcs = append(ast.Funcs, p.parseFunc())

		case "var":
			ast.Vars = append(ast.Vars, p.parseVarDecl())

		default:
			p.pos++
		}
	}
	return ast
}
