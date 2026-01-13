package main

import (
	"fmt"
	"os"
)

func main() {
	// Read the Fox file
	data, err := os.ReadFile("test.fox")
	if err != nil {
		panic(err)
	}

	content := string(data)
	tokens := tokenize(content)

	// Print tokens
	for i, t := range tokens {
		fmt.Printf("%03d  `%s`:\t `%s`\n", i+1, t.Type, t.Value)
	}

	astBuilder(tokens)
}
