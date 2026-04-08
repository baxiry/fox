package main

import (
	"fmt"
)

func parseUnary(tokens []Token, pos *int) Expression {
	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing unary expression")
	}

	tok := tokens[*pos]

	if tok.Type == STAR || tok.Type == AND || tok.Type == NOT || tok.Type == MINUS {
		(*pos)++
		if *pos >= len(tokens) {
			panic(fmt.Sprintf("unexpected end of input after unary operator '%s'", tok.Lexeme))
		}
		expr := parseUnary(tokens, pos)
		return UnaryExpr{Op: tok.Lexeme, Expr: expr, Line: tok.Line}
	}

	return parsePostfix(tokens, pos)
}

func parsePostfix(tokens []Token, pos *int) Expression {
	expr := parsePrimary(tokens, pos)

	for *pos < len(tokens) && tokens[*pos].Lexeme == "." {
		*pos++ // consume '.'
		field := expectIdent(tokens, pos).Lexeme
		expr = FieldAccessExpr{Object: expr, Field: field}
	}

	return expr
}

func parsePrimary(tokens []Token, pos *int) Expression {
	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing expression")
	}

	tok := tokens[*pos]
	switch tok.Type {
	case IDENT:
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
		panic(fmt.Sprintf("expected expression at line %d, got %s (%q)", tok.Line, tok.Type, tok.Lexeme))
	}
}

func parseType(tokens []Token, pos *int) Type {

	if *pos >= len(tokens) {
		panic("unexpected end of input while parsing type")
	}

	if tokens[*pos].Type == AND { // &
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
		left = BinaryExpr{Left: left, Op: op.Lexeme, Right: right}
	}
	return left
}

// Equality == & !=
func parseEquality(tokens []Token, pos *int) Expression {
	left := parseAdd(tokens, pos)

	for tokens[*pos].Lexeme == "==" || tokens[*pos].Lexeme == "!=" {
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
	// TODO: need && ||. ex: parseLogic()
	return parseEquality(tokens, pos)
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

	// parse frameBlock | body
	//expectType(tokens, pos, OPN_BRACE)
	/*
		frameBlock := &FrameBlock{Stmts: []Statement{}}
		for tokens[*pos].Type != CLS_BRACE {
			stmt := parseStatement(tokens, pos)
			frameBlock.Stmts = append(frameBlock.Stmts, stmt)
			funcNode.Body = frameBlock
		}

		expectType(tokens, pos, CLS_BRACE)
	*/
	funcNode.Body = parseBlock(tokens, pos)

	return funcNode
}

// Block Parsing

func parseBlock(tokens []Token, pos *int) *FrameBlock {
	expectType(tokens, pos, OPN_BRACE)
	frameBlock := &FrameBlock{Stmts: []Statement{}}

	for *pos < len(tokens) && tokens[*pos].Type != CLS_BRACE {
		frameBlock.Stmts = append(frameBlock.Stmts, parseStatement(tokens, pos))
	}
	expectType(tokens, pos, CLS_BRACE)

	return frameBlock
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

// ===== Top-Level Parsers =====

func parsePackage(tokens []Token, pos *int) string {
	expectType(tokens, pos, PACKAGE)
	pkg := tokens[*pos].Lexeme
	*pos++
	return pkg
}

func parseImport(tokens []Token, pos *int) []string {
	expectType(tokens, pos, IMPORT)
	expectType(tokens, pos, OPN_PAREN)

	libs := []string{}
	for tokens[*pos].Lexeme != ")" {
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
	for tokens[*pos].Lexeme != "}" {
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
