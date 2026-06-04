package aster

import "fox/symbols"

// Project is the top-level root for a multi-package compilation
type Project struct {
	Packages []*Package

	SymbolTable *symbols.SymbolTable
}

// Package now contains files and its name
type Package struct {
	Name  string
	Files []*File // New: Supporting multiple files per package
}

func (p *Package) GetLine() int { return 1 }

// File represents a single .fox source file
type File struct {
	Path  string // To help with debugging as you requested
	Decls []Decl
}

type AST struct {
	Package Package
	Imports []string
	Structs []*Struct
	//	Funcs   []*Func
	//	Vars    []*VarDeclar
	Decls []Decl
}

// Expr represents expressions that return a value (e.g., numbers, calls, binary ops).
type Expression interface {
	Node
	GetLine() int
	isExpr()
}

type IndexExpr struct {
	Target Expression // The array variable (e.g., list)
	Index  Expression // The index expression (e.g., i + 1)
	Line   int
}

func (*IndexExpr) isExpr()        {}
func (e *IndexExpr) GetLine() int { return e.Line }

// VarDeclar represents a variable declaration.
type VarDeclar struct {
	Name  string
	Type  *symbols.Type
	Value Expression
	Line  int
}

func (v *VarDeclar) GetLine() int { return v.Line }
func (v *VarDeclar) isDecl()      {}

// Node is the base interface for all elements in the AST.
type Node interface {
	GetLine() int // Returns the source code line number
}

// Stmt represents executable statements (e.g., assignments, if-statements, returns).
type Statement interface {
	//GetLine() int
	Node
	isStmt()
}

type Struct struct {
	Name   string
	Fields []Field
	Line   int
}

func (s *Struct) isDecl()      {}
func (s *Struct) GetLine() int { return s.Line }

type Field struct {
	Name string
	Type *symbols.Type
	Line int
}

type FrameBlock struct {
	Stmts []Statement
	Line  int
}

func (*FrameBlock) isStat() {}

type Func struct {
	FuncName string
	Params   []Param
	Return   *ReturnSig
	Body     *FrameBlock // BlockStmt
	Line     int
}

func (f *Func) GetLine() int { return f.Line }
func (f *Func) isDecl()      {}

type Param struct {
	Name string
	Type *symbols.Type
	Line int
}

type ReturnSig struct {
	IsErrorUnion bool
	Name         string
	Type         *symbols.Type
	Line         int
}

type FieldAccessExpr struct {
	Object Expression
	Field  string
	Line   int
}

// AliasDecl represents statements like: alias Name = string
type AliasDecl struct {
	Name string        // Name of the new alias identifier (e.g., "Name")
	Type *symbols.Type // Underlying base data type pointer (e.g., "string")
	Line int           // Line number for compilation error reporting
}

// isDecl implements the Decl interface cleanly at file scope layout level
func (a *AliasDecl) isDecl()      {}
func (a *AliasDecl) isStmt()      {}
func (a *AliasDecl) GetLine() int { return a.Line }

func (f *FieldAccessExpr) GetLine() int { return f.Line }
func (*FieldAccessExpr) isExpr()        {}

type TypeNode interface {
	isType()
}

type TypeRef struct {
	Name string
}

// Decl represents top-level declarations like functions, structs, and global variables.
type Decl interface {
	Node
	isDecl() // Marker method to ensure type safety
}

// For Declar struct
func (s *Declar) GetLine() int { return s.Line }
func (s *Declar) isStmt()      {}

// For Assign struct
func (s *Assign) GetLine() int { return s.Line }
func (s *Assign) isStmt()      {}

// For ExprStmt struct
func (s *ExprStmt) GetLine() int { return s.Line }
func (s *ExprStmt) isStmt()      {}
