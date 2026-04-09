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

	//for k, v := range tokens {
	//fmt.Println(k, " ", v.Lexeme)
	//}
	println()

	fmt.Println("\nresult as AST:")
	pretty.Print(astBuilder(tokens))
}
