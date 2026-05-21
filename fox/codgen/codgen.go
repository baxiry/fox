package codgen

import (
	"fmt"
	"fox/aster"
	"fox/symbols"
	"strings"
)

type Codegen struct {
	symbolTable *symbols.SymbolTable
	builder     strings.Builder
	unit        *aster.AST
	project     *aster.Project
	indent      int
}

func NewCodegen(proj *aster.Project) *Codegen {
	var firstUnit *aster.AST
	if len(proj.Packages) > 0 && len(proj.Packages[0].Files) > 0 {
		firstUnit = &aster.AST{
			Decls: proj.Packages[0].Files[0].Decls,
		}
	}

	activeTable := proj.SymbolTable
	if activeTable == nil {
		activeTable = symbols.NewSymbolTable(nil)
	}

	return &Codegen{
		unit:        firstUnit,
		project:     proj,
		symbolTable: activeTable,
	}
}

func (cg *Codegen) Generate() string {
	if cg.unit == nil {
		panic("Codegen unit is nil!")
	}

	// 1. Emit the absolute foundational headers at the top of the C translation unit
	cg.builder.WriteString("#include <stdio.h>\n#include <stdbool.h>\n#include <stdint.h>\n\n")

	// 2. Emit the forward declaration of your garbage collector to prevent compilation warnings
	// cg.builder.WriteString("void fgc_init(void);\n\n")

	cg.builder.WriteString("#include \"foxgc/fgc.h\"\n\n")

	// 3. Generate all Structural definitions first (File Scope Layouts)
	for _, decl := range cg.unit.Decls {
		if d, ok := decl.(*aster.Struct); ok {
			cg.genStruct(d)
		}
	}

	// 4. Generate Global Variables and Functions sequentially
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
		retType = cg.mapType(f.Return.Type)
	}

	// 2. Generate function signature
	fmt.Fprintf(&cg.builder, "%s %s(", retType, f.FuncName)
	for i, p := range f.Params {
		pType := cg.mapType(p.Type)
		fmt.Fprintf(&cg.builder, "%s %s", pType, p.Name)
		if i < len(f.Params)-1 {
			cg.builder.WriteString(", ")
		}
	}
	cg.builder.WriteString(") {\n")

	cg.indent++

	// If this is the main application entry point, inject foxGC initialization safely
	if f.FuncName == "main" {
		cg.writeIndent()
		// Defining a directory variable to store the stack top address at startup
		fmt.Fprintf(&cg.builder, "int32_t stack_top_anchor;\n")
		cg.writeIndent()
		// Passing the variable's address to the Runtime initialization function
		fmt.Fprintf(&cg.builder, "fgc_init(&stack_top_anchor);\n")
	}
	if f.Body != nil {
		for _, stmt := range f.Body.Stmts {
			cg.genStmt(stmt)
		}
	}

	// 3. Smart return handling
	// Only add "return 0" if it's the main function OR a void function
	if f.FuncName == "main" {
		cg.writeIndent()
		cg.builder.WriteString("return 0;\n")
	}

	cg.indent--
	cg.builder.WriteString("}\n\n")
}

func (cg *Codegen) genExpr(expr aster.Expression) {
	switch e := expr.(type) {

	case *aster.UnaryExpr:
		// Handle unary operators like address-of (&)
		// Note: C uses the same symbols as Fox for these operators
		fmt.Fprintf(&cg.builder, "%s", e.Op) // Prints '&'
		cg.genExpr(e.Expr)                   // Prints the target (e.g., 'user')

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
		// 1. Generate the object (e.g., ptr)
		cg.genExpr(e.Object)

		// 2. Decide between '.' and '->' using the verified PtrDepth
		separator := "."
		if ident, ok := e.Object.(*aster.IdentExpr); ok && ident.Type != nil {
			if ident.Type.PtrDepth > 0 {
				separator = "->"
			}
		}

		// 3. Print the separator and the field name
		fmt.Fprintf(&cg.builder, "%s%s", separator, e.Field)

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
func (cg *Codegen) mapType(foxType *symbols.Type) string {
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
				// Check if we are allocating a Struct Literal that belongs on the Heap
				if structLit, ok := s.Value.(*aster.StructLiteral); ok && s.Type.PtrDepth > 0 {
					// Route allocation to the safe 8-byte padded heap allocator
					cg.genHeapStructLiteral(structLit, s.Name)
				} else {
					// Traditional Stack/Value allocation layout
					fmt.Fprintf(&cg.builder, "%s %s = ", cType, s.Name)
					cg.genExpr(s.Value)
					cg.builder.WriteString(";\n")
				}
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
		cg.writeIndent()
		cg.genExpr(s.Expr)
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

		/* 🔍 Dynamic Type Switch evaluates both mutations and inline structural shorthand declarations */
		if s.Init != nil {
			switch initStmt := s.Init.(type) {
			case *aster.Assign:
				/* Standard Path: Emit regular assignment configuration for existing bounds counters */
				cg.genExpr(initStmt.Target)
				cg.builder.WriteString(" = ")
				cg.genExpr(initStmt.Value)
			case *aster.Declar:
				/* Shorthand Path: Emit clean localized dynamic type bindings directly into the loop header */
				cg.builder.WriteString("int32_t ")
				cg.genExpr(initStmt.Name)
				cg.builder.WriteString(" = ")
				cg.genExpr(initStmt.Value)
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

func (cg *Codegen) genHeapStructLiteral(lit *aster.StructLiteral, targetVarName string) {
	structName := lit.Type.Name

	classIdx := cg.calculateClassIndex(structName)

	// 1. Generate the raw pointer extraction and embed the 8-byte offset calculation directly in C
	// We cast to (char*) first to perform clean single-byte arithmetic pointer increments
	fmt.Fprintf(&cg.builder, "    %s* %s = (%s*)((char*)fgc_alloc(%d) + 8);\n",
		structName, targetVarName, structName, classIdx)

	// 2. Initialize the struct member variables values lineally using direct genExpr recursion
	for _, providedField := range lit.Fields {
		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "%s->%s = ", targetVarName, providedField.Name)
		cg.genExpr(providedField.Value) // Directly emits the C-literal expression into cg.builder
		cg.builder.WriteString(";\n")
	}
}

// Generating the structure
func (cg *Codegen) genStruct(s *aster.Struct) {
	fmt.Fprintf(&cg.builder, "typedef struct {\n")
	for _, field := range s.Fields {
		// Make sure to use mapType to convert string to char* or similar
		cType := cg.mapType(field.Type)

		// Fix: Handle static arrays inside struct fields to ensure correct physical memory stride
		if field.Type != nil && field.Type.IsArray {
			fmt.Fprintf(&cg.builder, "    %s %s[%d];\n", cType, field.Name, field.Type.Size)
		} else {
			fmt.Fprintf(&cg.builder, "    %s %s;\n", cType, field.Name)
		}
	}
	fmt.Fprintf(&cg.builder, "} %s;\n\n", s.Name)
}

// isPointerType checks if the object being accessed is a pointer
func (cg *Codegen) isPointerType(expr aster.Expression) bool {
	switch e := expr.(type) {
	case *aster.IdentExpr:
		// Search in local or global declarations for this identifier's type
		// For now, we can look into the current function scope if available
		// Or assume the parser has already marked the PtrDepth
		return e.Type != nil && e.Type.PtrDepth > 0

	case *aster.UnaryExpr:
		// Address-of (&) always results in a pointer
		return e.Op == "&"
	}
	return false
}

func (cg *Codegen) calculateClassIndex(sName string) int {
	// Querying the unified truth source via the domain tree
	structSym, exists := cg.symbolTable.Resolve(sName)
	if !exists || structSym == nil {
		return 0
	}

	totalSize := 0
	for _, field := range structSym.Fields {
		fieldSize := 0

		// Calculating physical volumes based on basic types
		if field.Type.PtrDepth > 0 || field.Type.Name == "string" {
			fieldSize = 8
		} else if field.Type.Name == "int" {
			fieldSize = 4
		} else if field.Type.Name == "bool" {
			fieldSize = 1
		} else {
			fieldSize = 8
		}
		// Multiplying the physical space step if the field is a fixed matrix
		if field.Type.IsArray && field.Type.Size > 0 {
			fieldSize = fieldSize * field.Type.Size
		}

		totalSize += fieldSize
	}

	// Physical alignment of 8 bytes to prevent gaps within the cache
	if totalSize%8 != 0 {
		totalSize = ((totalSize / 8) + 1) * 8
	}

	totalNeeded := totalSize + 8

	// Matching the final size with the fixed foxGC pools
	configurations := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}
	for idx, maxCapacity := range configurations {
		if totalNeeded <= maxCapacity {
			return idx
		}
	}

	// Passing large objects to the large pool slot POOL_LARGE
	return 8
}
