package main

import (
	"fmt"
	"fox/aster"
	"fox/codgen"
	"fox/runner"
	"fox/tchecker"
	"os"
)

func main() {
	// 1. Basic validation for command line arguments
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

	fmt.Println("Fox sorce code:\n", string(content))

	// 2. Parsing phase
	parser := aster.NewParser()
	ast := parser.Builder(content)

	// Check for Syntax Errors immediately after parsing
	if len(parser.Errors) > 0 {
		fmt.Printf("Found %d syntax errors:\n", len(parser.Errors))
		for _, err := range parser.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	// 3. Type Checking phase
	tc := tchecker.NewTypeChecker()
	tc.Check(ast)

	// Check for Type Errors
	if len(tc.Errors) > 0 {
		fmt.Printf("Found %d type errors:\n", len(tc.Errors))
		for _, err := range tc.Errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	// Debugging: Print AST if needed

	fmt.Println("\nAST Structure:")
	//	pretty.Println(ast)
	//	fmt.Println()

	// 4. Prepare the Project structure for Codegen
	// We wrap the current file into a Package and then into a Project
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
	}

	// 5. Code Generation phase
	cg := codgen.NewCodegen(project)
	cCode := cg.Generate()

	fmt.Println("C Output Code:")
	fmt.Println(cCode)
	fmt.Println()

	// 6. Execution phase (using TCC)

	fmt.Println(" Fox Program Running ")
	runErr := runner.Run(cCode)
	if runErr != nil {
		fmt.Printf("\nRuntime Error: %v\n", runErr)
		os.Exit(1)
	}
}
