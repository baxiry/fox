package tchecker

import (
	"fmt"
	"fox/aster"
	"fox/symbols"
	"strings"
)

func (tc *TypeChecker) checkStructLiteral(lit *aster.StructLiteral) *symbols.Type {
	// 1. Lookup the Struct definition in the Global Table
	structName := lit.Type.Name
	structSym, exists := tc.GlobalTable.Resolve(structName)
	if !exists {
		tc.appendErrorf("undefined type: %s", lit.Line, structName)
		return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}
	}

	// 2. Map fields for easy lookup during validation
	expectedFields := make(map[string]symbols.Type)
	for _, f := range structSym.Fields {
		expectedFields[f.Name] = *f.Type
	}

	// 3. Validate and decorate each field provided in the literal
	for _, providedField := range lit.Fields {
		expectedType, fieldExists := expectedFields[providedField.Name]

		if !fieldExists {
			tc.appendErrorf("struct %s has no field %s", providedField.Line, structName, providedField.Name)
			continue
		}

		// Fix: Infer and propagate decoration down to the field values
		providedType := tc.inferType(providedField.Value)

		if providedType != nil && fmt.Sprintf("%v", providedField.Value) != "<nil>" {
			// Complete the inner node decoration if it is an identifier expression
			if ident, ok := providedField.Value.(*aster.IdentExpr); ok {
				ident.Type = providedType
			}

			// Compare Names, PtrDepth, and IsArray property accurately
			if providedType.Name != expectedType.Name ||
				providedType.PtrDepth != expectedType.PtrDepth ||
				providedType.IsArray != expectedType.IsArray {
				tc.appendErrorf("type mismatch in %s.%s: expected %s (ptr %d), got %s (ptr %d)",
					providedField.Line, structName, providedField.Name,
					expectedType.Name, expectedType.PtrDepth,
					providedType.Name, providedType.PtrDepth)
			}
		}
	}

	// 4. Return the explicit struct as an symbols.Type pointer with 0 pointer depth
	return &symbols.Type{
		Name:     structName,
		PtrDepth: 0,
		IsArray:  false,
	}
}

func (tc *TypeChecker) checkGlobalVars(vars []aster.VarDeclar) {
	// Set the context to GlobalTable
	tc.CurrentTable = tc.GlobalTable

	for _, v := range vars {
		// We reuse checkVarDeclar logic
		tc.checkVarDeclar(&v)
	}
}

func (tc *TypeChecker) checkGlobalVarsAndStructs(ast *aster.AST) {
	// 1. Register all Structs and Global Variables from Decls
	for _, decl := range ast.Decls {
		switch d := decl.(type) {

		case *aster.Struct:
			sym := &symbols.Symbol{
				Name:   d.Name,
				Kind:   "struct",
				Fields: []symbols.StructField{},
			}

			for _, f := range d.Fields {
				if f.Name != "" {
					sym.Fields = append(sym.Fields, symbols.StructField{
						Name: f.Name,
						Type: &symbols.Type{
							Name:     f.Type.Name,
							PtrDepth: f.Type.PtrDepth,
							IsArray:  f.Type.IsArray,
							Size:     f.Type.Size,
						},
					})
				}
			}
			tc.GlobalTable.Define(d.Name, sym)

		case *aster.VarDeclar:
			var finalType *symbols.Type

			// Handle explicit type: var a int
			if d.Type != nil {
				finalType = d.Type
			} else if d.Value != nil {
				// Handle type inference: var c = 10 + 10
				finalType = tc.inferType(d.Value)
			}

			if finalType == nil {
				finalType = &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}
			}

			// Register the variable in the GlobalTable using an explicit Type pointer
			sym := &symbols.Symbol{
				Name: d.Name,
				Kind: "var",
				Type: &symbols.Type{
					Name:     finalType.Name,
					PtrDepth: finalType.PtrDepth,
					IsArray:  finalType.IsArray,
				},
			}

			// Now functions can resolve these variables through the global scope
			tc.GlobalTable.Define(d.Name, sym)
		}
	}
}

func (tc *TypeChecker) checkMultiAssignment(left []aster.Expression, right []aster.Expression, isDefine bool, line int) {

	var expandedRightTypes []*symbols.Type
	for _, expr := range right {

		// استخدام inferReturnTypes لفك القيم المرجعة إذا كانت الدالة ترجع أكثر من قيمة

		retTypes := tc.inferReturnTypes(expr)
		expandedRightTypes = append(expandedRightTypes, retTypes...)
	}

	if len(left) != len(expandedRightTypes) {
		tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: assignment mismatch: %d variables but %d values",
			line, len(left), len(expandedRightTypes)))
		return
	}

	for i, leftExpr := range left {
		rightType := expandedRightTypes[i]

		ident, isIdent := leftExpr.(*aster.IdentExpr)
		if isIdent && ident.Name == "_" {
			continue
		}

		if isDefine && isIdent {
			// إسناد النوع لعقدة الـ AST حتى يراها الـ Dumper والـ Codegen
			ident.Type = rightType

			// إنشاء الرمز وتسجيله في الجدول الحالي (CurrentTable أو GlobalTable)
			newSymbol := &symbols.Symbol{
				Name: ident.Name,
				Type: rightType,
			}
			tc.CurrentTable.Define(ident.Name, newSymbol)
		} else {
			leftType := tc.inferType(leftExpr)
			// مقارنة أسماء الأنواع أو كائن النوع
			if leftType == nil || rightType == nil || leftType.Name != rightType.Name {
				tc.appendErrorf("line %d: cannot assign %s to %s", line, rightType.Name, leftType.Name)
			}
		}
	}
}

func (tc *TypeChecker) checkVarDeclar(decl *aster.VarDeclar) {
	var finalType *symbols.Type

	// 1. Determine the type (Explicit or Inferred)
	if decl.Type != nil {
		finalType = decl.Type
	} else if decl.Value != nil {
		finalType = tc.inferType(decl.Value)
	}

	if finalType == nil || finalType.Name == aster.INVALID.String() {
		return
	}

	// 2. Decorate the AST node with a clean clone
	decl.Type = tc.cloneType(finalType)

	// 3. Register the symbol with its own type copy
	sym := &symbols.Symbol{
		Name:    decl.Name,
		Type:    tc.cloneType(finalType),
		ScopeID: tc.CurrentTable.ScopeID,
	}

	if err := tc.CurrentTable.Define(decl.Name, sym); err != nil {
		tc.appendErrorf("variable `%s` redeclared", decl.Line, decl.Name)
	}
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	sym, exists := tc.GlobalTable.Resolve(fn.FuncName)
	if !exists {
		tc.appendErrorf("undefined function: %s", fn.Line, fn.FuncName)
		return
	}

	// 1. إنشاء scope الدالة الداخلي
	funcScope := symbols.NewSymbolTable(tc.CurrentTable)
	funcScope.ScopeID = tc.CurrentTable.GenerateChildID()

	previousTable := tc.CurrentTable
	tc.CurrentTable = funcScope

	// 2. تسجيل المعاملات (Parameters) داخل الدالة
	for _, param := range fn.Params {
		cleanTypeName := strings.TrimPrefix(param.Type.Name, "*")
		ptrDepth := param.Type.PtrDepth
		if strings.HasPrefix(param.Type.Name, "*") && ptrDepth == 0 {
			ptrDepth = 1
		}

		paramSym := &symbols.Symbol{
			Name:    param.Name,
			Kind:    "var",
			ScopeID: funcScope.ScopeID,
			Type: &symbols.Type{
				Name:     cleanTypeName,
				PtrDepth: ptrDepth,
				IsArray:  param.Type.IsArray,
			},
		}
		tc.CurrentTable.Define(param.Name, paramSym)
	}

	// 3. حقن المخرجات المسمّاة كمتغيرات محلية داخل الدالة
	if fn.Return != nil && len(fn.Return.Fields) > 0 {
		for _, retField := range fn.Return.Fields {
			if retField.Name != "" {
				cleanTypeName := strings.TrimPrefix(retField.Type.Name, "*")
				ptrDepth := retField.Type.PtrDepth
				if strings.HasPrefix(retField.Type.Name, "*") && ptrDepth == 0 {
					ptrDepth = 1
				}

				retSym := &symbols.Symbol{
					Name:    retField.Name,
					Kind:    "var",
					ScopeID: funcScope.ScopeID,
					Type: &symbols.Type{
						Name:     cleanTypeName,
						PtrDepth: ptrDepth,
						IsArray:  retField.Type.IsArray,
					},
				}
				tc.CurrentTable.Define(retField.Name, retSym)
			}
		}

		// 4. تحديد نوع الدالة بالنسبة للخارج (المستدعي)
		if len(fn.Return.Fields) == 1 && !fn.Return.HasError {
			sym.Type = &symbols.Type{
				Name:     strings.TrimPrefix(fn.Return.Fields[0].Type.Name, "*"),
				PtrDepth: fn.Return.Fields[0].Type.PtrDepth,
				IsArray:  fn.Return.Fields[0].Type.IsArray,
			}
		} else {
			// حالة المخرجات المتعددة: توليد Struct Symbol مطابق تماماً لبنية symbols.Symbol لديكم
			envelopeName := "_res_" + fn.FuncName

			envelopeSym := &symbols.Symbol{
				Name:    envelopeName,
				Kind:    "struct",
				ScopeID: tc.GlobalTable.ScopeID,
				Type: &symbols.Type{
					Name:     envelopeName,
					PtrDepth: 0,
					IsArray:  false,
				},
				Fields: make([]symbols.StructField, 0, len(fn.Return.Fields)),
			}

			// تعبئة حقول الـ Struct مستخدمين symbols.StructField
			for _, retField := range fn.Return.Fields {
				cleanTypeName := strings.TrimPrefix(retField.Type.Name, "*")
				ptrDepth := retField.Type.PtrDepth
				if strings.HasPrefix(retField.Type.Name, "*") && ptrDepth == 0 {
					ptrDepth = 1
				}

				envelopeSym.Fields = append(envelopeSym.Fields, symbols.StructField{
					Name: retField.Name,
					Type: &symbols.Type{
						Name:     cleanTypeName,
						PtrDepth: ptrDepth,
						IsArray:  retField.Type.IsArray,
					},
				})
			}

			// إضافة حقل الخطأ عند وجود HasError
			if fn.Return.HasError {
				envelopeSym.Fields = append(envelopeSym.Fields, symbols.StructField{
					Name: "err",
					Type: &symbols.Type{
						Name:     "error",
						PtrDepth: 0,
						IsArray:  false,
					},
				})
			}

			// تسجيل الـ Struct Symbol في جدول الرموز العام
			tc.GlobalTable.Define(envelopeName, envelopeSym)

			// تعيين نوع إرجاع الدالة ليكون اسم هذا الـ Envelope Struct
			sym.Type = &symbols.Type{
				Name:     envelopeName,
				PtrDepth: 0,
				IsArray:  false,
			}
		}

		tc.CurrentRetTypes = fn.Return
	} else {
		sym.Type = &symbols.Type{Name: "void", PtrDepth: 0, IsArray: false}
		tc.CurrentRetTypes = nil
	}

	tc.CurrFn = sym

	// 5. فحص جسم الدالة
	if fn.Body != nil {
		tc.checkBlock(fn.Body)
	}

	tc.CurrentTable = previousTable
	tc.CurrFn = nil
}

func (tc *TypeChecker) checkBlock(block *aster.FrameBlock) {
	if block == nil {
		return
	}

	// Dispatch each statement to the central checkStmt handler
	for _, stmt := range block.Stmts {
		tc.checkStmt(stmt)
	}
}

func (tc *TypeChecker) checkDeclar(decl *aster.Declar) {
	// 1. Safety check for missing value
	if decl.Value == nil {
		tc.appendErrorf("syntax error: := must have a value on the right", decl.Line)
		return
	}

	// 2. Infer the return types directly as a slice []*symbols.Type
	returnTypes := tc.inferReturnTypes(decl.Value)

	if len(returnTypes) == 0 {
		for _, target := range decl.Targets {
			if ident, ok := target.(*aster.IdentExpr); ok {
				ident.Type = &symbols.Type{Name: "INVALID"}
			}
		}
		return
	}

	// 3. Ensure targets count matches returned values count
	if len(decl.Targets) != len(returnTypes) {
		tc.appendErrorf("assignment mismatch: %d variables but right side provides %d values", decl.Line, len(decl.Targets), len(returnTypes))
		return
	}

	// 4. Iterate over targets slice and bind types to identifiers
	for i, target := range decl.Targets {
		ident, ok := target.(*aster.IdentExpr)
		if !ok {
			tc.appendErrorf("non-name on the left side of :=", decl.Line)
			continue
		}

		targetType := returnTypes[i]
		varName := ident.Name

		// Skip registration if it's the blank identifier "_"
		if varName == "_" {
			continue
		}

		// Ensure we don't declare a variable with 'void'
		if targetType.Name == "void" {
			tc.appendErrorf("cannot assign void value to variable %s", decl.Line, varName)
			ident.Type = &symbols.Type{Name: "INVALID"}
			continue
		}

		// Assign type to identifier AST node
		ident.Type = targetType

		// Register the symbol in the current table
		sym := &symbols.Symbol{
			Name:    varName,
			Type:    targetType,
			ScopeID: tc.CurrentTable.ScopeID,
		}

		if err := tc.CurrentTable.Define(varName, sym); err != nil {
			tc.appendErrorf("variable `%s` redeclared in this block", decl.Line, varName)
		}
	}
}

func (tc *TypeChecker) checkAssign(stmt *aster.Assign) {
	// 1. Safety check for missing right-hand side value
	if stmt.Value == nil {
		tc.appendErrorf("syntax error: assignment must have a value on the right", stmt.Line)
		return
	}

	// 2. Infer return types for RHS
	rhsTypes := tc.inferReturnTypes(stmt.Value)
	if len(rhsTypes) == 0 {
		return
	}

	// 3. Validate target count vs RHS values count
	if len(stmt.Targets) != len(rhsTypes) {
		tc.appendErrorf("assignment mismatch: %d targets but right side provides %d values", stmt.Line, len(stmt.Targets), len(rhsTypes))
		return
	}

	// 4. Iterate over targets slice and validate each target assignment
	for i, target := range stmt.Targets {
		// Handle identifier-based targets directly
		if ident, ok := target.(*aster.IdentExpr); ok {
			// Skip blank identifier "_"
			if ident.Name == "_" {
				continue
			}

			// Intercept undefined variables before attempting type inference
			_, exists := tc.CurrentTable.Resolve(ident.Name)
			if !exists {
				tc.appendErrorf("at checkAssign func: variable '%s' is undefined before assignment", stmt.Line, ident.Name)

				// Inject a ghost symbol to prevent duplicate cascading errors
				ghostType := &symbols.Type{Name: "INVALID", PtrDepth: 0, Size: 0, IsArray: false}
				ghostSymbol := &symbols.Symbol{
					Name: ident.Name,
					Type: ghostType,
				}
				tc.CurrentTable.Define(ident.Name, ghostSymbol)
				continue
			}
		}

		// Infer type for LHS target
		lhsType := tc.inferType(target)
		targetRhsType := rhsTypes[i]

		// Skip type compatibility checks if either side failed type inference
		if lhsType == nil || targetRhsType == nil ||
			lhsType.Name == "INVALID" ||
			targetRhsType.Name == "INVALID" {
			continue
		}

		// Ensure we cannot assign a void expression
		if targetRhsType.Name == "void" {
			tc.appendErrorf("cannot assign void value on line %d", stmt.Line)
			continue
		}

		// Validate type compatibility
		if lhsType.Name != targetRhsType.Name ||
			lhsType.PtrDepth != targetRhsType.PtrDepth ||
			lhsType.IsArray != targetRhsType.IsArray {

			tc.appendErrorf("cannot assign %s (ptr %d) to %s (ptr %d)",
				stmt.Line,
				targetRhsType.Name, targetRhsType.PtrDepth,
				lhsType.Name, lhsType.PtrDepth)
		}
	}
}
