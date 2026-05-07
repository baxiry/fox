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
	cg.builder.WriteString("#include <stdbool.h>\n")
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
		// Now 's.Name' is a single IdentExpr and 's.Value' is a single Expression
		if ident, ok := s.Name.(*aster.IdentExpr); ok && ident.Name != "_" {
			cg.writeIndent()
			// Assuming int32_t for now, or use mapType if type info is available
			fmt.Fprintf(&cg.builder, "int32_t %s = ", ident.Name)
			cg.genExpr(s.Value)
			cg.builder.WriteString(";\n")
		}

	case *aster.VarDeclar:
		cg.writeIndent()

		// Get C type from the decorated AST node
		cType := cg.mapType(s.Type)

		if s.Type != nil && s.Type.IsArray {
			fmt.Fprintf(&cg.builder, "%s %s;\n", cType, s.Name)
		} else {
			if s.Value != nil {
				// Handle initialization: int32_t x = add(4, 3);
				fmt.Fprintf(&cg.builder, "%s %s = ", cType, s.Name)
				cg.genExpr(s.Value)
				cg.builder.WriteString(";\n")
			} else {
				// Default declaration
				fmt.Fprintf(&cg.builder, "%s %s;\n", cType, s.Name)
			}
		}

	case *aster.Assign:
		// Handling single target and single value
		if ident, ok := s.Target.(*aster.IdentExpr); ok && ident.Name == "_" {
			// If it's a call, just execute it: swap(x, y);
			cg.genExpr(s.Value)
			cg.builder.WriteString(";\n")
		} else {
			cg.writeIndent()
			cg.genExpr(s.Target)
			cg.builder.WriteString(" = ")
			cg.genExpr(s.Value)
			cg.builder.WriteString(";\n")
		}

	case *aster.ExprStmt:
		cg.writeIndent()
		cg.genExpr(s.Expr)
		cg.builder.WriteString(";\n")

	case *aster.IfStmt:
		cg.writeIndent()
		cg.builder.WriteString("if (")
		cg.genExpr(s.Cond)
		cg.builder.WriteString(")")

		// Generate the 'then' block
		cg.genBlock(s.Then) // Assuming genBlock handles '{' and '}' with indentation

		if s.Else != nil {
			// Fix: Remove the newline before else or ensure it aligns with 'if'
			cg.writeIndent()
			cg.builder.WriteString("else")

			if elseBlock, ok := s.Else.(*aster.FrameBlock); ok {
				cg.genBlock(elseBlock)
			}
		}

	case *aster.ReturnStmt:
		cg.writeIndent()
		cg.builder.WriteString("return ")
		if s.Result != nil {
			cg.genExpr(s.Result)
		} else {
			cg.builder.WriteString("0")
		}
		cg.builder.WriteString(";\n")

	case *aster.ForStmt:
		cg.writeIndent()
		cg.builder.WriteString("for (")

		// 1. Initializer (Simplified for single assignment)
		if s.Init != nil {
			if assign, ok := s.Init.(*aster.Assign); ok {
				cg.genExpr(assign.Target)
				cg.builder.WriteString(" = ")
				cg.genExpr(assign.Value)
			}
		}
		cg.builder.WriteString("; ")

		// 2. Condition
		if s.Cond != nil {
			cg.genExpr(s.Cond)
		}
		cg.builder.WriteString("; ")

		// 3. Post-iteration
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
