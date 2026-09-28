package tchecker

import (
	"fmt"
	"fox/aster"
	"fox/symbols"

	"strings"
)

// Update built-in functions to use aster.Type
var builtInFunctions = map[string]*symbols.Type{
	"printf": {Name: "void", IsArray: false},
	"print":  {Name: "void", IsArray: false},
	"len":    {Name: "int", IsArray: false},
	"panic":  {Name: "void", IsArray: false},
}

type TypeChecker struct {
	GlobalTable     *symbols.SymbolTable
	CurrentTable    *symbols.SymbolTable
	CurrFn          *symbols.Symbol
	CurrentRetTypes *symbols.ReturnSig
	Errors          []string
	CurrentLine     int
}

// NewTypeChecker

func NewTypeChecker() *TypeChecker {

	global := symbols.NewSymbolTable(nil)
	tc := &TypeChecker{
		GlobalTable:  global,
		CurrentTable: global,
		Errors:       make([]string, 0),
	}

	tc.injectBuiltIns()
	return tc
}

func (tc *TypeChecker) injectBuiltIns() {
	// Updated built-in registration to use explicit field mapping and pointers
	tc.GlobalTable.Define("printf", &symbols.Symbol{
		Name:       "printf",
		Kind:       "func",
		IsBuiltIn:  true,
		IsVariadic: true,
		// Using pointer to symbols.Type with explicit field assignment
		Type: &symbols.Type{
			Name:     "void",
			PtrDepth: 0,
			IsArray:  false,
		},
		RetSig: &symbols.ReturnSig{
			Fields: []symbols.ReturnField{
				{
					Name: "",
					Type: &symbols.Type{
						Name:     "void",
						PtrDepth: 0,
						IsArray:  false,
					},
				},
			},
			HasError: false,
		},
	})
}

// tchecker.go
var (
	IntType    = &symbols.Type{Name: "int"}
	StringType = &symbols.Type{Name: "string"}
	BoolType   = &symbols.Type{Name: "bool"}
)

func (tc *TypeChecker) inferType(expr aster.Expression) *symbols.Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {

	case *aster.IntExpr:
		return &symbols.Type{Name: "int", PtrDepth: 0, IsArray: false}

	case *aster.StringExpr:
		return &symbols.Type{Name: "string", PtrDepth: 0, IsArray: false}

	case *aster.BoolExpr:
		return &symbols.Type{Name: "bool", PtrDepth: 0, IsArray: false}

	case *aster.IdentExpr:
		sym, exists := tc.CurrentTable.Resolve(e.Name)
		if exists && sym.Type != nil {
			e.Type = &symbols.Type{
				Name:     sym.Type.Name,
				PtrDepth: sym.Type.PtrDepth,
				IsArray:  sym.Type.IsArray,
				Size:     sym.Type.Size,
			}
			return e.Type
		}
		return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0}

	case *aster.IndexExpr:
		targetType := tc.inferType(e.Target)
		if targetType == nil || !targetType.IsArray {
			tc.appendErrorf("cannot index into non-array type", e.Line)
			return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}
		}
		return &symbols.Type{Name: targetType.Name, PtrDepth: 0, IsArray: false}

	case *aster.CallExpr:
		for _, arg := range e.Args {
			tc.inferType(arg)
		}

		callee, ok := e.Callee.(*aster.IdentExpr)
		if !ok {
			return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0}
		}

		sym, exists := tc.GlobalTable.Resolve(callee.Name)
		if !exists || sym == nil || sym.Type == nil {
			retType := &symbols.Type{Name: "void", PtrDepth: 0, IsArray: false}
			callee.Type = retType
			return retType
		}

		// If the function returns a multi-return envelope struct
		if envelopeSym, found := tc.GlobalTable.Resolve("_res_" + callee.Name); found && envelopeSym != nil {
			callee.Type = envelopeSym.Type
			return callee.Type
		}

		callee.Type = &symbols.Type{
			Name:     sym.Type.Name,
			PtrDepth: sym.Type.PtrDepth,
			IsArray:  sym.Type.IsArray,
		}

		if e.UnwrapPanic && strings.HasPrefix(callee.Type.Name, "_Result_") {
			cleanName := strings.TrimPrefix(callee.Type.Name, "_Result_")
			callee.Type = &symbols.Type{
				Name:     cleanName,
				PtrDepth: callee.Type.PtrDepth,
				IsArray:  callee.Type.IsArray,
			}
			return callee.Type
		}

		return callee.Type

	case *aster.StructLiteral:
		return tc.checkStructLiteral(e)

	case *aster.FieldAccessExpr:
		return tc.checkFieldAccess(e)

	case *aster.BinaryExpr:
		leftType := tc.inferType(e.Left)
		rightType := tc.inferType(e.Right)

		if leftType == nil || rightType == nil {
			return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}
		}

		if leftType.Name == aster.INVALID.String() || rightType.Name == aster.INVALID.String() {
			return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}
		}

		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			return &symbols.Type{Name: "bool", PtrDepth: 0, IsArray: false}
		case "&&", "||":
			return &symbols.Type{Name: "bool", PtrDepth: 0, IsArray: false}
		}

		if leftType.Name == rightType.Name && leftType.IsArray == rightType.IsArray {
			return leftType
		}

		tc.appendErrorf("type mismatch: %s and %s", e.Line, leftType.Name, rightType.Name)
		return &symbols.Type{Name: aster.INVALID.String(), PtrDepth: 0, IsArray: false}

	case *aster.UnaryExpr:
		// Fix: Route directly to your robust checkUnaryExpr function
		return tc.checkUnaryExpr(e)

	default:
		return nil
	}
}

func (tc *TypeChecker) checkBinaryExpr(expr *aster.BinaryExpr) *symbols.Type {
	// 1. Get types of both sides (now as pointers)
	leftType := tc.inferType(expr.Left)
	rightType := tc.inferType(expr.Right)

	// 2. Safety check for nil or invalid types
	if leftType == nil || rightType == nil {
		return &symbols.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 3. Strict check: Compare Name and IsArray for precision
	if leftType.Name != rightType.Name || leftType.IsArray != rightType.IsArray {
		tc.appendErrorf("type error: mismatch between %s and %s", expr.Line, leftType.Name, rightType.Name)
		return &symbols.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 4. Determine result type based on the operator
	switch expr.Op {
	case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
		// Logical/Comparison ops always return a bool Type object
		return &symbols.Type{Name: "bool", IsArray: false}

	default:
		// Arithmetic ops return the same Type object (pointer)
		return leftType
	}
}

// Correct implementation: enforce line as the first parameter
func (tc *TypeChecker) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: %s", line, msg))
}

func (tc *TypeChecker) registerFunctions(ast *aster.AST) {
	for _, decl := range ast.Decls {
		if f, ok := decl.(*aster.Func); ok {

			var funcType *symbols.Type
			var returnSignature *symbols.ReturnSig = nil

			if f.Return != nil && len(f.Return.Fields) > 0 {
				returnSignature = &symbols.ReturnSig{
					Fields:   f.Return.Fields,
					HasError: f.Return.HasError,
					Line:     f.Return.Line,
				}

				if len(f.Return.Fields) == 1 && !f.Return.HasError {
					funcType = &symbols.Type{
						Name:     f.Return.Fields[0].Type.Name,
						PtrDepth: f.Return.Fields[0].Type.PtrDepth,
						IsArray:  f.Return.Fields[0].Type.IsArray,
						Size:     f.Return.Fields[0].Type.Size,
					}
				} else {
					// إرجاع متعدد: إنشاء نوع الهيكل الضمني
					envelopeName := "_res_" + f.FuncName
					funcType = &symbols.Type{
						Name:     envelopeName,
						PtrDepth: 0,
						IsArray:  false,
					}

					// *** الإصلاح الأساسي ***
					// تسجيل الهيكل الضمني وحقوله في GlobalTable ليتعرف عليه checkFieldAccess
					structSym := &symbols.Symbol{
						Name:   envelopeName,
						Kind:   "struct",
						Fields: []symbols.StructField{},
					}
					for idx, field := range f.Return.Fields {
						fieldName := field.Name
						if fieldName == "" {
							fieldName = fmt.Sprintf("f%d", idx)
						}
						structSym.Fields = append(structSym.Fields, symbols.StructField{
							Name: fieldName,
							Type: &symbols.Type{
								Name:     field.Type.Name,
								PtrDepth: field.Type.PtrDepth,
								IsArray:  field.Type.IsArray,
								Size:     field.Type.Size,
							},
						})
					}
					tc.GlobalTable.Define(envelopeName, structSym)
				}
			} else {
				funcType = &symbols.Type{
					Name:     "void",
					PtrDepth: 0,
					IsArray:  false,
				}
			}

			sym := &symbols.Symbol{
				Name:   f.FuncName,
				Kind:   "func",
				Type:   funcType,
				RetSig: returnSignature,
				Params: mapParamsToSymbols(f.Params),
			}

			tc.GlobalTable.Define(f.FuncName, sym)
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

func (tc *TypeChecker) checkFieldAccess(expr *aster.FieldAccessExpr) *symbols.Type {
	objType := tc.inferType(expr.Object)
	if objType == nil || objType.Name == "invalid" || objType.Name == "INVALID" {
		return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
	}

	switch node := expr.Object.(type) {
	case *aster.IdentExpr:
		node.Type = objType
	case *aster.CallExpr:
		if calleeIdent, ok := node.Callee.(*aster.IdentExpr); ok {
			calleeIdent.Type = objType
		}
	}

	// تنظيف البادئة '*' لضمان الحصول على الاسم الصريح للـ Struct حتى لو كان مؤشراً
	targetTypeName := strings.TrimPrefix(objType.Name, "*")

	// 1. البحث في جدول الرموز (منطقك الأساسي الممتاز للبحث عن الـ Envelope)
	structSym, exists := tc.GlobalTable.Resolve(targetTypeName)

	if !exists || structSym == nil {
		possibleNames := []string{
			"_res_" + targetTypeName,
			"_Fox_res_" + targetTypeName,
			strings.TrimPrefix(targetTypeName, "_res_"),
		}
		for _, name := range possibleNames {
			if sym, ok := tc.GlobalTable.Resolve(name); ok && sym != nil {
				structSym = sym
				exists = true
				break
			}
		}
	}

	if !exists || structSym == nil {
		tc.appendErrorf("unknown struct type %s", expr.Line, targetTypeName)
		return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
	}

	// 2. البحث عن الحقل واستخراج نوعه مع مطابقة دقيقة لنظام symbols.Type لديك
	for _, field := range structSym.Fields {
		if field.Name == expr.Field {
			// تأكيد أن field.Type محدد وغير nil
			if field.Type == nil {
				return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
			}

			fieldTypeName := field.Type.Name
			fieldPtrDepth := field.Type.PtrDepth

			if strings.HasPrefix(fieldTypeName, "*") {
				if fieldPtrDepth == 0 {
					fieldPtrDepth = 1
				}
				fieldTypeName = strings.TrimPrefix(fieldTypeName, "*")
			}

			resType := &symbols.Type{
				Name:     fieldTypeName,
				PtrDepth: fieldPtrDepth,
				Size:     field.Type.Size,
				IsArray:  field.Type.IsArray,
			}
			return resType
		}
	}

	tc.appendErrorf("field %s not found in struct %s", expr.Line, expr.Field, targetTypeName)
	return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
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

func (tc *TypeChecker) checkReturnStmt(stmt *aster.ReturnStmt) {
	if tc.CurrFn == nil {
		tc.appendErrorf("return statement outside function", stmt.Line)
		return
	}

	retSig := tc.CurrentRetTypes
	if retSig == nil {
		retSig = tc.CurrFn.RetSig
	}

	valuesCount := len(stmt.Results)

	// 1. حالة الدالة التي لا ترجع شيئاً (Void Function)
	if retSig == nil || len(retSig.Fields) == 0 {
		if valuesCount > 0 {
			tc.appendErrorf("too many arguments to return: expected 0, got %d", stmt.Line, valuesCount)
		}
		return
	}

	expectedFields := retSig.Fields
	expectedCount := len(expectedFields)

	// 2. حالة الإرجاع الفارغ (return)
	if valuesCount == 0 {
		hasNamedReturns := false
		for _, f := range expectedFields {
			if f.Name != "" {
				hasNamedReturns = true
				break
			}
		}

		if !hasNamedReturns {
			tc.appendErrorf("missing return values: expected %d values, got 0", stmt.Line, expectedCount)
		}
		return
	}

	// 3. حالة إرجاع قيمة واحدة في دالة تتوقع أكثر من قيمة (مثلاً return err أو return user)
	if valuesCount == 1 {
		actualExpr := stmt.Results[0]
		actualType := tc.inferType(actualExpr)
		if actualType == nil || actualType.Name == aster.INVALID.String() {
			return
		}

		// إذا كانت القيمة المرجعة من نوع error/Error والدالة تنتهي بـ HasError
		if (actualType.Name == "error" || actualType.Name == "Error") && retSig.HasError {
			tc.expandReturnWithZeros(stmt, expectedFields, actualExpr, true)
			return
		}

		// إذا كانت القيمة المرجعة تطابق النوع الأول والدالة ترجع عدة قيم
		if expectedCount > 1 && actualType.IsSameAs(expectedFields[0].Type) {
			tc.expandReturnWithZeros(stmt, expectedFields, actualExpr, false)
			return
		}

		// إذا كانت الدالة تتوقع قيمة واحدة فقط في الأصل
		if expectedCount == 1 {
			tc.checkTypeMatch(expectedFields[0].Type, actualType, stmt.Line)
			return
		}
	}

	// 4. مطابقة الأعداد في حالة الإرجاع المتعدد الكامل
	if valuesCount > expectedCount {
		tc.appendErrorf("too many arguments to return: expected %d, got %d", stmt.Line, expectedCount, valuesCount)
		return
	}

	if valuesCount < expectedCount {
		tc.appendErrorf("not enough arguments to return: expected %d, got %d", stmt.Line, expectedCount, valuesCount)
		return
	}

	// 5. فحص تطابق الأنواع 1:1 لكل عنصر في القائمة
	for i, actualExpr := range stmt.Results {
		actualType := tc.inferType(actualExpr)
		if actualType == nil || actualType.Name == aster.INVALID.String() {
			continue
		}
		expectedType := expectedFields[i].Type
		tc.checkTypeMatch(expectedType, actualType, stmt.Line)
	}
}

// دالة مساعدة لفحص تطابق الأنواع
func (tc *TypeChecker) checkTypeMatch(expected, actual *symbols.Type, line int) {
	if expected == nil || actual == nil {
		return
	}
	if !expected.IsSameAs(actual) {
		tc.appendErrorf("cannot use type %s (ptr %d) as expected type %s (ptr %d) in return argument",
			line, actual.Name, actual.PtrDepth, expected.Name, expected.PtrDepth)
	}
}

// دالة التوسيع التلقائي لملء بقية القائمة بالقيم الصفرية (ZeroValueExpr)
func (tc *TypeChecker) expandReturnWithZeros(stmt *aster.ReturnStmt, fields []symbols.ReturnField, expr aster.Expression, isErrReturn bool) {
	newValues := make([]aster.Expression, len(fields))

	for i, field := range fields {
		if isErrReturn && i == len(fields)-1 {
			newValues[i] = expr
		} else if !isErrReturn && i == 0 {
			newValues[i] = expr
		} else {
			newValues[i] = &aster.ZeroValueExpr{Type: field.Type, Line: stmt.Line}
		}
	}

	stmt.Results = newValues
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

func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) []*symbols.Type {
	if expr == nil {
		return nil
	}

	// إذا كان التعبير استدعاء دالة CallExpr
	if call, ok := expr.(*aster.CallExpr); ok {
		if ident, ok := call.Callee.(*aster.IdentExpr); ok {
			// البحث عن الدالة في جدول الرموز
			if funcSym, found := tc.CurrentTable.Resolve(ident.Name); found && funcSym != nil {
				// الاستخراج من حقل RetTp كائن التوقيع المرجع لديك
				if funcSym.RetSig != nil && len(funcSym.RetSig.Fields) > 0 {
					var types []*symbols.Type
					for _, field := range funcSym.RetSig.Fields {
						types = append(types, field.Type)
					}
					return types
				}
			}
		}
	}

	// للتعبيرات العادية التي ترجع قيمة واحدة
	t := tc.inferType(expr)
	if t == nil {
		return []*symbols.Type{{Name: "INVALID"}}
	}
	return []*symbols.Type{t}
}

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

func (tc *TypeChecker) Check(a *aster.AST) {
	tc.checkGlobalVarsAndStructs(a)
	tc.registerFunctions(a)

	for _, decl := range a.Decls {
		switch d := decl.(type) {
		case *aster.Func:
			tc.checkFuncDecl(d)
		}
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

func (tc *TypeChecker) checkCallExpr(call *aster.CallExpr) *symbols.Type {
	callee, ok := call.Callee.(*aster.IdentExpr)
	if !ok {
		tc.appendErrorf("invalid call: expected a function name", call.Line)
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	sym, exists := tc.CurrentTable.Resolve(callee.Name)
	if !exists {
		tc.appendErrorf("undefined function: %s", callee.Line, callee.Name)

		rootTable := tc.CurrentTable
		for rootTable.Parent != nil {
			rootTable = rootTable.Parent
		}

		rootTable.Define(callee.Name, &symbols.Symbol{
			Name: callee.Name,
			Kind: "func",
			Type: &symbols.Type{Name: aster.INVALID.String()},
		})
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	if sym.Type == nil || sym.Type.Name == aster.INVALID.String() {
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	if sym.Kind != "func" {
		tc.appendErrorf("%s is not a function", call.Line, callee.Name)
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	argCount := len(call.Args)
	paramCount := len(sym.Params)
	if sym.IsVariadic {
		if argCount < paramCount {
			tc.appendErrorf("too few arguments in call to %s", callee.Line, callee.Name)
		}
	} else {
		if argCount != paramCount {
			tc.appendErrorf("too many or too few arguments in call to %s", callee.Line, callee.Name)
		}
	}

	for i, arg := range call.Args {
		providedType := tc.inferType(arg)

		if i < len(sym.Params) {
			expectedType := sym.Params[i].Type.Name

			if providedType != nil && expectedType != providedType.Name {
				tc.appendErrorf("cannot use %s as %s in argument to %s",
					callee.Line, providedType.Name, expectedType, callee.Name)
			}
		} else if !sym.IsVariadic {
			break
		}
	}

	if sym.Type == nil {
		return &symbols.Type{Name: "void"}
	}

	return sym.Type
}

func (tc *TypeChecker) inferFieldAccessExpr(fa *aster.FieldAccessExpr) *symbols.Type {
	// 1. استنتاج نوع الطرف الأيسر (مثل res)
	lhsType := tc.inferType(fa.Object)
	if lhsType == nil || lhsType.Name == aster.INVALID.String() {
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	// 2. البحث عن الـ Struct Symbol الخاص به في جدول الرموز
	structSym, exists := tc.CurrentTable.Resolve(lhsType.Name)
	if !exists {
		tc.appendErrorf("undefined type %s", fa.Line, lhsType.Name)
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	// 3. البحث عن الحقل المطلوب بداخل الـ Struct
	for _, field := range structSym.Fields {
		if field.Name == fa.Field {
			return field.Type
		}
	}

	tc.appendErrorf("type %s has no field %s", fa.Line, lhsType.Name, fa.Field)
	return &symbols.Type{Name: aster.INVALID.String()}
}

func (tc *TypeChecker) checkStmt(stmt aster.Statement) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {

	case *aster.VarDeclar:
		tc.checkVarDeclar(s)

	case *aster.Declar:
		tc.checkDeclar(s)

	case *aster.Assign:
		tc.checkAssign(s)

	case *aster.ExprStmt:
		if s.Expr != nil {
			exprType := tc.inferType(s.Expr)

			if call, ok := s.Expr.(*aster.CallExpr); ok && exprType != nil {
				if strings.HasPrefix(exprType.Name, "_Result_") {
					if call.UnwrapPanic {
						if tc.CurrFn == nil || tc.CurrFn.RetSig == nil || !tc.CurrFn.RetSig.HasError {
							tc.appendErrorf("cannot use early-return modifier '!' in a function that does not return an error union", call.Line)
						}
					} else {
						tc.appendErrorf("unhandled error: function returns an error union that must be consumed via 'match' or propagated with '!'", call.Line)
					}
				}
			}
		}

	case *aster.IfStmt:
		tc.checkIfStmt(s)

	case *aster.ForStmt:
		tc.checkForStmt(s)

	case *aster.ReturnStmt:
		tc.checkReturnStmt(s)

	case *aster.MatchStmt:
		tc.checkMatchStmt(s)

	default:
	}
}

func (tc *TypeChecker) checkForStmt(stmt *aster.ForStmt) {
	// 1. Create a new scope for the loop
	childScopeID := tc.CurrentTable.GenerateChildID()
	childTable := &symbols.SymbolTable{
		Symbols: make(map[string]*symbols.Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	// 2. Check the Initialization (Init) part
	if stmt.Init != nil {
		tc.checkStmt(stmt.Init)
	}

	// 3. Check the Condition (Cond) part (must be boolean)
	if stmt.Cond != nil {
		condType := tc.inferType(stmt.Cond)
		if condType != nil && condType.Name != "bool" && condType.Name != aster.INVALID.String() {
			tc.appendErrorf("non-bool condition in for statement: got %s", stmt.Cond.GetLine(), condType.Name)
		}
	}

	// 4. Check the Post-iteration (Post) part
	if stmt.Post != nil {
		tc.checkStmt(stmt.Post)
	}

	// 5. Check the Loop Body
	if stmt.Body != nil {
		tc.checkBlock(stmt.Body)
	}

	// 6. Restore context
	tc.CurrentTable = previousTable
}

func (tc *TypeChecker) checkIfStmt(stmt *aster.IfStmt) {
	// 1. Verify the condition is a boolean expression
	condType := tc.inferType(stmt.Cond)

	if condType != nil && condType.Name != "bool" && condType.Name != aster.INVALID.String() {
		tc.appendErrorf("non-bool condition in if statement: got %s", stmt.Cond.GetLine(), condType.Name)
	}

	// 2. Check the "Then" block
	if stmt.Then != nil {
		tc.checkBlock(stmt.Then)
	}

	// 3. Handle the "Else" part
	if stmt.Else != nil {
		switch e := stmt.Else.(type) {
		case *aster.IfStmt:
			tc.checkIfStmt(e)
		case *aster.FrameBlock:
			tc.checkBlock(e)
		default:
			tc.Errors = append(tc.Errors, "invalid statement in else branch")
		}
	}
}

func (tc *TypeChecker) checkSpawnStmt(spawn *aster.SpawnStmt) {
	// 1. Validate that the spawned expression is a function call
	call, ok := spawn.Call.(*aster.CallExpr)
	if !ok {
		tc.Errors = append(tc.Errors, "spawn requires a function call expression")
		return
	}

	// 2. Perform regular type checking for the call
	tc.checkCallExpr(call)

	// 3. Mark variables in arguments as Shared for safety analysis
	for _, arg := range call.Args {
		if ident, ok := arg.(*aster.IdentExpr); ok {
			sym, exists := tc.CurrentTable.Resolve(ident.Name)
			if exists {
				//  Tagging for future lock-detection warnings
				sym.IsShared = true
			}
		}
	}
}

func (tc *TypeChecker) checkUnaryExpr(expr *aster.UnaryExpr) *symbols.Type {
	// 1. Identify the operand's type
	operandType := tc.inferType(expr.Expr)
	if operandType == nil || operandType.Name == aster.INVALID.String() {
		return &symbols.Type{Name: aster.INVALID.String()}
	}

	switch expr.Op {
	case "&":
		// Fox Rule: No multi-level pointers (PtrDepth must be 0 before taking address)
		if operandType.PtrDepth >= 1 {
			tc.appendErrorf("multi-level ptr are not allowed", expr.Line)
			return &symbols.Type{Name: aster.INVALID.String()}
		}

		// Address-of: Increment the pointer depth to 1
		return &symbols.Type{
			Name:     operandType.Name,
			IsArray:  operandType.IsArray,
			PtrDepth: operandType.PtrDepth + 1,
		}

	case "*":
		// Dereference: Ensure we have exactly depth 1 to strip
		if operandType.PtrDepth <= 0 {
			tc.appendErrorf("invalid indirect: %s is not a pointer", expr.Line, operandType.Name)
			return &symbols.Type{Name: aster.INVALID.String()}
		}

		// Return a copy with PtrDepth 0
		return &symbols.Type{
			Name:     operandType.Name,
			IsArray:  operandType.IsArray,
			PtrDepth: 0,
		}

	case "!":
		// Logical Negation: Only for bool and depth 0
		if operandType.Name != "bool" || operandType.PtrDepth > 0 || operandType.IsArray {
			tc.appendErrorf("operator '!' not defined for type %s", expr.Line, operandType.Name)
			return &symbols.Type{Name: aster.INVALID.String()}
		}
		return &symbols.Type{Name: "bool", IsArray: false, PtrDepth: 0}

	case "-":
		// Numeric Negation: Only for depth 0
		if operandType.PtrDepth > 0 || operandType.IsArray {
			tc.appendErrorf("cannot use '-' on pointer or array type", expr.Line)
			return &symbols.Type{Name: aster.INVALID.String()}
		}
		return operandType

	default:
		return operandType
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

// cloneType creates a deep copy of a Type to avoid circular references
// and field mismatching.
func (tc *TypeChecker) cloneType(t *symbols.Type) *symbols.Type {
	if t == nil {
		return nil
	}
	return &symbols.Type{
		Name:     t.Name,
		PtrDepth: t.PtrDepth,
		IsArray:  t.IsArray,
		Size:     t.Size,
	}
}
