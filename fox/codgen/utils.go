package codgen

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
