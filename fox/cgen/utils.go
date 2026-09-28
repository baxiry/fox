package cgen

import (
	"fox/aster"
	"fox/symbols"
	"strings"
)

func (cg *Codegen) structHasPointers(sName string) int {
	structSym, exists := cg.symbolTable.Resolve(sName)
	if !exists || structSym == nil {
		return 0
	}

	for _, field := range structSym.Fields {
		if field.Type.PtrDepth > 0 || field.Type.Name == "string" {
			return 1
		}
	}

	return 0
}

func (cg *Codegen) calculateClassIndex(sName string) int {
	structSym, exists := cg.symbolTable.Resolve(sName)
	if !exists || structSym == nil {
		return 0
	}

	totalSize := 0
	for _, field := range structSym.Fields {
		fieldSize := 0

		if field.Type.PtrDepth > 0 || field.Type.Name == "string" {
			fieldSize = 8
		} else if field.Type.Name == "int" {
			fieldSize = 4
		} else if field.Type.Name == "bool" {
			fieldSize = 1
		} else {
			fieldSize = 8
		}

		if field.Type.IsArray && field.Type.Size > 0 {
			fieldSize = fieldSize * field.Type.Size
		}

		totalSize += fieldSize
	}

	if totalSize%8 != 0 {
		totalSize = ((totalSize / 8) + 1) * 8
	}

	totalNeeded := totalSize + 8

	configurations := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}
	for idx, maxCapacity := range configurations {
		if totalNeeded <= maxCapacity {
			return idx
		}
	}

	return 8
}

func (cg *Codegen) mapType(foxType *symbols.Type) string {
	if foxType == nil {
		return "int32_t"
	}

	var cType string
	typeName := foxType.Name

	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		typeName = parts[0]
	}

	if strings.HasPrefix(typeName, "_Result_") {
		cType = typeName
	} else {
		switch typeName {
		case "int":
			cType = "int32_t"
		case "string":
			cType = "char*"
		case "bool":
			cType = "bool"
		default:
			cType = typeName
		}
	}

	for i := 0; i < foxType.PtrDepth; i++ {
		cType += "*"
	}

	return cType
}

func (cg *Codegen) findFunc(name string) *aster.Func {
	for _, decl := range cg.unit.Decls {
		if fn, ok := decl.(*aster.Func); ok && fn.FuncName == name {
			return fn
		}
	}
	return nil
}

func (cg *Codegen) writeIndent() {
	for i := 0; i < cg.indent; i++ {
		cg.sourceStream.WriteString("    ")
	}
}
