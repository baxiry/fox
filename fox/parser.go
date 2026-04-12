package main

import (
	"fmt"
)

/*
	func parseLogicAnd(tokens []Token, pos *int) Expression {
		left := parseEquality(tokens, pos)

		for *pos < len(tokens) && tokens[*pos].Type == AND {
			op := tokens[*pos]
			*pos++

			right := parseEquality(tokens, pos)

			left = &BinaryExpr{
				Left:  left,
				Op:    op.Lexeme,
				Right: right,
			}
		}

		return left
	}

	func parseLogicOr(tokens []Token, pos *int) Expression {
		left := parseLogicAnd(tokens, pos)

		for *pos < len(tokens) && tokens[*pos].Type == OR {
			op := tokens[*pos]
			*pos++

			right := parseLogicAnd(tokens, pos)

			left = &BinaryExpr{
				Left:  left,
				Op:    op.Lexeme,
				Right: right,
			}
		}

		return left
	}
*/
func isStartOfPrimary(tok Token) bool {
	switch tok.Type {

	case IDENT,
		INT, FLOAT, STRING,
		TRUE, FALSE,
		OPN_PAREN:

		return true
	}

	return false
}

func parseUnary(tokens []Token, pos *int) Expression {
	//skip(tokens, pos)

	if *pos < len(tokens) {
		switch tokens[*pos].Type {
		case NOT, MINUS, AMP, STAR:
			op := tokens[*pos]
			*pos++

			right := parseUnary(tokens, pos)
			return UnaryExpr{Op: op.Lexeme, Expr: right}
		}
	}

	return parsePostfix(tokens, pos)
}

func parsePostfix(tokens []Token, pos *int) Expression {
	skip(tokens, pos)

	// IMPORTANT: start from primary ONLY (no unary call here)
	expr := parsePrimary(tokens, pos)

	for *pos < len(tokens) {
		switch tokens[*pos].Type {

		case DOT:
			*pos++
			skip(tokens, pos)
			field := expectIdent(tokens, pos).Lexeme
			expr = FieldAccessExpr{Object: expr, Field: field}

		default:
			return expr
		}
	}

	return expr
}

func parseBinary(tokens []Token, pos *int, minPrec int) Expression {
	left := parseUnary(tokens, pos)

	for *pos < len(tokens) {
		op := tokens[*pos]

		prec, ok := precedence(op.Type)
		if !ok || prec < minPrec {
			break
		}

		*pos++ // consume operator

		right := parseBinary(tokens, pos, prec+1)

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
func getPrecedence(t TokenType) int {
	switch t {
	case STAR:
		return 5
	case PLUS:
		return 4
	case EQ, NEQ:
		return 3
	case AND:
		return 2
	case OR:
		return 1
	default:
		return -1
	}
}

func isUnaryStart(tok Token) bool {
	return tok.Type == AMP ||
		tok.Type == STAR ||
		tok.Type == NOT ||
		tok.Type == MINUS
}

func parsePrimary(tokens []Token, pos *int) Expression {
	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing expression")
	}
	tok := tokens[*pos]
	fmt.Printf("PRIMARY -> pos=%d token=%+v\n", *pos, tokens[*pos])

	tok = tokens[*pos]
	switch tok.Type {
	case IDENT:
		//name := tok.Lexeme
		*pos++
		if *pos < len(tokens) && tokens[*pos].Type == OPN_PAREN {
			return parseCall(tok.Lexeme, tokens, pos)
		}

		return IdentExpr{Name: tok.Lexeme}

	case INT, FLOAT:
		*pos++
		return NumberExpr{Literal: tok.Lexeme}

	case STRING:
		*pos++
		return StringExpr{Literal: tok.Lexeme}

	case OPN_PAREN:
		*pos++
		expr := parseExpr(tokens, pos)
		expectType(tokens, pos, CLS_PAREN)
		return expr

	default:
		if tok.Type == STAR {
			return parseUnary(tokens, pos)
		}
		panic(fmt.Sprintf("expected expression at line %d, got %s (%q)", tok.Line, tok.Type, tok.Lexeme))
	}
}

func parseType(tokens []Token, pos *int) Type {

	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing type")
	}

	if tokens[*pos].Type == AMP { // &
		panic(fmt.Sprintf(
			"syntax error at line %d, column %d: cannot use & in parameter signature",
			tokens[*pos].Line, tokens[*pos].Column,
		))
	}

	ptrDepth := 0
	for *pos < len(tokens) && tokens[*pos].Type == STAR {
		ptrDepth++
		*pos++
	}

	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing type after pointers")
	}

	name := expectIdent(tokens, pos).Lexeme

	return Type{
		Name:     name,
		PtrDepth: ptrDepth,
	}
}

// parse * and /
func parseMul(tokens []Token, pos *int) Expression {
	left := parseUnary(tokens, pos)
	for tokens[*pos].Type == STAR || tokens[*pos].Type == SLASH {
		op := tokens[*pos]
		*pos++
		right := parseUnary(tokens, pos)
		left = &BinaryExpr{Left: left, Op: op.Lexeme, Right: right}
	}
	return left
}

// Bitwise AND &
func parseBitwiseAnd(tokens []Token, pos *int) Expression {
	left := parseAdd(tokens, pos)

	for tokens[*pos].Lexeme == "&" {
		op := tokens[*pos]
		*pos++
		right := parseAdd(tokens, pos)

		left = &BinaryExpr{
			Left:  left,
			Op:    op.Lexeme,
			Right: right,
		}
	}

	return left
}

// Equality == & !=
func parseEquality(tokens []Token, pos *int) Expression {
	left := parseBitwiseAnd(tokens, pos)

	for tokens[*pos].Lexeme == "==" || tokens[*pos].Lexeme == "!=" {
		op := tokens[*pos]
		*pos++
		right := parseBitwiseAnd(tokens, pos)

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
func parseAdd(tokens []Token, pos *int) Expression {
	left := parseMul(tokens, pos)
	for tokens[*pos].Lexeme == "+" || tokens[*pos].Lexeme == "-" {
		op := tokens[*pos]
		*pos++
		right := parseMul(tokens, pos)
		left = &BinaryExpr{Left: left, Op: op.Lexeme, Right: right}
	}
	return left
}

// top-level expression
func parseExpr(tokens []Token, pos *int) Expression {

	return parseBinary(tokens, pos, 0)
}

// Functions
func parseFunc(tokens []Token, pos *int) FuncDecl {
	funcNode := FuncDecl{}

	// parse func word then name func
	expectType(tokens, pos, FUNC)
	funcNode.Name = expectIdent(tokens, pos).Lexeme

	// parse open paren ( then params
	expectType(tokens, pos, OPN_PAREN)

	for tokens[*pos].Type != CLS_PAREN {
		// skip comma
		if tokens[*pos].Type == COMMA {
			*pos++
			continue
		}

		// param name
		name := expectIdent(tokens, pos).Lexeme

		typ := parseType(tokens, pos)

		funcNode.Params = append(funcNode.Params, ParamDecl{
			Name: name,
			Type: typ,
		})
	}

	// )
	expectType(tokens, pos, CLS_PAREN)

	// return signature
	funcNode.Returns = parseRetSign(tokens, pos)

	funcNode.Body = parseBlock(tokens, pos)

	return funcNode
}

// Block Parsing
func parseBlock(tokens []Token, pos *int) *FrameBlock {
	expectType(tokens, pos, OPN_BRACE)
	frameBlock := &FrameBlock{Stmts: []Statement{}}

	skip(tokens, pos)
	for *pos < len(tokens) && tokens[*pos].Type != CLS_BRACE {
		skip(tokens, pos)

		frameBlock.Stmts = append(frameBlock.Stmts, parseStatement(tokens, pos))

		skip(tokens, pos)
	}
	expectType(tokens, pos, CLS_BRACE)

	return frameBlock
}

// Top-Level Parsers

// parse package
func parsePackage(tokens []Token, pos *int) string {
	expectType(tokens, pos, PACKAGE)
	pkg := tokens[*pos].Lexeme
	*pos++
	return pkg
}

// import
func parseImport(tokens []Token, pos *int) []string {
	expectType(tokens, pos, IMPORT)
	expectType(tokens, pos, OPN_PAREN)

	libs := []string{}

	for {
		skipNewlines(tokens, pos)

		if tokens[*pos].Type == CLS_PAREN {
			break
		}

		pkg := expectIdent(tokens, pos)
		libs = append(libs, pkg.Lexeme)
	}

	expectType(tokens, pos, CLS_PAREN)
	return libs
}

func parseStruct(tokens []Token, pos *int) StructDecl {
	expectType(tokens, pos, TYPE)
	name := expectIdent(tokens, pos)
	expectType(tokens, pos, STRUCT)
	expectType(tokens, pos, OPN_BRACE)

	fields := []FieldDecl{}
	for {
		skip(tokens, pos)

		if tokens[*pos].Type == CLS_BRACE {
			break
		}

		if tokens[*pos].Type != IDENT {
			panic(fmt.Sprintf("expected field name, got %v", tokens[*pos]))
		}
		fields = append(fields, parseField(tokens, pos))
	}

	expectType(tokens, pos, CLS_BRACE)
	return StructDecl{Name: name.Lexeme, Fields: fields}
}

func parseField(tokens []Token, pos *int) FieldDecl {
	nameTok := expectIdent(tokens, pos)
	typeTok := expectIdent(tokens, pos)
	return FieldDecl{Name: nameTok.Lexeme, Type: typeTok.Lexeme}
}

// AST Builder
func astBuilder(tokens []Token) *AST {
	p := 0
	pos := &p
	ast := &AST{}

	fmt.Println("len of tokens  : ", len(tokens))
	for *pos < len(tokens) {
		token := tokens[*pos]

		switch token.Lexeme {
		case "package":
			ast.PackageName = parsePackage(tokens, pos)

		case "import":
			ast.Imports = parseImport(tokens, pos)

		case "type":
			ast.Structs = append(ast.Structs, parseStruct(tokens, pos))

		case "fn":
			ast.Funcs = append(ast.Funcs, parseFunc(tokens, pos))

		default:
			*pos++
		}
	}
	return ast
}
