package main

type AST struct {
	PackageName string
	Imports     ImportNode
	Structs     []StructNode
	Funcs       []FuncNode
	//ExportVars  []string
	//Vars        []string
}

type ImportNode []string

type StructNode struct {
	Name   string
	Fields []FieldNode
}

type FieldNode struct {
	Name string
	Type string
}

type ReturnNode struct {
	Value ExpressionNode
}

type FuncNode struct {
	Name    string
	Params  []ParamNode
	Returns []ReturnNode
	Body    []StatementNode
}

type ParamNode struct {
	Name string
	Type string
}
