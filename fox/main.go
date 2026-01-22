package main

import (
	"os"
)

func main() {
	// Read the Fox file
	testFile := os.Args[1]
	data, err := os.ReadFile("../" + testFile)
	if err != nil {
		panic(err)
	}

	content := string(data)
	tokens := tokenize(content)

	astBuilder(tokens)
}
