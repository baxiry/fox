package main

import (
	"os"

	"github.com/kr/pretty"
)

func main() {

	foxFile := os.Args[1]
	data, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}

	tokens := Lexer(string(data))
	pretty.Print(astBuilder(tokens))
}
