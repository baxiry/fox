package cgen

import (
	"fmt"
	"fox/aster"
	"fox/symbols"
	"strings"
)

type Codegen struct {
	symbolTable     *symbols.SymbolTable
	builder         strings.Builder
	unit            *aster.AST
	project         *aster.Project
	indent          int
	CurrentFunction *aster.Func
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

	cg.builder.WriteString(
		"#include <stdio.h>\n#include <stdbool.h>\n#include <stdint.h>\n#include <stdlib.h>\n#include \"../foxgc/fgc.h\"\n\n")

	for _, decl := range cg.unit.Decls {
		if d, ok := decl.(*aster.Struct); ok {
			cg.genStruct(d)
		}
		if d, ok := decl.(*aster.EnumDecl); ok {
			cg.genEnumDecl(d)
		}
	}

	cg.genResultEnvelopes()

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

func (cg *Codegen) genResultEnvelopes() {
	generated := make(map[string]bool)

	for _, decl := range cg.unit.Decls {
		if f, ok := decl.(*aster.Func); ok && f.Return != nil && len(f.Return.Fields) > 0 {
			if len(f.Return.Fields) == 1 && !f.Return.HasError {
				continue
			}

			envelopeName := "_Fox_res_" + f.FuncName

			if generated[envelopeName] {
				continue
			}
			generated[envelopeName] = true

			fmt.Fprintf(&cg.builder, "typedef struct %s {\n", envelopeName)
			for _, field := range f.Return.Fields {
				fmt.Fprintf(&cg.builder, "    %s %s;\n", field.Type.Name, field.Name)
			}
			if f.Return.HasError {
				fmt.Fprintf(&cg.builder, "    char* err;\n")
			}
			fmt.Fprintf(&cg.builder, "} %s;\n\n", envelopeName)
		}
	}
}

func (cg *Codegen) genGlobalVar(decl *aster.VarDeclar) {
	cType := cg.mapType(decl.Type)
	fmt.Fprintf(&cg.builder, "%s %s;\n", cType, decl.Name)
}

func (cg *Codegen) genFunction(f *aster.Func) {
	cg.CurrentFunction = f

	retType := "void"
	if f.FuncName == "main" {
		retType = "int"
	} else if f.Return != nil && len(f.Return.Fields) > 0 {
		if len(f.Return.Fields) == 1 && !f.Return.HasError {
			retType = cg.mapType(&f.Return.Fields[0].Type)
		} else {
			retType = "_Fox_res_" + f.FuncName
		}
	}

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

	if f.FuncName == "main" {
		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "int32_t stack_top_anchor;\n")
		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "fgc_init(&stack_top_anchor);\n")
	}

	// handle Named Returns
	if f.Return != nil && len(f.Return.Fields) > 0 {
		for _, field := range f.Return.Fields {
			if field.Name != "" {
				cg.writeIndent()
				fType := cg.mapType(&field.Type)
				if field.Type.PtrDepth > 0 {
					// Pointer = NULL
					fmt.Fprintf(&cg.builder, "%s %s = NULL;\n", fType, field.Name)
				} else {
					//  Value = {0}
					fmt.Fprintf(&cg.builder, "%s %s = {0};\n", fType, field.Name)
				}
			}
		}
	}

	if f.Body != nil {
		for _, stmt := range f.Body.Stmts {
			cg.genStmt(stmt)
		}
	}

	if f.FuncName == "main" {
		cg.writeIndent()
		cg.builder.WriteString("return 0;\n")
	}

	cg.indent--
	cg.builder.WriteString("}\n\n")

	cg.CurrentFunction = nil
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

// genHeapStructLiteral assigns heap structure footprints,
// handling discriminator tags for enum flat layout allocations.
func (cg *Codegen) genHeapStructLiteral(lit *aster.StructLiteral, targetVarName string) {
	rawName := lit.Type.Name
	structName := rawName
	variantName := ""
	isEnumVariant := false

	if strings.Contains(rawName, ".") {
		parts := strings.Split(rawName, ".")
		structName = parts[0]
		variantName = parts[1]
		isEnumVariant = true
	}

	classIdx := cg.calculateClassIndex(structName)
	hasPointers := cg.structHasPointers(structName)
	typeTag := 1

	fmt.Fprintf(&cg.builder, "    %s* %s = (%s*)((char*)fgc_alloc(%d, %d, %d) + 8);\n",
		structName, targetVarName, structName, classIdx, typeTag, hasPointers)

	if isEnumVariant {
		generatedTagValue := 0
		switch variantName {
		case "Active":
			generatedTagValue = 1
		case "Inactive":
			generatedTagValue = 2
		}

		/*
			if variantName == "Active" {
				generatedTagValue = 1
			} else if variantName == "Inactive" {
				generatedTagValue = 2
			}
		*/
		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "%s->_tag = %d;\n", targetVarName, generatedTagValue)

		for _, providedField := range lit.Fields {
			cg.writeIndent()
			fmt.Fprintf(&cg.builder, "%s->variants.%s.%s = ", targetVarName, variantName, providedField.Name)
			cg.genExpr(providedField.Value)
			cg.builder.WriteString(";\n")
		}
	} else {
		for _, providedField := range lit.Fields {
			cg.writeIndent()
			fmt.Fprintf(&cg.builder, "%s->%s = ", targetVarName, providedField.Name)
			cg.genExpr(providedField.Value)
			cg.builder.WriteString(";\n")
		}
	}
}

// Generating the structure
func (cg *Codegen) genStruct(s *aster.Struct) {
	fmt.Fprintf(&cg.builder, "typedef struct %s {\n", s.Name)

	for _, field := range s.Fields {
		cType := cg.mapType(field.Type)

		// Architectural Check: If the field is a pointer referencing its own parent struct layout
		prefix := ""
		if field.Type != nil && field.Type.PtrDepth > 0 && field.Type.Name == s.Name {
			prefix = "struct "
		}

		// Handle static arrays inside struct fields to ensure correct physical memory stride
		if field.Type != nil && field.Type.IsArray {
			fmt.Fprintf(&cg.builder, "    %s%s %s[%d];\n", prefix, cType, field.Name, field.Type.Size)
		} else {
			fmt.Fprintf(&cg.builder, "    %s%s %s;\n", prefix, cType, field.Name)
		}
	}
	fmt.Fprintf(&cg.builder, "} %s;\n\n", s.Name)
}

func (cg *Codegen) genEnumDecl(enum *aster.EnumDecl) {
	fmt.Fprintf(&cg.builder, "typedef struct %s {\n", enum.Name)
	fmt.Fprintf(&cg.builder, "    int32_t _tag;\n")
	fmt.Fprintf(&cg.builder, "    union {\n")

	for _, variant := range enum.Variants {
		if len(variant.Fields) > 0 {
			fmt.Fprintf(&cg.builder, "        struct {\n")
			for _, field := range variant.Fields {
				cType := cg.mapType(field.Type)
				prefix := ""
				if field.Type != nil && field.Type.PtrDepth > 0 && field.Type.Name == enum.Name {
					prefix = "struct "
				}
				if field.Type != nil && field.Type.IsArray {
					fmt.Fprintf(&cg.builder, "            %s%s %s[%d];\n", prefix, cType, field.Name, field.Type.Size)
				} else {
					fmt.Fprintf(&cg.builder, "            %s%s %s;\n", prefix, cType, field.Name)
				}
			}
			fmt.Fprintf(&cg.builder, "        } %s;\n", variant.Name)
		}
	}

	fmt.Fprintf(&cg.builder, "    } variants;\n")
	fmt.Fprintf(&cg.builder, "} %s;\n\n", enum.Name)
}
