package cgen

import (
	"fmt"
	"fox/aster"
	"strings"
)

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
		cg.builder.WriteString("(")
		cg.genExpr(e.Left)
		fmt.Fprintf(&cg.builder, " %s ", e.Op)
		cg.genExpr(e.Right)
		cg.builder.WriteString(")")

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

			if cg.CurrentFunction != nil && cg.CurrentFunction.Return != nil && cg.CurrentFunction.Return.HasError {
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
