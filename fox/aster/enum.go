package aster

import (
	"fox/symbols"
)

// EnumField represents a named field inside a specific enum variant, like: id int
type EnumField struct {
	Name string        
	Type *symbols.Type 
	Line int           
}

// EnumVariant represents a single structured case within the enum, like: Active{id int}
type EnumVariant struct {
	Name   string      
	Fields []EnumField 
	Line   int         
}

// EnumDecl represents the top-level enum block declaration statement
type EnumDecl struct {
	Name     string        
	Variants []EnumVariant 
	Line     int           
}

// implements the Decl interface
func (e *EnumDecl) GetLine() int { return e.Line }
func (e *EnumDecl) isDecl()      {}

type CaseBlock struct {
	Conditions []Expression
	Body       *FrameBlock
	Line       int
}

// SwitchStmt represents the complete context-aware switch statement block
type SwitchStmt struct {
	Object Expression  
	Cases  []CaseBlock 
	Else   *FrameBlock 
	Line   int         
}

// implements the Statement interface so it can reside inside function bodies
func (s *SwitchStmt) GetLine() int { return s.Line }
func (s *SwitchStmt) isStmt()      {}

// MatchStmt represents the context-aware structural type pattern matching block
type MatchStmt struct {
	Object Expression  
	Cases  []CaseBlock 
	Else   *FrameBlock 
	Line   int         
}

// stmtNode implements the Statement interface uniquely for pattern matching
func (m *MatchStmt) isStmt() {}
func (m *MatchStmt) GetLine() int {return m.Line}

// EnumDecl
func (p *Parser) parseEnum() *EnumDecl {
	line := p.currentToken().Line

	p.expectType(ENUM)

	enumName := p.expectIdent()

	p.expectType(OPN_BRACE)

	variants := []EnumVariant{}

	for {
		p.skip()

		if p.currentToken().Type == CLS_BRACE {
			break
		}

		if p.currentToken().Type == EOF {
			p.appendErrorf("unexpected EOF inside enum declaration", p.currentToken().Line)
			break
		}

		variantName := p.expectIdent()
		fields := []EnumField{}

		if p.currentToken().Type == OPN_BRACE {
			p.expectType(OPN_BRACE)

			for {
				p.skip()

				if p.currentToken().Type == CLS_BRACE {
					break
				}

				if p.currentToken().Type == EOF {
					p.appendErrorf("unexpected EOF inside variant fields definition", p.currentToken().Line)
					break
				}

				fieldNameTok := p.expectIdent()

				fieldType := p.parseType()

				fields = append(fields, EnumField{
					Name: fieldNameTok.Lexeme,
					Type: &fieldType,
					Line: fieldNameTok.Line,
				})
			}
			p.expectType(CLS_BRACE)
		}

		variants = append(variants, EnumVariant{
			Name:   variantName.Lexeme,
			Fields: fields,
			Line:   variantName.Line,
		})
	}

	p.expectType(CLS_BRACE)

	return &EnumDecl{
		Name:     enumName.Lexeme,
		Variants: variants,
		Line:     line,
	}
}

func (p *Parser) parseEnumLiteral(parentName string) Expression {
	// 1. استهلاك النقطة الفاصلة '.' الصريحة
	p.expectType(DOT)

	// 2. قراءة اسم الحالة الفرعية المتميز (مثل Active)
	variantNameTok := p.expectIdent()

	// دمج الاسم الهيكلي المركب المعزول ليصبح "Status.Active" لحماية فاحص الأنواع
	fullVariantName := parentName + "." + variantNameTok.Lexeme

	// 3. استهلاك القوس المتعرج المفتوح '{' للبدء في تجميع الحقول
	p.expectType(OPN_BRACE)
	fields := []FieldInit{}

	// 4. استخدام نفس المنطق الهندسي القياسي الخاص بك لقراءة وتجميع الخصائص
	for p.pos < len(p.tokens) && p.currentToken().Type != CLS_BRACE {
		var value Expression
		line := p.currentToken().Line

		if p.currentToken().Type == IDENT && p.peekToken().Type == COLON {
			nameTok := p.expectIdent()
			p.expectType(COLON)
			value = p.parseExpr()

			fields = append(fields, FieldInit{
				Name:  nameTok.Lexeme,
				Value: value,
				Line:  line,
			})
		} else {
			value = p.parseExpr()
			if value == nil {
				p.advanceToken()
				continue
			}
			fields = append(fields, FieldInit{
				Name:  "",
				Value: value,
				Line:  line,
			})
		}

		if p.currentToken().Type == COMMA {
			p.advanceToken()
		}

		if p.currentToken().Type == EOF {
			break
		}
	}

	p.expectType(CLS_BRACE) // استهلاك قوس الإغلاق '}'

	// 5. العودة بـ StructLiteral نقي ومخصص للشجرة يحمل الاسم المدمج الفريد
	return &StructLiteral{
		Type:   &symbols.Type{Name: fullVariantName},
		Fields: fields,
		Line:   variantNameTok.Line,
	}
}
func (p *Parser) parseSwitch() Statement {
	line := p.currentToken().Line

	p.expectType(SWITCH)

	objExpr := p.parseExpr()

	p.skip()
	p.expectType(OPN_BRACE)
	p.skip()

	cases := []CaseBlock{}
	var elseBlock *FrameBlock = nil

	for p.pos < len(p.tokens) && p.currentToken().Type != CLS_BRACE {
		p.skip()

		if p.currentToken().Type == EOF {
			p.appendErrorf("unexpected EOF inside switch statement block", p.currentToken().Line)
			break
		}

		if p.currentToken().Type == CASE {
			p.expectType(CASE)

			conditions := []Expression{}

			for {
				p.skip()

				condExpr := p.parseExpr()
				conditions = append(conditions, condExpr)

				p.skip()

				if p.currentToken().Type == COMMA {
					p.expectType(COMMA)
					continue
				}

				if p.currentToken().Type == COLON {
					break
				}

				if p.currentToken().Type == EOF {
					break
				}
			}

			p.expectType(COLON)
			p.skip()

			caseStmts := p.parseCaseStatements()

			cases = append(cases, CaseBlock{
				Conditions: conditions,

				Body: &FrameBlock{Stmts: caseStmts, Line: line},
				Line: line,
			})

		} else if p.currentToken().Type == ELSE {
			p.expectType(ELSE)
			p.expectType(COLON)
			p.skip()

			elseStmts := p.parseCaseStatements()

			elseBlock = &FrameBlock{Stmts: elseStmts, Line: line}
			break

		} else {
			p.pos++
		}
	}

	p.expectType(CLS_BRACE)

	return &SwitchStmt{
		Object: objExpr,
		Cases:  cases,
		Else:   elseBlock,
		Line:   line,
	}
}

func (p *Parser) parseCaseStatements() []Statement {
	stmts := []Statement{}

	for p.pos < len(p.tokens) &&
		p.currentToken().Type != CASE &&
		p.currentToken().Type != ELSE &&
		p.currentToken().Type != CLS_BRACE {

		p.skip()

		if p.currentToken().Type == EOF {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}

		p.skip()
	}

	return stmts
}

func (p *Parser) parseMatch() Statement {
	line := p.currentToken().Line
	p.expectType(MATCH)

	var objExpr Expression

	if p.currentToken().Type == IDENT && p.pos+1 < len(p.tokens) &&
		(p.tokens[p.pos+1].Type == OPN_BRACE || p.tokens[p.pos+1].Type == NEW_LINE) {

		tok := p.currentToken()
		p.pos++
		objExpr = &IdentExpr{Name: tok.Lexeme, Line: tok.Line, Type: nil}
	} else {
		objExpr = p.parseExpr()
	}

	p.skipNewlinesAndSpaces()
	p.expectType(OPN_BRACE)
	p.skipNewlinesAndSpaces()

	cases := []CaseBlock{}
	var elseBlock *FrameBlock = nil


	for p.pos < len(p.tokens) && p.currentToken().Type != CLS_BRACE {
		p.skipNewlinesAndSpaces()

		if p.currentToken().Type == EOF {
			p.appendErrorf("unexpected EOF inside match block statement", p.currentToken().Line)
			break
		}

		if p.currentToken().Type == CASE {
			p.expectType(CASE)
			p.skipNewlinesAndSpaces()

			variantNameTok := p.expectIdent()
			conditions := []Expression{
				&IdentExpr{Name: variantNameTok.Lexeme, Line: variantNameTok.Line, Type: nil},
			}

			p.skipNewlinesAndSpaces()
			p.expectType(COLON)
			p.skipNewlinesAndSpaces()

			hasBraces := false
			if p.currentToken().Type == OPN_BRACE {
				p.expectType(OPN_BRACE)
				hasBraces = true
				p.skipNewlinesAndSpaces()
			}

			caseStmts := p.parseCaseStatements()

			if hasBraces {
				p.skipNewlinesAndSpaces()
				p.expectType(CLS_BRACE)
				p.skipNewlinesAndSpaces()
			}

			cases = append(cases, CaseBlock{
				Conditions: conditions,
				Body:       &FrameBlock{Stmts: caseStmts, Line: line},
				Line:       variantNameTok.Line,
			})

		} else if p.currentToken().Type == ELSE {
			p.expectType(ELSE)
			p.skipNewlinesAndSpaces()
			p.expectType(COLON)
			p.skipNewlinesAndSpaces()

			elseStmts := p.parseCaseStatements()
			elseBlock = &FrameBlock{Stmts: elseStmts, Line: line}
			break
		} else {
			p.appendErrorf("syntax error inside match block: unexpected token %s (%q)", 
				p.currentToken().Line, p.currentToken().Type.String(), p.currentToken().Lexeme)
			p.pos++
		}
	}

	p.skipNewlinesAndSpaces()
	p.expectType(CLS_BRACE) 

	return &MatchStmt{
		Object: objExpr,
		Cases:  cases,
		Else:   elseBlock,
		Line:   line,
	}
}
