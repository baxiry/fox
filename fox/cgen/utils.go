package cgen

/*
func (cg *Codegen) inferType(expr aster.Expression) string {
	if expr == nil {
		return "int32_t"
	}

	// 1. Check for UnwrapPanic calls
	if call, ok := expr.(*aster.CallExpr); ok && call.UnwrapPanic {
		return "int32_t"
	}
	if bin, ok := expr.(*aster.BinaryExpr); ok {
		if leftCall, ok := bin.Left.(*aster.CallExpr); ok && leftCall.UnwrapPanic {
			return "int32_t"
		}
		if rightCall, ok := bin.Right.(*aster.CallExpr); ok && rightCall.UnwrapPanic {
			return "int32_t"
		}
	}

	// 2. Struct Literals
	if lit, ok := expr.(*aster.StructLiteral); ok && lit.Type != nil {
		return lit.Type.Name
	}

	// 3. Function Calls (Result Types & Multi-return Envelopes)
	if call, ok := expr.(*aster.CallExpr); ok {
		if callIdent, ok := call.Callee.(*aster.IdentExpr); ok {
			// Check if type is already populated in AST
			if callIdent.Type != nil {
				if strings.HasPrefix(callIdent.Type.Name, "_Result_") || strings.HasPrefix(callIdent.Type.Name, "_res_") {
					return callIdent.Type.Name
				}
			}
			// Fallback: resolution via symbol table / AST declarations
			if fn := cg.findFunc(callIdent.Name); fn != nil && fn.Return != nil {
				if len(fn.Return.Fields) > 1 || fn.Return.HasError {
					return cg.getEnvelopeName(fn.FuncName)
				} else if len(fn.Return.Fields) == 1 {
					return cg.mapType(&fn.Return.Fields[0].Type)
				}
			}
		}
	}

	// 4. Field Access (e.g., u := res.user or o := res.obj)
	if fa, ok := expr.(*aster.FieldAccessExpr); ok {
		if faType := cg.inferFieldAccessType(fa); faType != "" {
			return faType
		}
	}

	// 5. Direct Identifier / Typed Expressions
	if ident, ok := expr.(*aster.IdentExpr); ok && ident.Type != nil {
		return cg.mapType(ident.Type)
	}

	return "int32_t"
}

func (cg *Codegen) findFunc(name string) *aster.Func {
	for _, decl := range cg.unit.Decls {
		if fn, ok := decl.(*aster.Func); ok && fn.FuncName == name {
			return fn
		}
	}
	return nil
}

func (cg *Codegen) inferFieldAccessType(fa *aster.FieldAccessExpr) string {
	objIdent, ok := fa.Object.(*aster.IdentExpr)
	if !ok {
		return ""
	}

	// Inspect struct types or multi-return envelope definitions in AST
	for _, decl := range cg.unit.Decls {
		if fn, ok := decl.(*aster.Func); ok && fn.Return != nil {
			envelopeName := cg.getEnvelopeName(fn.FuncName)
			// Check if the source object matches an envelope type
			if objIdent.Type != nil && objIdent.Type.Name == envelopeName {
				for _, field := range fn.Return.Fields {
					if field.Name == fa.Field {
						return cg.mapType(&field.Type)
					}
				}
			}
		}
	}
	return ""
}

// ceneratePtrStars
func makePtrStars(ptrDepth int) string {
	if ptrDepth <= 0 {
		return ""
	}
	return strings.Repeat("*", ptrDepth)
}

func (cg *Codegen) writeIndent() {
	for i := 0; i < cg.indent; i++ {
		cg.builder.WriteString("    ")
	}
}

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
	// Querying the unified truth source via the domain tree
	structSym, exists := cg.symbolTable.Resolve(sName)
	if !exists || structSym == nil {
		return 0
	}

	totalSize := 0
	for _, field := range structSym.Fields {
		fieldSize := 0

		// Calculating physical volumes based on basic types
		if field.Type.PtrDepth > 0 || field.Type.Name == "string" {
			fieldSize = 8
		} else if field.Type.Name == "int" {
			fieldSize = 4
		} else if field.Type.Name == "bool" {
			fieldSize = 1
		} else {
			fieldSize = 8
		}
		// Multiplying the physical space step if the field is a fixed matrix
		if field.Type.IsArray && field.Type.Size > 0 {
			fieldSize = fieldSize * field.Type.Size
		}

		totalSize += fieldSize
	}

	// Physical alignment of 8 bytes to prevent gaps within the cache
	if totalSize%8 != 0 {
		totalSize = ((totalSize / 8) + 1) * 8
	}

	totalNeeded := totalSize + 8

	// Matching the final size with the fixed foxGC pools
	configurations := []int{32, 64, 128, 256, 512, 1024, 2048, 4096}
	for idx, maxCapacity := range configurations {
		if totalNeeded <= maxCapacity {
			return idx
		}
	}

	// Passing large objects to the large pool slot POOL_LARGE
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
*/
