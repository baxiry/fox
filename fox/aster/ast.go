package aster

type AST struct {
	PackageName string
	Imports     []string
	Structs     []Struct
	Funcs       []Func
	Vars        []VarDeclar
}

type Type struct {
	Name     string
	PtrDepth int
}

type Struct struct {
	Name   string
	Fields []Field
}

type Field struct {
	Name string
	Type string
}

type FrameBlock struct {
	Stmts []Statement
}

func (FrameBlock) isStat() {}

type Func struct {
	FuncName string
	Params   []Param
	Returns  []ReturnSig
	Body     *FrameBlock // BlockStmt
}

type Param struct {
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

type TypeRef struct {
	Name string
}
