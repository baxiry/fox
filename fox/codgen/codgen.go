package codgen

import (
	"fmt"
	"fox/aster"
	"strings"
)

type Codegen struct {
	builder strings.Builder
	unit    *aster.AST
	project *aster.Project
	indent  int // To keep the generated C code clean
}

func NewCodegen(proj *aster.Project) *Codegen {
	var firstUnit *aster.AST
	if len(proj.Packages) > 0 && len(proj.Packages[0].Files) > 0 {
		firstUnit = &aster.AST{
			Decls: proj.Packages[0].Files[0].Decls,
		}
	}

	return &Codegen{
		unit:    firstUnit,
		project: proj,
	}
}

func (cg *Codegen) Generate() string {
	if cg.unit == nil {
		panic("Codegen unit is nil!")
	}

	cg.builder.WriteString("#include <stdio.h>\n#include <stdbool.h>\n#include <stdint.h>\n\n")

	//  Structs
	for _, decl := range cg.unit.Decls {
		if d, ok := decl.(*aster.Struct); ok {
			cg.genStruct(d)
		}
	}

	for _, decl := range cg.unit.Decls {
		switch d := decl.(type) {
		case *aster.VarDeclar:
			cg.genGlobalVar(d)
		case *aster.Func:
			cg.genFunction(d)
		}
	}

	return cg.builder.String()
}

func (cg *Codegen) genGlobalVar(decl *aster.VarDeclar) {
	cType := cg.mapType(decl.Type)
	fmt.Fprintf(&cg.builder, "%s %s;\n", cType, decl.Name)
}

func (cg *Codegen) genFunction(f *aster.Func) {
	// 1. Determine Return Type
	retType := "void"
	// Special case: C's main function MUST return int
	if f.FuncName == "main" {
		retType = "int"
	} else if f.Return != nil {
		retType = cg.mapType(&f.Return.Type)
	}

	// 2. Generate function signature
	fmt.Fprintf(&cg.builder, "%s %s(", retType, f.FuncName)
	for i, p := range f.Params {
		pType := cg.mapType(&p.Type)
		fmt.Fprintf(&cg.builder, "%s %s", pType, p.Name)
		if i < len(f.Params)-1 {
			cg.builder.WriteString(", ")
		}
	}
	cg.builder.WriteString(") {\n")

	cg.indent++
	if f.Body != nil {
		for _, stmt := range f.Body.Stmts {
			cg.genStmt(stmt)
		}
	}

	// 3. Smart return handling
	// Only add "return 0" if it's the main function OR a void function
	// (To prevent duplicate return in functions like 'add')
	if f.FuncName == "main" {
		cg.writeIndent()
		cg.builder.WriteString("return 0;\n")
	} else if f.Return == nil {
		// Implicit return for void functions in C (optional but safe)
		// cg.builder.WriteString("return;\n")
	}

	cg.indent--
	cg.builder.WriteString("}\n\n")
}

func (cg *Codegen) genExpr(expr aster.Expression) {
	switch e := expr.(type) {

	case *aster.Declar:
		if ident, ok := e.Name.(*aster.IdentExpr); ok {
			// Determine the type: Struct name or default int32_t
			if lit, ok := e.Value.(*aster.StructLiteral); ok {
				fmt.Fprintf(&cg.builder, "%s %s = ", lit.Type.Name, ident.Name)
			} else {
				fmt.Fprintf(&cg.builder, "int32_t %s = ", ident.Name)
			}
			cg.genExpr(e.Value)
			// Semicolon is written here for Declarations
			cg.builder.WriteString(";\n")
		}

	case *aster.StructLiteral:
		// C99 compound literal: (TypeName){.field = value}
		fmt.Fprintf(&cg.builder, "(%s){", e.Type.Name)
		for i, field := range e.Fields {
			fmt.Fprintf(&cg.builder, ".%s = ", field.Name)
			cg.genExpr(field.Value)
			if i < len(e.Fields)-1 {
				cg.builder.WriteString(", ")
			}
		}
		cg.builder.WriteString("}")

	case *aster.FieldAccessExpr:
		// Direct member access: object.field
		cg.genExpr(e.Object)
		fmt.Fprintf(&cg.builder, ".%s", e.Field)

	case *aster.IntExpr:
		fmt.Fprintf(&cg.builder, "%d", e.Value)

	case *aster.StringExpr:
		fmt.Fprintf(&cg.builder, "\"%s\"", e.Literal)

	case *aster.IdentExpr:
		cg.builder.WriteString(e.Name)

	case *aster.BinaryExpr:
		cg.builder.WriteString("(")
		cg.genExpr(e.Left)
		fmt.Fprintf(&cg.builder, " %s ", e.Op)
		cg.genExpr(e.Right)
		cg.builder.WriteString(")")

	case *aster.CallExpr:
		cg.genCall(e)

	case *aster.IndexExpr:
		cg.genExpr(e.Target)
		cg.builder.WriteString("[")
		cg.genExpr(e.Index)
		cg.builder.WriteString("]")
	}
}

func (cg *Codegen) genCall(e *aster.CallExpr) {
	ident, _ := e.Callee.(*aster.IdentExpr)
	cg.builder.WriteString(ident.Name + "(")
	for i, arg := range e.Args {
		cg.genExpr(arg)
		if i < len(e.Args)-1 {
			cg.builder.WriteString(", ")
		}
	}
	cg.builder.WriteString(")")
}

// mapType converts Fox types to C standard types
func (cg *Codegen) mapType(foxType *aster.Type) string {
	// Safety check to prevent Panic
	if foxType == nil {
		return "int32_t"
	}

	var cType string
	switch foxType.Name {
	case "int":
		cType = "int32_t"
	case "string":
		cType = "char*"
	case "bool":
		cType = "bool"
	default:
		cType = foxType.Name
	}

	for i := 0; i < foxType.PtrDepth; i++ {
		cType += "*"
	}

	return cType
}

func (cg *Codegen) writeIndent() {
	for i := 0; i < cg.indent; i++ {
		cg.builder.WriteString("    ")
	}
}

func (cg *Codegen) genStmt(stmt aster.Statement) {
	switch s := stmt.(type) {

	case *aster.Declar:
		cg.writeIndent()
		cg.genExpr(s)

	case *aster.VarDeclar:
		cg.writeIndent()
		cType := cg.mapType(s.Type)
		if s.Type != nil && s.Type.IsArray {
			fmt.Fprintf(&cg.builder, "%s %s[%d];\n", cType, s.Name, s.Type.Size)
		} else {
			if s.Value != nil {
				fmt.Fprintf(&cg.builder, "%s %s = ", cType, s.Name)
				cg.genExpr(s.Value)
				cg.builder.WriteString(";\n")
			} else {
				fmt.Fprintf(&cg.builder, "%s %s;\n", cType, s.Name)
			}
		}
	case *aster.Assign:
		cg.writeIndent()
		cg.genExpr(s.Target)
		cg.builder.WriteString(" = ")
		cg.genExpr(s.Value)
		cg.builder.WriteString(";\n")

	case *aster.ExprStmt:
		// We must ensure the expression (like Declar) is generated
		cg.writeIndent()
		cg.genExpr(s.Expr)

		// Add semicolon only if the expression doesn't handle its own terminator
		if _, isDecl := s.Expr.(*aster.Declar); !isDecl {
			cg.builder.WriteString(";\n")
		}

	case *aster.ReturnStmt:
		cg.writeIndent()
		cg.builder.WriteString("return ")
		if s.Result != nil {
			cg.genExpr(s.Result)
		}
		cg.builder.WriteString(";\n")

	case *aster.ForStmt:
		cg.writeIndent()
		cg.builder.WriteString("for (")
		// Back to your original working logic for ForStmt
		if s.Init != nil {
			if assign, ok := s.Init.(*aster.Assign); ok {
				cg.genExpr(assign.Target)
				cg.builder.WriteString(" = ")
				cg.genExpr(assign.Value)
			}
		}
		cg.builder.WriteString("; ")
		if s.Cond != nil {
			cg.genExpr(s.Cond)
		}
		cg.builder.WriteString("; ")
		if s.Post != nil {
			if assign, ok := s.Post.(*aster.Assign); ok {
				cg.genExpr(assign.Target)
				cg.builder.WriteString(" = ")
				cg.genExpr(assign.Value)
			}
		}
		cg.builder.WriteString(") ")
		cg.genBlock(s.Body)

	}
}

func (cg *Codegen) genBlock(block *aster.FrameBlock) {
	cg.builder.WriteString("{\n")
	cg.indent++

	for _, stmt := range block.Stmts {
		cg.writeIndent()
		cg.genStmt(stmt)
	}

	cg.indent-- // Decrease indentation before closing
	cg.writeIndent()
	cg.builder.WriteString("}\n")
}

// Generating the structure
func (cg *Codegen) genStruct(s *aster.Struct) {
	fmt.Fprintf(&cg.builder, "typedef struct {\n")
	for _, field := range s.Fields {
		// Make sure to use mapType to convert string to char* or similar
		cType := cg.mapType(&field.Type)
		fmt.Fprintf(&cg.builder, "    %s %s;\n", cType, field.Name)
	}
	fmt.Fprintf(&cg.builder, "} %s;\n\n", s.Name)
}
