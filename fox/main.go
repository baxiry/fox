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
	fmt.Println("indexs & tokens:\n")
	for k, token := range tokens {
		fmt.Print(k, " ", token.Lexeme, ", ")
	}

	fmt.Println("\nresult as AST:\n")
	pretty.Print(astBuilder(tokens))
}
