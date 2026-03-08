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
	Type    string
	Results []Expression
}

type IfStmt struct {
	Cond Expression
	Then []Statement
	Else []Statement
}

type ForStmt struct {
	Init Statement
	Cond Expression
	Post Statement
	Body []Statement
}

type AssignStmt struct {
	Target Expression
	Op     string
	Value  Expression
}

type DefineStmt struct {
	Name  string
	Value Expression
}

type ExprStmt struct {
	Expr Expression
}

func (ReturnStmt) isStat()   {}
func (ContinueNode) isStat() {}
func (IfStmt) isStat()       {}
func (BreakNode) isStat()    {}
func (ForStmt) isStat()      {}
func (AssignStmt) isStat()   {}
func (DefineStmt) isStat()   {}
func (ExprStmt) isStat()     {}

//  Parsing Helpers

func parseStatement(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Value {
	case keywords.Return:
		results := parseReturn(tokens, pos)
		return results

	case keywords.If:
		return parseIf(tokens, pos)

	case keywords.For:
		return parseFor(tokens, pos)

	case keywords.Break:
		*pos++
		return BreakNode{Tok: tok}

	case keywords.Continue:
		*pos++
		return ContinueNode{Tok: tok}

	default:
		// Identifier followed by = or := → assignment/definition
		if *pos+1 < len(tokens) {
			op := tokens[*pos+1].Value
			if op == "=" {
				return parseAssign(tokens, pos)
			}
			if op == ":=" {
				return parseDefine(tokens, pos)
			}
		}
		return parseExprStatement(tokens, pos)
	}
}

// Block Parsing

func parseBlock(tokens []Token, pos *int) []Statement {
	stmts := []Statement{}
	expectType(tokens, pos, Delimiter.LBrace)
	for *pos < len(tokens) && tokens[*pos].Type != Delimiter.RBrace {
		stmts = append(stmts, parseStatement(tokens, pos))
	}
	expectType(tokens, pos, Delimiter.RBrace)
	return stmts
}

//  Statement Parsers

func parseIf(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, keywords.If)
	cond := parseExpr(tokens, pos)
	thenBlock := parseBlock(tokens, pos)

	var elseBlock []Statement
	if *pos < len(tokens) && tokens[*pos].Type == keywords.Else {
		*pos++
		elseBlock = parseBlock(tokens, pos)
	}

	return IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}
}

func parseExprUntil(tokens []Token, pos *int, stop string) Expression {
	expr := parseExpr(tokens, pos)

	for *pos < len(tokens) && tokens[*pos].Value != stop {
		op := tokens[*pos]
		if op.Kind != OperatorKind {
			break
		}
		*pos++

		right := parseExpr(tokens, pos)
		expr = BinaryExpr{
			Op:    op.Type,
			Left:  expr,
			Right: right,
		}
	}
	return expr
}

func parseFor(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, keywords.For)
	forStmt := ForStmt{}

	//  INIT
	// check ";" "{" befor init
	if tokens[*pos].Type != Delimiter.Semic && tokens[*pos].Type != Delimiter.LBrace {
		if *pos+1 < len(tokens) && (tokens[*pos+1].Type == Operator.Assign || tokens[*pos+1].Type == Operator.Define) {
			forStmt.Init = parseDefOrAssign(tokens, pos)
		} else {
			forStmt.Init = parseExprStatement(tokens, pos)
		}
	}
	expectType(tokens, pos, Delimiter.Semic) // use ;

	// CONDITION
	if tokens[*pos].Type != Delimiter.Semic && tokens[*pos].Type != Delimiter.LBrace {
		forStmt.Cond = parseExprUntil(tokens, pos, ";")
	}
	expectType(tokens, pos, Delimiter.Semic) // use ;

	// POST
	if tokens[*pos].Type != Delimiter.LBrace {
		if *pos+1 < len(tokens) && (tokens[*pos+1].Type == Operator.Assign || tokens[*pos+1].Type == Operator.Define) {
			forStmt.Post = parseDefOrAssign(tokens, pos)
		} else if tokens[*pos].Type != Delimiter.RBrace {
			forStmt.Post = parseExprStatement(tokens, pos)
		}
	}

	//  BODY
	forStmt.Body = parseBlock(tokens, pos)
	return forStmt
}

func parseReturn(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, keywords.Return)

	results := []Expression{}

	if tokens[*pos].Type != Delimiter.Semic && tokens[*pos].Type != Delimiter.RBrace {
		results = append(results, parseExpr(tokens, pos))
		for *pos < len(tokens) && tokens[*pos].Value == Delimiter.Comma {
			*pos++
			results = append(results, parseExpr(tokens, pos))
		}
	}

	return ReturnStmt{Type: "ReturnStmt", Results: results}
}

func parseExprStatement(tokens []Token, pos *int) Statement {
	expr := parseExpr(tokens, pos)
	return ExprStmt{Expr: expr}
}

func parseRetSign(tokens []Token, pos *int) []ReturnSig {

	var retSigns []ReturnSig

	for *pos < len(tokens) && tokens[*pos].Type != Delimiter.LBrace {

		// skip commas
		if tokens[*pos].Type == Delimiter.Comma {
			(*pos)++
			continue
		}

		if *pos >= len(tokens) || tokens[*pos].Type == Delimiter.LBrace {
			break
		}

		var name string

		// case: named return value  (name type)
		if *pos+1 < len(tokens) &&
			tokens[*pos].Type == Ident.Ident &&
			tokens[*pos+1].Type == Ident.Ident {

			name = expectIdent(tokens, pos).Value
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

// Assignment / Definition Parsers
func parseAssign(tokens []Token, pos *int) Statement {
	nameTok := expectType(tokens, pos, Ident.Ident)
	name := nameTok.Value

	opTok := tokens[*pos]
	if opTok.Type != Operator.Assign && opTok.Type != Operator.Define {
		panic("expected = or :=")
	}
	*pos++

	value := parseExpr(tokens, pos)

	return AssignStmt{
		Target: &IdentExpr{Name: name},
		Op:     opTok.Value,
		Value:  value,
	}
}

func parseDefine(tokens []Token, pos *int) Statement {
	name := tokens[*pos].Value
	*pos++
	expectType(tokens, pos, Operator.Define)
	val := parseExpr(tokens, pos)
	return DefineStmt{Name: name, Value: val}
}

// For init/post
func parseDefOrAssign(tokens []Token, pos *int) Statement {
	if *pos+1 < len(tokens) && tokens[*pos+1].Type == Operator.Define {
		return parseDefine(tokens, pos)
	}
	return parseAssign(tokens, pos)
}

// end
