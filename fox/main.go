package main

import (
	"os"
)

func main() {
	// Read the Fox file
	data, err := os.ReadFile("../test.fox")
	if err != nil {
		panic(err)
	}

	content := string(data)
	tokens := tokenize(content)

	astBuilder(tokens)
}
