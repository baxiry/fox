package tchecker

import (
	"fmt"
	"fox/aster"
	"fox/symbols"
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
	CurrentFunction *symbols.Symbol
	CurrentRetTypes *aster.ReturnSig
	CurrentLine     int
	Errors          []string
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
		ReturnType: &symbols.ReturnSig{
			Type: &symbols.Type{
				Name:     "void",
				PtrDepth: 0,
				IsArray:  false,
			},
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
			// Dynamic fallback for standard functions like printf
			retType := &symbols.Type{Name: "void", PtrDepth: 0, IsArray: false}
			callee.Type = retType
			return retType
		}

		// Fix: Decorate the callee identifier node with the resolved signature
		callee.Type = &symbols.Type{
			Name:     sym.Type.Name,
			PtrDepth: sym.Type.PtrDepth,
			IsArray:  sym.Type.IsArray,
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

			// Determine the primary Type pointer for the function symbol
			var funcType *symbols.Type
			var returnSignature *symbols.ReturnSig = nil

			if f.Return != nil && f.Return.Type != nil {
				funcType = &symbols.Type{
					Name:     f.Return.Type.Name,
					PtrDepth: f.Return.Type.PtrDepth,
					IsArray:  f.Return.Type.IsArray,
					Size:     f.Return.Type.Size,
				}
				// Build the unified return signature only if the function has an actual return.
				returnSignature = &symbols.ReturnSig{
					Type: (*symbols.Type)(f.Return.Type),
				}
			} else {
				// Default to void if no return signature exists safely
				funcType = &symbols.Type{
					Name:     "void",
					PtrDepth: 0,
					IsArray:  false,
				}
			}

			sym := &symbols.Symbol{
				Name:       f.FuncName,
				Kind:       "func",
				Type:       funcType, // Crucial for Resolve() during CallExpr inference
				ReturnType: returnSignature,
				Params:     mapParamsToSymbols(f.Params),
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
	// 1. Trace entry and the object being accessed
	objType := tc.inferType(expr.Object)
	if objType == nil {
		return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
	}

	// 2. Decorating the target Object expression dynamically to clear nil gaps
	// This covers nested expressions, call returns, and standalone identifiers
	if ident, ok := expr.Object.(*aster.IdentExpr); ok {
		ident.Type = &symbols.Type{
			Name:     objType.Name,
			PtrDepth: objType.PtrDepth,
			IsArray:  objType.IsArray,
		}
	} else if call, ok := expr.Object.(*aster.CallExpr); ok {
		if calleeIdent, ok := call.Callee.(*aster.IdentExpr); ok {
			calleeIdent.Type = &symbols.Type{
				Name:     objType.Name,
				PtrDepth: objType.PtrDepth,
				IsArray:  objType.IsArray,
			}
		}
	}

	// 3. Resolve the struct in the global table
	structSym, exists := tc.GlobalTable.Resolve(objType.Name)
	if !exists {
		return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
	}

	// 4. Search for the field inside the struct fields slice
	for _, field := range structSym.Fields {
		if field.Name == expr.Field {
			return &symbols.Type{
				Name:     field.Type.Name,
				PtrDepth: field.Type.PtrDepth,
				IsArray:  field.Type.IsArray,
			}
		}
	}

	tc.appendErrorf("field %s not found in struct %s", expr.Line, expr.Field, objType.Name)
	return &symbols.Type{Name: "invalid", PtrDepth: 0, IsArray: false}
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	sym, exists := tc.GlobalTable.Resolve(fn.FuncName)
	if !exists {
		tc.appendErrorf("undefined function: %s", fn.Line, fn.FuncName)
		return
	}

	// Correctly set return signature and associate the Symbol's Type pointer
	if fn.Return != nil {
		// Use explicit field mapping to prevent positional struct errors
		sym.Type = &symbols.Type{
			Name:     fn.Return.Type.Name,
			PtrDepth: fn.Return.Type.PtrDepth,
			IsArray:  fn.Return.Type.IsArray,
		}
		tc.CurrentRetTypes = fn.Return
	} else {
		sym.Type = &symbols.Type{Name: "void", PtrDepth: 0, IsArray: false}
		tc.CurrentRetTypes = nil
	}

	tc.CurrentFunction = sym

	// Proper scope management using explicit field assignments
	childScopeID := tc.CurrentTable.GenerateChildID()

	childTable := &symbols.SymbolTable{
		Symbols: make(map[string]*symbols.Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	// Register parameters into the function's local scope
	for _, param := range fn.Params {
		typeName := param.Type.Name

		// Validate type existence (Basic or Custom Structs)
		isBuiltin := typeName == "int" || typeName == "string" || typeName == "bool" || typeName == "void"
		if !isBuiltin {
			if _, exists := tc.GlobalTable.Resolve(typeName); !exists {
				tc.appendErrorf("undefined type: %s", param.Line, typeName)
			}
		}

		// Ensure parameter type is accurately stored as a pointer
		paramSym := &symbols.Symbol{
			Name:    param.Name,
			Kind:    "var",
			ScopeID: childScopeID,
			Type: &symbols.Type{
				Name:     param.Type.Name,
				PtrDepth: param.Type.PtrDepth,
				IsArray:  param.Type.IsArray,
			},
		}

		if err := tc.CurrentTable.Define(param.Name, paramSym); err != nil {
			tc.appendErrorf(err.Error(), param.Line)
		}
	}

	if fn.Body != nil {
		tc.checkBlock(fn.Body)
	}

	// Restore previous scope after function analysis
	tc.CurrentTable = previousTable
	tc.CurrentFunction = nil
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
	// 1. Ensure the return is inside a function context
	if tc.CurrentFunction == nil {
		tc.appendErrorf("return statement outside function", stmt.Line)
		return
	}

	expectedSig := tc.CurrentFunction.ReturnType
	actualExpr := stmt.Result

	// 2. Handle void returns (no return signature or void type)
	if expectedSig == nil || expectedSig.Type.Name == "void" {
		if actualExpr != nil {
			tc.appendErrorf("too many arguments to return: expected 0, got 1", stmt.Line)
		}
		return
	}

	// 3. Infer the actual return type from the expression
	if actualExpr == nil {
		tc.appendErrorf("missing return value: expected %s", stmt.Line, expectedSig.Type.Name)
		return
	}

	actualType := tc.inferType(actualExpr)
	if actualType == nil || actualType.Name == aster.INVALID.String() {
		return // Error already reported by inferType
	}

	// 4. Validate Type compatibility (Name, PtrDepth, and IsArray)
	expectedType := expectedSig.Type
	if actualType.Name != expectedType.Name ||
		actualType.PtrDepth != expectedType.PtrDepth ||
		actualType.IsArray != expectedType.IsArray {

		tc.appendErrorf("cannot use %s (ptr %d) as type %s (ptr %d) in return argument",
			stmt.Line,
			actualType.Name, actualType.PtrDepth,
			expectedType.Name, expectedType.PtrDepth)
	}
}

func (tc *TypeChecker) checkDeclar(decl *aster.Declar) {
	// 1. Safety check for missing value
	if decl.Value == nil {
		tc.appendErrorf("syntax error: := must have a value on the right", decl.Line)
		return
	}

	// 2. Infer the type of the single value on the right
	inferredType := tc.inferType(decl.Value)
	if inferredType == nil || inferredType.Name == aster.INVALID.String() {
		// Error already reported by inferType
		return
	}

	// 3. Handle the single name on the left
	ident, ok := decl.Name.(*aster.IdentExpr)
	if !ok {
		tc.appendErrorf("non-name on the left side of :=", decl.Line)
		return
	}

	varName := ident.Name
	// Skip registration if it's the blank identifier "_"
	if varName == "_" {
		return
	}

	// 4. Ensure we don't declare a variable with 'void' or 'invalid'
	if inferredType.Name == "void" {
		tc.appendErrorf("cannot assign void value to variable %s", decl.Line, varName)
		return
	}

	// 5. Register the symbol in the current table
	sym := &symbols.Symbol{
		Name:    varName,
		Type:    inferredType,
		ScopeID: tc.CurrentTable.ScopeID,
	}

	if err := tc.CurrentTable.Define(varName, sym); err != nil {
		// Handle redeclaration error
		tc.appendErrorf("variable `%s` redeclared in this block", ident.Line, varName)
	}
}

func (tc *TypeChecker) checkAssign(stmt *aster.Assign) {
	// Step 1: Intercept completely undefined variables early using CurrentTable context
	if ident, ok := stmt.Target.(*aster.IdentExpr); ok {
		_, exists := tc.CurrentTable.Resolve(ident.Name)
		if !exists {
			// Log the true root cause directly, EXACTLY ONCE
			tc.appendErrorf("at checkAssign func: variable '%s' is undefined before assignment", stmt.Line, ident.Name)

			// Inject a ghost Symbol into the Symbols map to silence subsequent blind duplicate calls
			ghostType := &symbols.Type{Name: "INVALID", PtrDepth: 0, Size: 0, IsArray: false}
			ghostSymbol := &symbols.Symbol{
				Name: ident.Name,
				Type: ghostType,
			}

			tc.CurrentTable.Define(ident.Name, ghostSymbol)
			return // Short-circuit instantly, skipping inferType entirely
		}
	}

	// 2. Infer types safely only for symbols known to exist
	lhsType := tc.inferType(stmt.Target)
	rhsType := tc.inferType(stmt.Value)

	// 3. Immediate safety check for nil types
	if lhsType == nil || rhsType == nil {
		return
	}

	// 4. Skip validation for the blank identifier "_"
	if ident, ok := stmt.Target.(*aster.IdentExpr); ok && ident.Name == "_" {
		return
	}

	// 5. Validate type compatibility including Name, PtrDepth, and IsArray
	if lhsType.Name != rhsType.Name ||
		lhsType.PtrDepth != rhsType.PtrDepth ||
		lhsType.IsArray != rhsType.IsArray {

		tc.appendErrorf("cannot assign %s (ptr %d) to %s (ptr %d)",
			stmt.Line,
			rhsType.Name, rhsType.PtrDepth,
			lhsType.Name, lhsType.PtrDepth)
	}
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

func (tc *TypeChecker) checkCallExpr(call *aster.CallExpr) string {
	// 1. Assert that Callee is an IdentExpr to get the Name
	callee, ok := call.Callee.(*aster.IdentExpr)
	if !ok {
		tc.appendErrorf("invalid call: expected a function name", call.Line)
		return aster.INVALID.String()
	}

	// 2. Resolve the function name in the SymbolTable
	sym, exists := tc.CurrentTable.Resolve(callee.Name)
	if !exists {
		// 1. Report only once
		tc.appendErrorf("undefined function: %s", callee.Line, callee.Name)

		// 2. Silent Injection: Register as a "dummy" function to stop future errors
		rootTable := tc.CurrentTable
		for rootTable.Parent != nil {
			rootTable = rootTable.Parent
		}

		rootTable.Define(callee.Name, &symbols.Symbol{
			Name: callee.Name,
			Kind: "func",
			// We give it a special marker to avoid parameter count errors later
			Type: &symbols.Type{Name: aster.INVALID.String()},
		})
		return aster.INVALID.String()
	}

	// 3. Early exit if it's a previously flagged invalid function
	if sym.Type.Name == aster.INVALID.String() {
		return aster.INVALID.String()
	}
	// 3. Ensure the symbol is actually a function
	if sym.Kind != "func" {
		tc.appendErrorf(" %s is not a function", call.Line, callee.Name)
		return aster.INVALID.String()
	}

	// 4. Check arguments count
	argCount := len(call.Args)
	paramCount := len(sym.Params)
	if sym.IsVariadic {

		// For printf(fmt, ...), we need at least the fixed parameters (like 'fmt')
		if argCount < paramCount {
			tc.appendErrorf("too few arguments in call to %s", callee.Line, callee.Name)
		}
	} else {
		// Normal exact matching
		if argCount != paramCount {
			tc.appendErrorf("too many or too few arguments in call to %s", callee.Line, callee.Name)
		}
	}

	// 5. Validate each argument type against parameter type
	// For variadic functions, we only perform strict type checking on the fixed parameters.
	for i, arg := range call.Args {
		providedType := tc.inferType(arg)

		// Check if the current argument has a defined parameter (Fixed Parameter).
		if i < len(sym.Params) {
			// only retrieve the expected type if the index is within bounds.
			expectedType := sym.Params[i].Type.Name

			if expectedType != providedType.Name {
				tc.appendErrorf("cannot use %s as %s in argument to %s",
					callee.Line, providedType, expectedType, callee.Name)
			}
		} else if !sym.IsVariadic {
			// we break here for safety.
			break
		}
	}

	// Return the single available type name
	return sym.ReturnType.Type.Name
}

func (tc *TypeChecker) checkStmt(stmt aster.Statement) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	// 1. Explicit Variable Declarations (var i int = 10)
	case *aster.VarDeclar:
		tc.checkVarDeclar(s)

	// 2. Short Variable Declarations (i := 10)
	case *aster.Declar:
		tc.checkDeclar(s)

	// 3. Assignment Operations (x = 20, user.name = "Adam")
	case *aster.Assign:
		tc.checkAssign(s)

	// 4. Expression as a Statement (e.g., standalone function calls)
	case *aster.ExprStmt:
		if s.Expr != nil {
			tc.inferType(s.Expr)
		}

	// 5. If Conditional Statements
	case *aster.IfStmt:
		tc.checkIfStmt(s)

	// 6. For Loop Statements
	case *aster.ForStmt:
		tc.checkForStmt(s)

	// 7. Return Statements
	case *aster.ReturnStmt:
		tc.checkReturnStmt(s)

	default:
		// Unknown statement type encountered
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
		retType := tc.inferType(expr)                            // inferReturnTypes(expr)
		expandedRightTypes = append(expandedRightTypes, retType) //retTypes...)
	}

	if len(left) != len(expandedRightTypes) {
		tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: assignment mismatch: %d variables but %d values",
			line, len(left), len(expandedRightTypes)))
		return
	}

	for i, leftExpr := range left {
		rightTypeName := expandedRightTypes[i]

		ident, isIdent := leftExpr.(*aster.IdentExpr)
		if isIdent && ident.Name == "_" {
			continue
		}

		if isDefine && isIdent {
			// Create a Symbol pointer as required by your Define method
			newSymbol := &symbols.Symbol{
				Name: ident.Name,
				Type: rightTypeName,
			}
			tc.GlobalTable.Define(ident.Name, newSymbol)
		} else {
			leftTypeName := tc.inferType(leftExpr)
			if leftTypeName != rightTypeName {
				tc.appendErrorf("line %d: cannot assign %s to %s", line, rightTypeName, leftTypeName)
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
