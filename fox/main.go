package main

import (
	"fox/aster"
	"os"

	"github.com/kr/pretty"
)

func main() {

	foxFile := os.Args[1]
	content, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}

	println("input src:\n", string(content))

	parser := aster.NewParser()
	pretty.Print(parser.Builder(content))

	/*
		tc := tchecker.NewTypeChecker()

		ast := parser.Builder(content)
		//pretty.Println(ast)

		tc.Check(ast)

		println("\noutput:\n")

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
		fmt.Println("Type check passed. Running program...")

	*/
}
