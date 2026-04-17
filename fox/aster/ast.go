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
	Line int
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
	Line     int
}

type Param struct {
	Name string
	Type Type
	Line int
}

type ReturnSig struct {
	Name string
	Type Type
	Line int
}

type FieldAccessExpr struct {
	Object Expression
	Field  string
	Line   int
}

func (FieldAccessExpr) isExpr() {}

type TypeNode interface {
	isType()
}

type TypeRef struct {
	Name string
}
