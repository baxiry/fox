package main

import (
	"fmt"
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

	tokens := Lexer(sdata)

	fmt.Println("lexem.len == 2:\n", sdata)
	for k, v := range tokens {
		if len(v.Lexeme) == 2 {
			fmt.Println(k, " ", v.Lexeme)
		}
	}
	println()

	fmt.Println("\nresult as AST:")
	pretty.Print(astBuilder(tokens))
}
