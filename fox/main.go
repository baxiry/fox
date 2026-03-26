package main

import (
	"os"
)

func main() {

	foxFile := os.Args[1]
	data, err := os.ReadFile(foxFile)
	if err != nil {
		panic(err)
	}

	tokens := Lexer(string(data))

	Dump(astBuilder(tokens))
}
