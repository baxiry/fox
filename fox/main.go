package main

import (
	"fmt"
	"os"

	"fox/aster"
	"fox/codgen"
	"fox/runner"
	"fox/tchecker"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: fox <file.fox>")
		return
	}

	foxFile := os.Args[1]
	content, err := os.ReadFile(foxFile)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Fox source code:\n", string(content))

	// 1. Parsing phase
	parser := aster.NewParser()
	ast := parser.Builder(content)

	// Check for Syntax Errors immediately
	if len(parser.Errors) > 0 {
		fmt.Printf("Found %d syntax errors:\n", len(parser.Errors))
		for _, err := range parser.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	// 2. Type Checking phase (CRITICAL: Must happen before Codegen)
	tc := tchecker.NewTypeChecker()
	tc.Check(ast)

	// 3. Debugging: Print AST AFTER Type Checking to see the inferred types
	fmt.Println("\nAST Structure:")

	dump.Dump(ast)
	fmt.Println()

	// Check for Type Errors
	if len(tc.Errors) > 0 {
		fmt.Printf("Found %d type errors:\n", len(tc.Errors))
		for _, err := range tc.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	// 4. Prepare the Project structure for Codegen
	// Now ast.Decls contains the inferred types (decl.Type = finalType)
	project := &aster.Project{
		Packages: []*aster.Package{
			{
				Name: "main",
				Files: []*aster.File{
					{
						Path:  foxFile,
						Decls: ast.Decls,
					},
				},
			},
		},
		//	SymbolTable: tc.GlobalTable.Symbols,

		SymbolTable: tc.GlobalTable,
	}
	fmt.Println("symbols in projet", project.SymbolTable)

	// 5. Code Generation phase
	cg := codgen.NewCodegen(project)
	cCode := cg.Generate()

	fmt.Println("C Output Code:")
	fmt.Println(cCode)
	fmt.Println()

	// 6. Execution phase
	fmt.Println(" Fox Program Running ")
	runErr := runner.Run(cCode)
	if runErr != nil {
		fmt.Printf("\nRuntime Error: %v\n", runErr)
		os.Exit(1)
	}
}
