package codgen

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
		"#include <stdio.h>\n#include <stdbool.h>\n#include <stdint.h>\n#include <stdlib.h>\n#include \"foxgc/fgc.h\"\n\n")

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
		if f, ok := decl.(*aster.Func); ok && f.Return != nil && f.Return.IsErrorUnion {
			baseTypeName := f.Return.Type.Name
			envelopeName := "_Result_" + baseTypeName

			if generated[envelopeName] {
				continue
			}
			generated[envelopeName] = true

			fmt.Fprintf(&cg.builder, "typedef struct %s {\n", envelopeName)
			fmt.Fprintf(&cg.builder, "    FoxHeader header;\n")
			fmt.Fprintf(&cg.builder, "    union {\n")
			fmt.Fprintf(&cg.builder, "        %s success;\n", baseTypeName)
			fmt.Fprintf(&cg.builder, "        Error error;\n")
			fmt.Fprintf(&cg.builder, "    } value;\n")
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
	} else if f.Return != nil {
		if f.Return.IsErrorUnion {
			retType = "_Result_" + f.Return.Type.Name
		} else {
			retType = cg.mapType(f.Return.Type)
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

func (cg *Codegen) genExpr(expr aster.Expression) {
	switch e := expr.(type) {

	case *aster.UnaryExpr:
		// Handle unary operators like address-of (&)
		// Note: C uses the same symbols as Fox for these operators
		fmt.Fprintf(&cg.builder, "%s", e.Op) // Prints '&'
		cg.genExpr(e.Expr)                   // Prints the target (e.g., 'user')

	case *aster.Declar:
		if ident, ok := e.Name.(*aster.IdentExpr); ok {
			typeName := "int32_t"
			if e.Value != nil {
				isUnwrapped := false
				if call, ok := e.Value.(*aster.CallExpr); ok && call.UnwrapPanic {
					isUnwrapped = true
				} else if bin, ok := e.Value.(*aster.BinaryExpr); ok {
					if leftCall, ok := bin.Left.(*aster.CallExpr); ok && leftCall.UnwrapPanic {
						isUnwrapped = true
					}
					if rightCall, ok := bin.Right.(*aster.CallExpr); ok && rightCall.UnwrapPanic {
						isUnwrapped = true
					}
				}

				if isUnwrapped {
					typeName = "int32_t"
				} else if call, ok := e.Value.(*aster.CallExpr); ok {
					if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Type != nil {
						if strings.HasPrefix(callIdent.Type.Name, "_Result_") {
							typeName = callIdent.Type.Name
						}
					}
				} else if lit, ok := e.Value.(*aster.StructLiteral); ok && lit.Type != nil {
					typeName = lit.Type.Name
				}
			}

			fmt.Fprintf(&cg.builder, "%s %s = ", typeName, ident.Name)
			cg.genExpr(e.Value)
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
		if ident, ok := e.Object.(*aster.IdentExpr); ok && ident.Type != nil {
			typeName := strings.TrimPrefix(ident.Type.Name, "*")

			if strings.HasPrefix(typeName, "_Result_") {
				separator := "."
				if ident.Type.PtrDepth > 0 {
					separator = "->"
				}

				if e.Field == "msg" || e.Field == "code" {
					fmt.Fprintf(&cg.builder, "%s%svalue.error.%s", ident.Name, separator, e.Field)
				} else {
					fmt.Fprintf(&cg.builder, "%s%svalue.success.%s", ident.Name, separator, e.Field)
				}
				break
			}

			if typeName == "Status" {
				separator := "."
				if ident.Type.PtrDepth > 0 {
					separator = "->"
				}
				variantName := "Active"
				if e.Field == "reason" {
					variantName = "Inactive"
				}
				fmt.Fprintf(&cg.builder, "%s%svariants.%s.%s", ident.Name, separator, variantName, e.Field)
				break
			}
		}

		cg.genExpr(e.Object)
		separator := "."
		if ident, ok := e.Object.(*aster.IdentExpr); ok && ident.Type != nil {
			if ident.Type.PtrDepth > 0 {
				separator = "->"
			}
		}
		fmt.Fprintf(&cg.builder, "%s%s", separator, e.Field)

	case *aster.IntExpr:
		fmt.Fprintf(&cg.builder, "%d", e.Value)

	case *aster.StringExpr:
		fmt.Fprintf(&cg.builder, "\"%s\"", e.Literal)

	case *aster.IdentExpr:
		cg.builder.WriteString(e.Name)

	case *aster.BinaryExpr:
		cg.genExpr(e.Left)
		fmt.Fprintf(&cg.builder, " %s ", e.Op)
		cg.genExpr(e.Right)

	case *aster.CallExpr:
		if e.UnwrapPanic {
			cg.builder.WriteString("({\n")
			cg.indent++

			baseTypeName := "int"
			if ident, ok := e.Callee.(*aster.IdentExpr); ok && ident.Type != nil {
				baseTypeName = strings.TrimPrefix(ident.Type.Name, "_Result_")
			}
			envelopeName := "_Result_" + baseTypeName

			cg.writeIndent()
			fmt.Fprintf(&cg.builder, "%s __tmp_err_env = ", envelopeName)
			cg.genCall(e)
			cg.builder.WriteString(";\n")

			cg.writeIndent()
			cg.builder.WriteString("if (__tmp_err_env.header.error_flag == 1) {\n")
			cg.indent++

			// الفرز الذكي والمعزول مئة بالمئة لمسار تمرير الخطأ أو الخروج الاضطراري
			if cg.CurrentFunction != nil && cg.CurrentFunction.Return != nil && cg.CurrentFunction.Return.IsErrorUnion {
				cg.writeIndent()
				cg.builder.WriteString("return __tmp_err_env;\n")
			} else {
				cg.writeIndent()
				fmt.Fprintf(&cg.builder, "printf(\"Runtime Panic: unhandled error in function main! Message: %%s\\n\", __tmp_err_env.value.error.msg);\n")
				cg.writeIndent()
				cg.builder.WriteString("exit(1);\n")
			}

			cg.indent--
			cg.writeIndent()
			cg.builder.WriteString("}\n")

			cg.writeIndent()
			cg.builder.WriteString("__tmp_err_env.value.success;\n")

			cg.indent--
			cg.writeIndent()
			cg.builder.WriteString("})")
			break
		}

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

func (cg *Codegen) mapType(foxType *symbols.Type) string {
	if foxType == nil {
		return "int32_t"
	}

	var cType string
	typeName := foxType.Name

	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		typeName = parts[0]
	}

	if strings.HasPrefix(typeName, "_Result_") {
		cType = typeName
	} else {
		switch typeName {
		case "int":
			cType = "int32_t"
		case "string":
			cType = "char*"
		case "bool":
			cType = "bool"
		default:
			cType = typeName
		}
	}

	for i := 0; i < foxType.PtrDepth; i++ {
		cType += "*"
	}

	return cType
}

func (cg *Codegen) genStmt(stmt aster.Statement) {
	switch s := stmt.(type) {

	case *aster.IfStmt:
		cg.writeIndent()
		cg.builder.WriteString("if (")
		cg.genExpr(s.Cond)
		cg.builder.WriteString(") ")

		savedFunc := cg.CurrentFunction

		if s.Then != nil {
			cg.genBlock(s.Then)
		}

		cg.CurrentFunction = savedFunc

		if s.Else != nil {
			cg.builder.WriteString(" else ")
			if block, ok := s.Else.(*aster.FrameBlock); ok {
				cg.genBlock(block)
			} else {
				cg.writeIndent()
				cg.builder.WriteString("{\n")
				cg.indent++
				cg.writeIndent()
				cg.genStmt(s.Else)
				cg.indent--
				cg.writeIndent()
				cg.builder.WriteString("}\n")
			}
		} else {
			cg.builder.WriteString("\n")
		}

		cg.CurrentFunction = savedFunc

	case *aster.MatchStmt:
		isErrorEnvelope := false
		var objectIdentName string
		if ident, ok := s.Object.(*aster.IdentExpr); ok && ident.Type != nil {
			objectIdentName = ident.Name
			if strings.HasPrefix(ident.Type.Name, "_Result_") {
				isErrorEnvelope = true
			}
		}

		cg.writeIndent()
		if isErrorEnvelope {
			cg.builder.WriteString("switch (")
			cg.genExpr(s.Object)
			cg.builder.WriteString(".header.error_flag) {\n")
		} else {
			cg.builder.WriteString("switch (")
			cg.genExpr(s.Object)
			cg.builder.WriteString("->_tag) {\n")
		}

		for _, c := range s.Cases {
			tagValue := 0
			isErrorCase := false

			if len(c.Conditions) > 0 {
				if ident, ok := c.Conditions[0].(*aster.IdentExpr); ok {
					if isErrorEnvelope {
						if ident.Name == "Error" {
							isErrorCase = true
							tagValue = 1
						}
					} else {
						if ident.Name == "Active" {
							tagValue = 1
						} else if ident.Name == "Inactive" {
							tagValue = 2
						}
					}
				}
			}

			if isErrorEnvelope && !isErrorCase {
				tagValue = 0
			}

			cg.writeIndent()
			fmt.Fprintf(&cg.builder, "case %d:\n", tagValue)
			cg.indent++

			if c.Body != nil {
				numStmts := len(c.Body.Stmts)
				for idx, subStmt := range c.Body.Stmts {
					if idx == numStmts-1 {
						if exprStmt, ok := subStmt.(*aster.ExprStmt); ok && exprStmt.Expr != nil {
							if call, ok := exprStmt.Expr.(*aster.CallExpr); ok {
								cg.writeIndent()

								isPrintfCall := false
								if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Name == "printf" {
									isPrintfCall = true
								}

								if isErrorEnvelope && !isErrorCase && isPrintfCall {
									fmt.Fprintf(&cg.builder, "printf(")
									if len(call.Args) > 0 {
										cg.genExpr(call.Args[0])
										for k := 1; k < len(call.Args); k++ {
											cg.builder.WriteString(", ")
											if argIdent, ok := call.Args[k].(*aster.IdentExpr); ok && argIdent.Name == objectIdentName {
												fmt.Fprintf(&cg.builder, "%s.value.success", objectIdentName)
											} else {
												cg.genExpr(call.Args[k])
											}
										}
									}
									cg.builder.WriteString("); break;\n")
									continue
								}

								cg.genCall(call)
								cg.builder.WriteString("; break;\n")
								continue
							}
						}
						cg.genStmt(subStmt)
					} else {
						cg.genStmt(subStmt)
					}
				}
			}

			cg.indent--
		}

		if s.Else != nil {
			cg.writeIndent()
			cg.builder.WriteString("default:\n")
			cg.indent++

			for _, subStmt := range s.Else.Stmts {
				if exprStmt, ok := subStmt.(*aster.ExprStmt); ok && exprStmt.Expr != nil {
					if call, ok := exprStmt.Expr.(*aster.CallExpr); ok {
						isPrintfCall := false
						if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Name == "printf" {
							isPrintfCall = true
						}

						if isErrorEnvelope && isPrintfCall {
							cg.writeIndent()
							fmt.Fprintf(&cg.builder, "printf(")
							if len(call.Args) > 0 {
								cg.genExpr(call.Args[0])
								for k := 1; k < len(call.Args); k++ {
									cg.builder.WriteString(", ")
									if argIdent, ok := call.Args[k].(*aster.IdentExpr); ok && argIdent.Name == objectIdentName {
										fmt.Fprintf(&cg.builder, "%s.value.success", objectIdentName)
									} else {
										cg.genExpr(call.Args[k])
									}
								}
							}
							// إغلاق القوس الحاضن للدالة بنجاح وحقن الـ break المدمجة بسلام
							cg.builder.WriteString("); break;\n")
							continue
						}
					}
				}
				cg.writeIndent()
				cg.genStmt(subStmt)
			}

			cg.indent--
		}

		cg.writeIndent()
		cg.builder.WriteString("}\n")

	case *aster.Declar:
		cg.writeIndent()
		if ident, ok := s.Name.(*aster.IdentExpr); ok {
			typeName := "int32_t"
			if s.Value != nil {
				isUnwrapped := false
				if call, ok := s.Value.(*aster.CallExpr); ok && call.UnwrapPanic {
					isUnwrapped = true
				} else if bin, ok := s.Value.(*aster.BinaryExpr); ok {
					if leftCall, ok := bin.Left.(*aster.CallExpr); ok && leftCall.UnwrapPanic {
						isUnwrapped = true
					}
					if rightCall, ok := bin.Right.(*aster.CallExpr); ok && rightCall.UnwrapPanic {
						isUnwrapped = true
					}
				}

				if isUnwrapped {
					typeName = "int32_t"
				} else if call, ok := s.Value.(*aster.CallExpr); ok {
					if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Type != nil {
						if strings.HasPrefix(callIdent.Type.Name, "_Result_") {
							typeName = callIdent.Type.Name
						}
					}
				} else if lit, ok := s.Value.(*aster.StructLiteral); ok && lit.Type != nil {
					typeName = lit.Type.Name
				}
			}

			fmt.Fprintf(&cg.builder, "%s %s = ", typeName, ident.Name)
			cg.genExpr(s.Value)
			cg.builder.WriteString(";\n")
		}

	case *aster.VarDeclar:
		cg.writeIndent()
		cType := cg.mapType(s.Type)
		if s.Type != nil && s.Type.IsArray {
			fmt.Fprintf(&cg.builder, "%s %s[%d];\n", cType, s.Name, s.Type.Size)
		} else {
			if s.Value != nil {
				if structLit, ok := s.Value.(*aster.StructLiteral); ok && s.Type.PtrDepth > 0 {
					cg.genHeapStructLiteral(structLit, s.Name)
				} else {
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
		if cg.CurrentFunction != nil && cg.CurrentFunction.Return != nil && cg.CurrentFunction.Return.IsErrorUnion {
			baseTypeName := strings.TrimPrefix(cg.CurrentFunction.Return.Type.Name, "_Result_")
			envelopeName := "_Result_" + baseTypeName

			fmt.Fprintf(&cg.builder, "%s __ret_env;\n", envelopeName)

			isErrorValue := false
			if lit, ok := s.Result.(*aster.StructLiteral); ok {
				hasMsg := false
				hasCode := false
				for _, f := range lit.Fields {
					if f.Name == "msg" {
						hasMsg = true
					}
					if f.Name == "code" {
						hasCode = true
					}
				}
				if hasMsg && hasCode {
					isErrorValue = true
				}
			}

			if isErrorValue {
				cg.writeIndent()
				cg.builder.WriteString("__ret_env.header.error_flag = 1;\n")

				if lit, ok := s.Result.(*aster.StructLiteral); ok {
					for _, field := range lit.Fields {
						cg.writeIndent()
						fmt.Fprintf(&cg.builder, "__ret_env.value.error.%s = ", field.Name)
						cg.genExpr(field.Value)
						cg.builder.WriteString(";\n")
					}
				}
			} else {
				cg.writeIndent()
				cg.builder.WriteString("__ret_env.header.error_flag = 0;\n")
				cg.writeIndent()
				cg.builder.WriteString("__ret_env.value.success = ")
				cg.genExpr(s.Result)
				cg.builder.WriteString(";\n")
			}

			cg.writeIndent()
			cg.builder.WriteString("return __ret_env;\n")
			break
		}

		cg.builder.WriteString("return ")
		if s.Result != nil {
			cg.genExpr(s.Result)
		}
		cg.builder.WriteString(";\n")

	case *aster.ForStmt:
		cg.writeIndent()
		cg.builder.WriteString("for (")

		if s.Init != nil {
			switch initStmt := s.Init.(type) {
			case *aster.Assign:
				cg.genExpr(initStmt.Target)
				cg.builder.WriteString(" = ")
				cg.genExpr(initStmt.Value)
			case *aster.Declar:
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
		if variantName == "Active" {
			generatedTagValue = 1
		} else if variantName == "Inactive" {
			generatedTagValue = 2
		}

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
