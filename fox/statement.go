package main

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

type CompositeLiteralExpr struct {
	Type   Type         // like "Obj"
	Fields []FieldValue // field's value {"age": NumberExpr{5}}
}

type FieldValue struct {
	Field IdentExpr  // something like 'obj'
	Value Expression // NumberExpr{10}
}

func (CompositeLiteralExpr) isExpr() {}
func (VarDeclar) isStat()            {}
func (ReturnStmt) isStat()           {}
func (ContinueNode) isStat()         {}
func (IfStmt) isStat()               {}
func (BreakNode) isStat()            {}
func (ForStmt) isStat()              {}
func (Assign) isStat()               {}
func (Declar) isStat()               {}
func (ExprStmt) isStat()             {}

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

func parseStructBlock(tokens []Token, pos *int) CompositeLiteralExpr {
	// Obj{}, Obj{a:10}

	expectType(tokens, pos, OPN_BRACE)

	var compLiterExp CompositeLiteralExpr
	compLiterExp.Type = Type{Name: tokens[*pos-2].Lexeme} // , PtrDepth: ?

	fieldVal := FieldValue{}

	for tokens[*pos].Type != CLS_BRACE {
		field := expectType(tokens, pos, IDENT).Lexeme
		expectType(tokens, pos, DUBLE_DOT)
		value := parseExpr(tokens, pos)

		fieldVal.Field = IdentExpr{field}
		fieldVal.Value = value
		compLiterExp.Fields = append(compLiterExp.Fields, fieldVal)

		if tokens[*pos].Type == COMMA {
			*pos++
		}
	}
	expectType(tokens, pos, CLS_BRACE)
	return compLiterExp
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

func parseStatement(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Lexeme {
	case "var":
		return parseVarDeclar(tokens, pos)
	case "return":
		return parseReturn(tokens, pos)
	case "if":
		return parseIf(tokens, pos)
	case "for":
		return parseFor(tokens, pos)
	case "break":
		*pos++
		return BreakNode{Tok: tok}
	case "continue":
		*pos++
		return ContinueNode{Tok: tok}
	default:

		startPos := *pos
		_ = parsePostfix(tokens, pos)

		if *pos < len(tokens) {
			next := tokens[*pos]
			if next.Type == ASSIGN {
				*pos = startPos
				return parseAssign(tokens, pos)
			}
			if next.Type == DEFINE {
				*pos = startPos
				return parseDefine(tokens, pos)
			}
		}

		// fallback → expression statement
		*pos = startPos
		return parseExprStatement(tokens, pos)
	}
}

func parseStatements(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Lexeme {
	case "return":
		results := parseReturn(tokens, pos)
		return results

	case "if":
		return parseIf(tokens, pos)

	case "for":
		return parseFor(tokens, pos)

	case "break":
		*pos++
		return BreakNode{Tok: tok}

	case "continue":
		*pos++
		return ContinueNode{Tok: tok}

	default:
		if *pos < len(tokens) {
			// peek ahead
			if *pos+1 < len(tokens) {
				next := tokens[*pos+1]
				if next.Type == ASSIGN {
					return parseAssign(tokens, pos)
				}
				if next.Type == DEFINE {
					return parseDefine(tokens, pos)
				}
			}
		}

		return parseExprStatement(tokens, pos)
	}
}

//  Statement Parsers

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

func (t Token) IsOperator() bool {
	switch t.Type {
	case PLUS, MINUS, STAR, SLASH, ASSIGN, DEFINE,
		EQ, NEQ, LT, GT, LTE, GTE, AND, OR, NOT, DOT:
		return true
	default:
		return false
	}
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
		expr = BinaryExpr{
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

func parseReturn(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, RETURN)

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

func parseExprStatement(tokens []Token, pos *int) Statement {
	expr := parseExpr(tokens, pos)
	if *pos < len(tokens) && tokens[*pos].Type == SEMICOLON {
		*pos++
	}

	return ExprStmt{Expr: expr}
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

func parseAssign(tokens []Token, pos *int) Statement {
	target := parsePostfix(tokens, pos)

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

// end
