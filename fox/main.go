package main

import (
	"fmt"
	"fox/aster"
	"os"

	"github.com/kr/pretty"
)

func main() {

	foxFile := os.Args[1]
	data, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}
	sdata := string(data)
	fmt.Println("input src:\n", sdata)

	tokens := aster.Lexer(sdata)

	/*
		for k, v := range tokens {
			fmt.Print(k, " ", v.Type, " ", v.Lexeme, ", ")
		}
		println()
	*/

	fmt.Println("\nresult as AST:")
	pretty.Print(aster.Builder(tokens))
}
