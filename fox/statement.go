package main

// ===== Interfaces =====

type Statement interface {
	isStatement()
}

// ===== AST Nodes (Statements) =====

type BreakNode struct {
	Tok Token
}

func (BreakNode) isStatement() {}

type ContinueNode struct {
	Tok Token
}

func (ContinueNode) isStatement() {}

type ReturnStmt struct {
	RetValues []Expression
}

func (ReturnStmt) isStatement() {}

type IfStmt struct {
	Cond Expression
	Then []Statement
	Else []Statement
}

func (IfStmt) isStatement() {}

type ForStmt struct {
	Init Statement
	Cond Expression
	Post Statement
	Body []Statement
}

func (ForStmt) isStatement() {}

type AssignStmt struct {
	Name  string
	Op    string // "=" or ":="
	Value Expression
}

func (AssignStmt) isStatement() {}

type DefineStmt struct {
	Name  string
	Value Expression
}

func (DefineStmt) isStatement() {}

type ExprStmt struct {
	Expr Expression
}

func (ExprStmt) isStatement() {}

// ===== Parsing Helpers =====

func parseStatement(tokens []Token, pos *int) Statement {
	tok := tokens[*pos]

	switch tok.Value {
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

// ===== Block Parsing =====

func parseBlock(tokens []Token, pos *int) []Statement {
	stmts := []Statement{}
	expectType(tokens, pos, Delimiter.LBrace)
	for *pos < len(tokens) && tokens[*pos].Type != Delimiter.RBrace {
		stmts = append(stmts, parseStatement(tokens, pos))
	}
	expectType(tokens, pos, Delimiter.RBrace)
	return stmts
}

// ===== Statement Parsers =====

func parseIf(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, Keyword.If)
	cond := parseExpr(tokens, pos)
	thenBlock := parseBlock(tokens, pos)

	var elseBlock []Statement
	if *pos < len(tokens) && tokens[*pos].Type == Keyword.Else {
		*pos++
		elseBlock = parseBlock(tokens, pos)
	}

	return IfStmt{Cond: cond, Then: thenBlock, Else: elseBlock}
}

func parseFor(tokens []Token, pos *int) Statement {
	expectType(tokens, pos, Keyword.For)
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
		forStmt.Cond = parseExpr(tokens, pos)
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
	expectType(tokens, pos, Keyword.Return)
	values := []Expression{}

	if tokens[*pos].Type != Delimiter.Semic && tokens[*pos].Type != Delimiter.RBrace {
		values = append(values, parseExpr(tokens, pos))
		for *pos < len(tokens) && tokens[*pos].Value == "," {
			*pos++
			values = append(values, parseExpr(tokens, pos))
		}
	}

	return ReturnStmt{RetValues: values}
}

func parseExprStatement(tokens []Token, pos *int) Statement {
	expr := parseExpr(tokens, pos)
	return ExprStmt{Expr: expr}
}

// ===== Assignment / Definition Parsers =====

func parseRetSign(tokens []Token, pos *int) []ReturnSig {
	var retSigns []ReturnSig

	for *pos < len(tokens) && tokens[*pos].Value != Delimiter.LBrace {
		if tokens[*pos].Value == Delimiter.Comma && *pos+1 < len(tokens) && tokens[*pos+1].Value != Delimiter.LBrace {
			*pos++
		}
		tok := expectIdent(tokens, pos)
		retSigns = append(retSigns, ReturnSig{
			Name: tok.Value,
			Type: tok.Type,
		})
	}

	return retSigns
}
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
		Name:  name,
		Op:    opTok.Value,
		Value: value,
	}
}

func parseDefine(tokens []Token, pos *int) Statement {
	name := tokens[*pos].Value
	*pos++
	expectType(tokens, pos, Operator.Define)
	val := parseExpr(tokens, pos)
	return DefineStmt{Name: name, Value: val}
}

// ----- مساعدة للـ For init/post (دمج تعريف أو تعيين) -----
func parseDefOrAssign(tokens []Token, pos *int) Statement {
	if *pos+1 < len(tokens) && tokens[*pos+1].Type == Operator.Define {
		return parseDefine(tokens, pos)
	}
	return parseAssign(tokens, pos)
}
