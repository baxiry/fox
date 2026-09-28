package main

import (
	"fmt"
	"fox/aster"
	"strings"
)

func DumpAST(node any, indent string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *aster.AST:
		fmt.Printf("%sAST:\n", indent)
		for _, decl := range n.Decls {
			DumpAST(decl, indent+"  ")
		}

	case *aster.Struct:
		fmt.Printf("%sStruct: %s\n", indent, n.Name)
		for _, field := range n.Fields {
			fmt.Printf("%s  field: %s", indent, field.Name)
			if field.Type != nil {
				fmt.Printf(" (type: %s)", field.Type.Name)
			}
			fmt.Printf("\n")
		}

	case *aster.Func:
		fmt.Printf("%sFunc: %s\n", indent, n.FuncName)
		if n.Return != nil && len(n.Return.Fields) > 0 {
			var fields []string
			for _, field := range n.Return.Fields {
				if field.Name != "" {
					fields = append(fields, field.Name+" "+field.Type.Name)
				} else if field.Type != nil {
					fields = append(fields, field.Type.Name)
				}
			}
			retStr := strings.Join(fields, ", ")
			if n.Return.HasError {
				retStr += " !"
			}
			fmt.Printf("%s  returns: %s\n", indent, retStr)
		}
		if n.Body != nil && len(n.Body.Stmts) > 0 {
			fmt.Printf("%s  body:\n", indent)
			for _, stmt := range n.Body.Stmts {
				DumpAST(stmt, indent+"    ")
			}
		}

	case *aster.VarDeclar:
		fmt.Printf("%sVarDecl: %s", indent, n.Name)
		if n.Type != nil {
			fmt.Printf(" (type: %s)", n.Type.Name)
		}
		fmt.Printf("\n")

	case *aster.Declar:
		fmt.Printf("%sDeclar (%s):\n", indent, n.Op)
		fmt.Printf("%s  targets:\n", indent)
		for _, target := range n.Targets {
			fmt.Printf("%s    - ", indent)
			DumpAST(target, "")
			fmt.Printf("\n")
		}
		fmt.Printf("%s  value: ", indent)
		DumpAST(n.Value, "")
		fmt.Printf("\n")

	case *aster.Assign:
		fmt.Printf("%sAssign (%s):\n", indent, n.Op)
		fmt.Printf("%s  targets:\n", indent)
		for _, target := range n.Targets {
			fmt.Printf("%s    - ", indent)
			DumpAST(target, "")
			fmt.Printf("\n")
		}
		fmt.Printf("%s  value: ", indent)
		DumpAST(n.Value, "")
		fmt.Printf("\n")

	case *aster.ExprStmt:
		fmt.Printf("%sExprStmt: ", indent)
		DumpAST(n.Expr, "")
		fmt.Printf("\n")

	case *aster.ReturnStmt:
		fmt.Printf("%sReturn:\n", indent)
		if n.Results != nil {
			for _, res := range n.Results {
				fmt.Printf("%s  - ", indent)
				DumpAST(res, "")
				fmt.Printf("\n")
			}
		} else {
			fmt.Printf("%s  value: void\n", indent)
		}

	case *aster.BinaryExpr:
		DumpAST(n.Left, "")
		fmt.Printf(" %s ", n.Op)
		DumpAST(n.Right, "")

	case *aster.IdentExpr:
		fmt.Printf("%s", n.Name)

		if n.Type != nil {
			fmt.Printf("[:%s]", n.Type.Name)
		}

	case *aster.IntExpr:
		fmt.Printf("%d", n.Value)

	case *aster.StringExpr:
		fmt.Printf("\"%s\"", n.Literal)

	case *aster.FieldAccessExpr:
		DumpAST(n.Object, "")
		fmt.Printf(".%s", n.Field)

	case *aster.CallExpr:
		DumpAST(n.Callee, "")
		if len(n.Args) > 0 {
			fmt.Printf(" args: ")
			for i, arg := range n.Args {
				DumpAST(arg, "")
				if i < len(n.Args)-1 {
					fmt.Printf(", ")
				}
			}
		} else {
			fmt.Printf(" args: none")
		}

	default:
		fmt.Printf("%sUnknownNode: %T\n", indent, n)
	}
}
