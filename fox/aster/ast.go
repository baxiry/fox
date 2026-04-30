package aster

// Project is the top-level root for a multi-package compilation
type Project struct {
	Packages []*Package
}

// Package now contains files and its name
type Package struct {
	Name  string
	Files []*File // New: Supporting multiple files per package
}

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

func (p *Package) GetLine() int { return 1 }

// Expr represents expressions that return a value (e.g., numbers, calls, binary ops).
type Expression interface {
	Node
	GetLine() int
	isExpr()
}

// VarDeclar represents a variable declaration.
type VarDeclar struct {
	Name  string
	Type  *Type
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

type Type struct {
	Name     string
	PtrDepth int
	Line     int
}

type Struct struct {
	Name   string
	Fields []Field
	Line   int
}

func (s *Struct) GetLine() int { return s.Line }

func (s *Struct) isDecl() {}

type Field struct {
	Name string
	Type string
	Line int
}

type FrameBlock struct {
	Stmts []Statement
	Line  int
}

func (FrameBlock) isStat() {}

type Func struct {
	FuncName string
	Params   []Param
	Returns  []ReturnSig
	Body     *FrameBlock // BlockStmt
	Line     int
}

func (f *Func) GetLine() int {
	return f.Line
}

func (f *Func) isDecl() {}

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

// Decl represents top-level declarations like functions, structs, and global variables.
type Decl interface {
	Node
	isDecl() // Marker method to ensure type safety
}

func (t *Type) getLine() int { return t.Line }

// For Declar struct
func (s *Declar) GetLine() int { return s.Line }
func (s *Declar) isStmt()      {}

// For Assign struct
func (s *Assign) GetLine() int { return s.Line }
func (s *Assign) isStmt()      {}

// For ExprStmt struct
func (s *ExprStmt) GetLine() int { return s.Line }
func (s *ExprStmt) isStmt()      {}
