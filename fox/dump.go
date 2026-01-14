package main

import (
	"encoding/json"
	"fmt"
)

func dump(ast *AST) {
	data, err := json.MarshalIndent(ast, ".", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}
