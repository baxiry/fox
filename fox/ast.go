package main

type AST struct {
	PackageName string
	Imports     []string
	Structs     []StructNode
	Funcs       []FuncNode
	//eVars     []string
	//Vars      []string
	//eConsts   []string
	//Consts    []string
}

type StructNode struct {
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	Name string
	Type string
}

type ReturnNode struct {
	Values []ExpressionNode
}

type RetSignsNode []string

type FuncNode struct {
	Name    string
	Params  []ParamNode
	Returns []ReturnNode
	Body    []StatementNode
	Vars    []string
}

type ParamNode struct {
	Name string
	Type string
}
