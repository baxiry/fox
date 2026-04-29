package main

import (
	"fmt"
	"fox/aster"
	"fox/codgen"
	"fox/tchecker"
	"fox/wrrap"
	"os"
)

func main() {
	_ = wrrap.Context{}

	foxFile := os.Args[1]
	content, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}

	parser := aster.NewParser()

	//println("input src:\n", string(content))
	//pretty.Print(parser.Builder(content))

	tc := tchecker.NewTypeChecker()

	ast := parser.Builder(content)

	tc.Check(ast)

	//println("\noutput:\n")

	// ast errors
	if len(parser.Errors) > 0 {
		fmt.Printf("Found %d syntax errors:\n", len(parser.Errors))
		for _, err := range parser.Errors {
			fmt.Printf("  - %s\n", err)
		}
		println()
		// Exit the program with an error code to prevent execution
		os.Exit(1)
	}

	// check errors
	if len(tc.Errors) > 0 {
		fmt.Printf("Found %d type errors:\n", len(tc.Errors))
		for _, err := range tc.Errors {
			fmt.Printf("  - %s\n", err)
		}
		// Exit the program with an error code to prevent execution
		os.Exit(1)
	}

	// ok
	println("Type check passed. Running program...\n")
	println("execution output:...\n")

	// Wrap your single file AST into a project structure
	project := &aster.Project{
		Packages: []*aster.Package{
			{
				Name: "main",
				Files: []*aster.File{
					{Path: foxFile, Decls: ast.Decls}, // Using Decls from your current ast
				},
			},
		},
	}

	cg := codgen.NewCodegen(project)
	cg.Generate()

}
