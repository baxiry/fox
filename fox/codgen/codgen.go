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
	cType := cg.mapType(decl.Type)
	fmt.Fprintf(&cg.builder, "%s %s;\n", cType, decl.Name)
}

func (cg *Codegen) genFunction(f *aster.Func) {
	// For simplicity, we assume int return for now
	fmt.Fprintf(&cg.builder, "int %s() {\n", f.FuncName)
	cg.indent++

	if f.Body != nil {
		for _, stmt := range f.Body.Stmts {
			cg.writeIndent()
			cg.genStmt(stmt)
			//cg.builder.WriteString(";/*ok2*/ \n")
		}
	}

	cg.writeIndent()
	cg.builder.WriteString("return 0;\n")
	cg.indent--
	cg.builder.WriteString("}\n\n")
}

func (cg *Codegen) genExpr(expr aster.Expression) {
	switch e := expr.(type) {
	case *aster.IntExpr:
		//cg.builder.WriteString(fmt.Sprintf())
		fmt.Fprintf(&cg.builder, "%d", e.Value)

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

	case *aster.IndexExpr:
		// the array name
		cg.genExpr(e.Target)

		cg.builder.WriteString("[")

		// like [a + 1]
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
	// add '*' depend of depth deref
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

	case *aster.VarDeclar:
		cType := cg.mapType(s.Type) // Use the centralized type mapper

		if s.Type != nil && s.Type.IsArray {
			// Simple C array implementation
			fmt.Fprintf(&cg.builder, "%s %s[100];\n", cType, s.Name)
		} else {
			fmt.Fprintf(&cg.builder, "%s %s;\n", cType, s.Name)
		}

	case *aster.Assign:
		if len(s.Targets) > 0 && len(s.Values) > 0 {
			cg.genExpr(s.Targets[0])

			cg.builder.WriteString(" = ")

			cg.genExpr(s.Values[0])
			cg.builder.WriteString(";\n")
		}

	case *aster.ExprStmt:
		cg.genExpr(s.Expr)
		// Ensure no extra newline was added inside genExpr
		cg.builder.WriteString(";\n")

	case *aster.IfStmt:
		cg.builder.WriteString("if ")
		cg.genExpr(s.Cond)
		cg.genBlock(s.Then) // Block ends with its own newline

		if s.Else != nil {
			// Remove any potential newline before 'else'
			cg.builder.WriteString(" else ")
			if elseBlock, ok := s.Else.(*aster.FrameBlock); ok {
				cg.genBlock(elseBlock)
			}
		}
		//cg.builder.WriteString("/* ok */\n")

	case *aster.ForStmt:
		cg.builder.WriteString("for (")

		// 1. Initializer
		if s.Init != nil {
			if assign, ok := s.Init.(*aster.Assign); ok {
				// Guard clause: if something is wrong, we skip this part
				if len(assign.Targets) == 0 || len(assign.Values) == 0 {
					goto skipInit
				}

				target, okT := assign.Targets[0].(*aster.IdentExpr)
				if okT {
					cg.builder.WriteString(target.Name + " = ")
					cg.genExpr(assign.Values[0])
				}
			}
		}
	skipInit:
		cg.builder.WriteString("; ")

		// 2. Condition
		if s.Cond != nil {
			cg.genExpr(s.Cond)
		}
		cg.builder.WriteString("; ")

		// 3. Post-iteration
		if s.Post != nil {
			if assign, ok := s.Post.(*aster.Assign); ok {
				if len(assign.Targets) == 0 || len(assign.Values) == 0 {
					goto skipPost
				}

				target, okT := assign.Targets[0].(*aster.IdentExpr)
				if okT {
					cg.builder.WriteString(target.Name + " = ")
					cg.genExpr(assign.Values[0])
				}
			}
		}
	skipPost:
		cg.builder.WriteString(") ")

		// 4. Loop Body
		cg.genBlock(s.Body)
		cg.builder.WriteString("\n")

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
