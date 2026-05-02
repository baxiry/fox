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

// Generate is the main entry point
func (cg *Codegen) Generate() string {
	if cg.unit == nil {
		panic("Codegen unit is nil!")
	}

	// 1. Add Standard C Headers
	cg.builder.WriteString("#include <stdio.h>\n")
	cg.builder.WriteString("#include <stdint.h>\n\n")

	// 2. Generate Declarations and Bodies
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
	cType := cg.mapType(decl.Type.Name)
	cg.builder.WriteString(fmt.Sprintf("%s %s;\n", cType, decl.Name))
}

func (cg *Codegen) genFunction(f *aster.Func) {
	// For simplicity, we assume int return for now
	cg.builder.WriteString(fmt.Sprintf("int %s() {\n", f.FuncName))
	cg.indent++

	if f.Body != nil {
		for _, stmt := range f.Body.Stmts {
			cg.writeIndent()
			cg.genStmt(stmt)
			cg.builder.WriteString(";\n")
		}
	}

	cg.writeIndent()
	cg.builder.WriteString("return 0;\n")
	cg.indent--
	cg.builder.WriteString("}\n\n")
}

func (cg *Codegen) genStmt(stmt aster.Statement) {
	switch s := stmt.(type) {
	case *aster.Assign:
		if len(s.Targets) > 0 {
			target, _ := s.Targets[0].(*aster.IdentExpr)
			cg.builder.WriteString(target.Name + " = ")
			cg.genExpr(s.Values[0])
		}
	case *aster.ExprStmt:
		cg.genExpr(s.Expr)
	}
}

func (cg *Codegen) genExpr(expr aster.Expression) {
	switch e := expr.(type) {
	case *aster.IntExpr:
		cg.builder.WriteString(fmt.Sprintf("%d", e.Value))

	case *aster.StringExpr:
		// Escape quotes for C string literal
		cg.builder.WriteString("\"")
		cg.builder.WriteString(e.Literal)
		cg.builder.WriteString("\"")

	case *aster.IdentExpr:
		cg.builder.WriteString(e.Name)

	case *aster.BinaryExpr:
		cg.builder.WriteString("(")
		cg.genExpr(e.Left)
		cg.builder.WriteString(" " + e.Op + " ")
		cg.genExpr(e.Right)
		cg.builder.WriteString(")")

	case *aster.CallExpr:
		cg.genCall(e)
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
func (cg *Codegen) mapType(foxType string) string {
	switch foxType {
	case "int":
		return "int32_t"
	case "string":
		return "char*"
	default:
		return "int32_t"
	}
}

func (cg *Codegen) writeIndent() {
	for i := 0; i < cg.indent; i++ {
		cg.builder.WriteString("    ")
	}
}
