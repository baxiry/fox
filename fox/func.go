package main

import "fmt"

func parseFunc(tokens []Token, pos *int) FuncNode {
	funcNode := FuncNode{}

	// read "func"
	// read func name
	expect(tokens, pos, "func")
	funcNode.Name = expectIdent(tokens, pos).Value

	// parse function's Params
	// read "("
	// parse params
	// read ")"
	expect(tokens, pos, "(")

	for tokens[*pos].Value != ")" {
		if tokens[*pos].Value == "," {
			*pos++
		}
		name := expectIdent(tokens, pos).Value
		typ := expectIdent(tokens, pos).Value
		param := ParamNode{name, typ}
		funcNode.Params = append(funcNode.Params, param)
	}

	expect(tokens, pos, ")")

	// parse budy function
	// read "{"
	// parse body
	// read "}"

	expect(tokens, pos, "{")
	for tokens[*pos].Value != "}" {

		fmt.Println("ret: ", tokens[*pos].Value)

		if tokens[*pos].Value == "," || tokens[*pos].Value == ":" {
			*pos++
		}

		name := expectIdent(tokens, pos).Value
		fmt.Println("ret: ", tokens[*pos].Value)
		typ := expectIdent(tokens, pos).Value
		//ret := ReturnNode{Name: name, Type: typ}
		funcNode.Body += name + ":" + typ //append(funcNode.Budy, ret)
	}

	return funcNode
}
