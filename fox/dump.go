package main

import (
	"fmt"
	"fox/aster"

	"github.com/davecgh/go-spew/spew"
)

var dump = spew.ConfigState{
	Indent:                  "    ",
	MaxDepth:                8,
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	ContinueOnMethod:        true,
}

func DumpAST(node interface{}, indent string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *aster.AST:
		fmt.Printf("%sAST:\n", indent)
		for _, decl := range n.Decls {
			DumpAST(decl, indent+"    ")
		}

	case *aster.Struct:
		fmt.Printf("%sStruct: %s\n", indent, n.Name)
		for _, field := range n.Fields {
			fmt.Printf("%s    Field: %s (Type: %s)\n", indent, field.Name, field.Type.Name)
		}

	case *aster.Func:
		retName := "void"
		if n.Return != nil {
			retName = n.Return.Type.Name
			if n.Return.IsErrorUnion {
				retName += "!"
			}
		}
		fmt.Printf("%sFunc: %s() %s\n", indent, n.FuncName, retName)
		if n.Body != nil {
			for _, stmt := range n.Body.Stmts {
				DumpAST(stmt, indent+"    ")
			}
		}

	case *aster.IfStmt:
		fmt.Printf("%sIfStmt (Cond: ", indent)
		DumpAST(n.Cond, "")
		fmt.Printf(")\n")
		if n.Then != nil {
			fmt.Printf("%s    Then:\n", indent)
			for _, stmt := range n.Then.Stmts {
				DumpAST(stmt, indent+"        ")
			}
		}
		if n.Else != nil {
			fmt.Printf("%s    Else:\n", indent)
			if block, ok := n.Else.(*aster.FrameBlock); ok {
				for _, stmt := range block.Stmts {
					DumpAST(stmt, indent+"        ")
				}
			} else {
				DumpAST(n.Else, indent+"        ")
			}
		}

	case *aster.ReturnStmt:
		fmt.Printf("%sReturnStmt: ", indent)
		if n.Result != nil {
			DumpAST(n.Result, "")
		} else {
			fmt.Printf("void")
		}
		fmt.Printf("\n")

	case *aster.MatchStmt:
		fmt.Printf("%sMatchStmt (Object: ", indent)
		DumpAST(n.Object, "")
		fmt.Printf(")\n")
		for _, c := range n.Cases {
			fmt.Printf("%s    Case: ", indent)
			for i, cond := range c.Conditions {
				DumpAST(cond, "")
				if i < len(c.Conditions)-1 {
					fmt.Printf(", ")
				}
			}
			fmt.Printf("\n")
			if c.Body != nil {
				for _, stmt := range c.Body.Stmts {
					DumpAST(stmt, indent+"        ")
				}
			}
		}
		if n.Else != nil {
			fmt.Printf("%s    Else:\n", indent)
			for _, stmt := range n.Else.Stmts {
				DumpAST(stmt, indent+"        ")
			}
		}

	case *aster.Declar:
		fmt.Printf("%sDeclar: ", indent)
		DumpAST(n.Name, "")
		fmt.Printf(" %s ", n.Op)
		DumpAST(n.Value, "")
		fmt.Printf("\n")

	case *aster.ExprStmt:
		DumpAST(n.Expr, indent)
		fmt.Printf("\n")

	// تفكيك التعبيرات الفرعية المتداخلة لطباعة محتواها بدقة خطية
	case *aster.BinaryExpr:
		DumpAST(n.Left, "")
		fmt.Printf(" %s ", n.Op)
		DumpAST(n.Right, "")

	case *aster.IdentExpr:
		fmt.Printf("%s", n.Name)

	case *aster.IntExpr:
		fmt.Printf("%d", n.Value)

	case *aster.StringExpr:
		fmt.Printf("\"%s\"", n.Literal)

	case *aster.CallExpr:
		DumpAST(n.Callee, indent)
		fmt.Printf("(")
		for i, arg := range n.Args {
			DumpAST(arg, "")
			if i < len(n.Args)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Printf(")")

	case *aster.StructLiteral:
		typeName := "Unknown"
		if n.Type != nil {
			typeName = n.Type.Name
		}
		fmt.Printf("%s{", typeName)
		for i, f := range n.Fields {
			fmt.Printf("%s: ", f.Name)
			DumpAST(f.Value, "")
			if i < len(n.Fields)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Printf("}")
	}
}
