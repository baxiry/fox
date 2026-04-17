package main

import (
	"fmt"
	"fox/aster"
	"fox/tchecker"
	"os"
)

func main() {

	foxFile := os.Args[1]
	data, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}

	fmt.Println("input src:\n", string(data))

	parser := aster.NewParser()

	/*
		for k, v := range tokens {
			fmt.Print(k, " ", v.Type, " ", v.Lexeme, ", ")
		}
		println()

		fmt.Println("\nresult as AST:")

		pretty.Print(parser.Builder(data))
	*/

	// 2. Type Checking stage
	tc := tchecker.NewTypeChecker()
	tc.Check(parser.Builder(data))

	// 3. IMPORTANT: Stop if there are semantic errors
	if len(tc.Errors) > 0 {
		fmt.Printf("Found %d type errors:\n", len(tc.Errors))
		for _, err := range tc.Errors {
			fmt.Printf("  - %s\n", err)
		}
		// Exit the program with an error code to prevent execution
		os.Exit(1)
	}

	// 4. Execution stage (Only if no errors were found)
	fmt.Println("Type check passed. Running program...")
	// runInterpreter(ast)

}
