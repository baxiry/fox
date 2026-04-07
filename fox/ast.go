package main

type AST struct {
	PackageName string
	Imports     []string
	Structs     []StructDecl
	Funcs       []FuncDecl
}

type Node interface{}

type Type struct {
	Name     string
	PtrDepth int
}

type StructDecl struct {
	Name   string
	Fields []FieldDecl
}
type FieldDecl struct {
	Name string
	Type string
}

type FrameBlock struct {
	Stmts []Statement
}

type FuncDecl struct {
	Name    string
	Params  []ParamDecl
	Returns []ReturnSig
	Body    *FrameBlock // BlockStmt
}

type ParamDecl struct {
	Name string
	Type Type
}

type ReturnSig struct {
	Name string
	Type Type
}

type FieldAccessExpr struct {
	Object Expression
	Field  string
}

func (FieldAccessExpr) isExpr() {}

type TypeNode interface {
	isType()
}

type SimpleType struct {
	Name string
}

func (SimpleType) isType() {}

type PointerType struct {
	Base TypeNode
}

func (PointerType) isType() {}
