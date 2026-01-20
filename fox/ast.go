package main

type AST struct {
	PackageName string
	Imports     []string
	Structs     []StructDecl
	Funcs       []FuncDecl
}

type StructDecl struct {
	Name   string
	Fields []FieldDecl
}

type FieldDecl struct {
	Name string
	Type string
}

type FuncDecl struct {
	Name    string
	Params  []ParamDecl
	Returns []ReturnSig
	Body    []Statement
	Vars    []string
}

type ParamDecl struct {
	Name string
	Type string
}

type ReturnSig struct {
	Name string
	Type string
}

//[]string
