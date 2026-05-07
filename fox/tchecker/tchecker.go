package tchecker

import (
	"fmt"
	"fox/aster"
)

// Update built-in functions to use aster.Type
var builtInFunctions = map[string]aster.Type{
	"printf": {Name: "void", IsArray: false},
	"print":  {Name: "void", IsArray: false},
	"len":    {Name: "int", IsArray: false},
	"panic":  {Name: "void", IsArray: false},
}

type TypeChecker struct {
	GlobalTable     *SymbolTable
	CurrentTable    *SymbolTable
	CurrentFunction *Symbol
	CurrentRetTypes *aster.ReturnSig
	CurrentLine     int
	Errors          []string
}

// Sympol
type Symbol struct {
	Name       string
	Type       aster.Type
	ScopeID    string
	Kind       string           // "var", "func", "struct"
	Params     []aster.Param    // For functions
	ReturnType *aster.ReturnSig // Changed: Slice of types for multiple returns
	Fields     []aster.Field    // For structs
	IsShared   bool
	IsBuiltIn  bool
	IsVariadic bool
}

// SymbolTable
type SymbolTable struct {
	Symbols map[string]*Symbol
	Parent  *SymbolTable
	ScopeID string // "1.1" , "1.2" ...
	nextID  int    // Internal counter for child scopes
}

// NewSymbolTable
func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  parent,
	}
}

// NewTypeChecker

func NewTypeChecker() *TypeChecker {

	global := NewSymbolTable(nil)
	tc := &TypeChecker{
		GlobalTable:  global,
		CurrentTable: global,
		Errors:       make([]string, 0),
	}

	tc.injectBuiltIns()
	return tc
}

func (tc *TypeChecker) injectBuiltIns() {
	tc.GlobalTable.Define("printf", &Symbol{
		Name:       "printf",
		Kind:       "func",
		IsBuiltIn:  true,
		IsVariadic: true,

		ReturnType: &aster.ReturnSig{
			Type: aster.Type{Name: "void"},
		},
	})
}

func (tc *TypeChecker) injectBuiltIns_old() {
	for name, retTypeName := range builtInFunctions {
		// Create a Symbol representing the built-in function
		sym := &Symbol{
			IsBuiltIn:  true,
			Name:       name,
			Kind:       "fn",
			IsVariadic: (name == "printf" || name == "println"),

			// Assuming Type{Name: ...} matches Type struct
			ReturnType: &aster.ReturnSig{
				Name: "", // Built-ins usually don't have named returns
				Type: retTypeName,
				Line: 0, // Standard for injected symbols
			},
		}

		if name == "printf" {
			sym.Params = []aster.Param{{Type: aster.Type{Name: "string"}, Name: "format"}}
			sym.IsVariadic = true
		}
		// Define it in the root table so it's accessible from anywhere
		tc.GlobalTable.Define(name, sym)
	}
}

func (tc *TypeChecker) inferType(expr aster.Expression) *aster.Type {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *aster.IntExpr:
		return &aster.Type{Name: "int", IsArray: false}

	case *aster.StringExpr:
		return &aster.Type{Name: "string", IsArray: false}

	case *aster.BoolExpr:
		return &aster.Type{Name: "bool", IsArray: false}

	case *aster.IdentExpr:
		sym, exists := tc.CurrentTable.Resolve(e.Name)
		if !exists {
			tc.appendErrorf("undefined variable: %v", e.GetLine(), e.Name)
			return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
		}
		// Return a pointer to the stored type directly
		return &sym.Type

	case *aster.IndexExpr:
		targetType := tc.inferType(e.Target)
		if targetType == nil || !targetType.IsArray {
			tc.appendErrorf("cannot index into non-array type", e.Line)
			return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
		}
		// Return the base type (not an array anymore)
		return &aster.Type{Name: targetType.Name, IsArray: false}

	case *aster.CallExpr:
		// 1. Identify the function name
		callee, ok := e.Callee.(*aster.IdentExpr)
		if !ok {
			return &aster.Type{Name: aster.INVALID.String()}
		}

		// 2. Look up the function in the Global Table
		sym, exists := tc.GlobalTable.Resolve(callee.Name)
		if !exists {
			// Error already reported during standard check
			return &aster.Type{Name: aster.INVALID.String()}
		}

		// 3. Return the function's return type accurately
		// We create a new Type object and copy the fields from the symbol's type
		return &aster.Type{
			Name:     sym.Type.Name,
			PtrDepth: sym.Type.PtrDepth,
			IsArray:  sym.Type.IsArray,
		}

	case *aster.StructLiteral:
		// Now this returns *aster.Type correctly
		return tc.checkStructLiteral(e)

	case *aster.FieldAccessExpr:
		// You should update checkFieldAccess to also return *aster.Type
		return tc.checkFieldAccess(*e)

	case *aster.BinaryExpr:
		leftType := tc.inferType(e.Left)
		rightType := tc.inferType(e.Right)

		if leftType == nil || rightType == nil {
			return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
		}

		if leftType.Name == aster.INVALID.String() || rightType.Name == aster.INVALID.String() {
			return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
		}

		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			return &aster.Type{Name: "bool", IsArray: false}
		case "&&", "||":
			return &aster.Type{Name: "bool", IsArray: false}
		}

		// Types match logic
		if leftType.Name == rightType.Name && leftType.IsArray == rightType.IsArray {
			return leftType
		}

		tc.appendErrorf("type mismatch: %s and %s", e.Line, leftType.Name, rightType.Name)
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}

	default:
		return nil
	}
}

func (st *SymbolTable) Define(name string, sym *Symbol) error {
	if _, exists := st.Symbols[name]; exists {
		return fmt.Errorf("error: symbol %s already defined in this scope", name)
	}
	st.Symbols[name] = sym
	return nil
}

// Resolve
func (st *SymbolTable) Resolve(name string) (*Symbol, bool) {
	// 1. get symbol from table of symols
	symb, exists := st.Symbols[name]
	if exists {
		return symb, true
	}

	// 2.  get symbol from table of parent symbols
	if st.Parent != nil {
		return st.Parent.Resolve(name)
	}

	return nil, false
}

func (tc *TypeChecker) checkBinaryExpr(expr *aster.BinaryExpr) *aster.Type {
	// 1. Get types of both sides (now as pointers)
	leftType := tc.inferType(expr.Left)
	rightType := tc.inferType(expr.Right)

	// 2. Safety check for nil or invalid types
	if leftType == nil || rightType == nil {
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 3. Strict check: Compare Name and IsArray for precision
	if leftType.Name != rightType.Name || leftType.IsArray != rightType.IsArray {
		tc.appendErrorf("type error: mismatch between %s and %s", expr.Line, leftType.Name, rightType.Name)
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 4. Determine result type based on the operator
	switch expr.Op {
	case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
		// Logical/Comparison ops always return a bool Type object
		return &aster.Type{Name: "bool", IsArray: false}

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

/*
func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) *aster.Type {
	// 1. If it's a function call, we look up its return types in the symbol table
	if call, ok := expr.(*aster.CallExpr); ok {
		// Run checkCallExpr to validate arguments and report errors
		_ = tc.checkCallExpr(call)

		// Resolve the function name
		if callee, ok := call.Callee.(*aster.IdentExpr); ok {
			sym, exists := tc.CurrentTable.Resolve(callee.Name)
			if !exists {
				// Return a pointer to an INVALID type if function not found
				return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
			}

			// If the function has no return types defined
			//if len(sym.ReturnTypes) == 0 {
			//	return []*aster.Type{{Name: "void", IsArray: false}}
			//}

			// Precision: Extract pointers to the types from ReturnTypes
			//var result []*aster.Type
			//for i := range sym.ReturnTypes {
			// We take the address of the type stored in the symbol table
			// to maintain consistency across the checker
			//result = append(result, &sym.ReturnTypes[i].Type)
			//}
			return &sym.ReturnType.Type
		}
	}

	// 2. For non-call expressions (e.g., variables, literals),
	// we infer the single type and return it as a slice of one pointer.
	singleType := tc.inferType(&expr)
	if singleType == nil {
		return &aster.Type{Name: "unknown", IsArray: false}
	}

	return singleType
}
*/

func (tc *TypeChecker) registerFunctions(ast *aster.AST) {
	for _, decl := range ast.Decls {
		if f, ok := decl.(*aster.Func); ok {

			sym := &Symbol{
				Name:       f.FuncName,
				Kind:       "func",
				ReturnType: f.Return,
				Params:     f.Params,
			}

			tc.GlobalTable.Define(f.FuncName, sym)
		}
	}
}

func (tc *TypeChecker) checkVarDeclar(decl *aster.VarDeclar) {
	var finalType *aster.Type

	// 1. If type is explicit
	if decl.Type != nil {
		finalType = decl.Type
		if decl.Value != nil {
			valueType := tc.inferType(decl.Value) // إزالة & تمت بنجاح
			if valueType != nil && valueType.Name != aster.INVALID.String() {
				if finalType.Name != valueType.Name || finalType.IsArray != valueType.IsArray {
					tc.appendErrorf("cannot use %s as %s in assignment", decl.Line, valueType.Name, finalType.Name)
				}
			}
		}

	} else {
		// 2. Type inference (var i = 10)
		if decl.Value == nil {
			tc.appendErrorf("variable %s: missing type or expression", decl.Line, decl.Name)
			return
		}
		finalType = tc.inferType(decl.Value)
	}

	if decl.Type == nil {
		decl.Type = finalType
	}

	// 3. Register the symbol
	sym := &Symbol{
		Name:    decl.Name,
		Type:    *finalType,
		ScopeID: tc.CurrentTable.ScopeID,
	}

	err := tc.CurrentTable.Define(decl.Name, sym)
	if err != nil {
		tc.appendErrorf("var `%s` redeclared in this block", decl.Line, decl.Name)
	}
}

func (tc *TypeChecker) checkFieldAccess(expr aster.FieldAccessExpr) *aster.Type {
	// 1. Infer the object type (now returns *aster.Type)
	objType := tc.inferType(expr.Object)
	if objType == nil || objType.Name == aster.INVALID.String() {
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 2. Handle Pointers or direct Structs
	// If you have a pointer property in aster.Type, use it.
	// Otherwise, we use the name to resolve the struct definition.
	actualTypeName := objType.Name

	structSym, exists := tc.GlobalTable.Resolve(actualTypeName)
	if !exists {
		tc.appendErrorf("type %s is not defined", expr.Line, actualTypeName)
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	if structSym.Kind != "struct" {
		tc.appendErrorf("%s is not a struct", expr.Line, actualTypeName)
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 3. Search for the field in the struct definition
	for _, field := range structSym.Fields {
		if field.Name == expr.Field {
			// Return the field type directly (assuming field.Type is now aster.Type)
			// If field.Type is still a string, wrap it:
			// return &aster.Type{Name: field.Type, IsArray: false}
			return &field.Type
		}
	}

	tc.appendErrorf("type %s has no field %s", expr.Line, actualTypeName, expr.Field)
	return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	sym, exists := tc.GlobalTable.Resolve(fn.FuncName)
	if !exists {
		tc.appendErrorf("undefined function: %s", fn.Line, fn.FuncName)
		return
	}

	// Fix: Directly assign the ReturnSig pointer instead of just the Type pointer
	if fn.Return != nil {
		sym.Type = fn.Return.Type
		// CurrentRetTypes should match the field type in your TypeChecker struct
		tc.CurrentRetTypes = fn.Return
		fmt.Printf("[DEBUG 1] Function Defined: %s, ReturnType: %s\n", fn.FuncName, sym.Type.Name)
	} else {
		tc.CurrentRetTypes = nil
	}

	tc.CurrentFunction = sym

	childScopeID := tc.CurrentTable.generateChildID()
	childTable := &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	for _, param := range fn.Params {
		typeName := param.Type.Name
		isBuiltin := typeName == "int" || typeName == "string" || typeName == "bool" || typeName == "void"

		if !isBuiltin {
			if _, exists := tc.GlobalTable.Resolve(typeName); !exists {
				tc.appendErrorf("undefined type: %s", param.Type.Line, typeName)
			}
		}

		paramSym := &Symbol{
			Name:    param.Name,
			Type:    param.Type,
			ScopeID: childScopeID,
		}

		if err := tc.CurrentTable.Define(param.Name, paramSym); err != nil {
			tc.appendErrorf(err.Error(), param.Type.Line)
		}
	}

	if fn.Body != nil {
		tc.checkBlock(fn.Body)
	}

	tc.CurrentTable = previousTable
	tc.CurrentFunction = nil
}

func (tc *TypeChecker) checkReturnStmt(stmt *aster.ReturnStmt) {
	if tc.CurrentFunction == nil {
		tc.Errors = append(tc.Errors, fmt.Sprintf("line %d return statement outside function", stmt.Line))
		return
	}

	expected := tc.CurrentFunction.ReturnType
	actualExpr := stmt.Result

	// 1. checke number of values

	// 2. checking match types
	//for i, expr := range actualExprs {
	expectedTypeName := expected.Type.Name
	actualTypeName := tc.inferType(actualExpr)

	if actualTypeName.Name == "" || actualTypeName.Name == aster.INVALID.String() {
		fmt.Println(actualTypeName.Name, "is void or invalid")
		//continue
	}

	if expectedTypeName != actualTypeName.Name {
		fmt.Println("actual type : ", actualTypeName)
		tc.appendErrorf(
			"cannot use %s as type %s in return argument",
			stmt.Line, actualTypeName, expectedTypeName,
		)
	}
	//}
}

func (st *SymbolTable) generateChildID() string {
	st.nextID++ // Increment internal counter
	if st.ScopeID == "" {
		return fmt.Sprintf("%d", st.nextID)
	}
	return fmt.Sprintf("%s.%d", st.ScopeID, st.nextID)
}

func (tc *TypeChecker) checkBlock(block *aster.FrameBlock) {
	if block == nil {

		return
	}

	// Iterate through each statement in the block
	for _, stmt := range block.Stmts {
		// IDIOMATIC: Use the existing checkStmt dispatcher
		// instead of re-implementing a limited switch here.
		tc.checkStmt(stmt)
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
	sym := &Symbol{
		Name:    varName,
		Type:    *inferredType,
		ScopeID: tc.CurrentTable.ScopeID,
	}

	if err := tc.CurrentTable.Define(varName, sym); err != nil {
		// Handle redeclaration error
		tc.appendErrorf("variable `%s` redeclared in this block", ident.Line, varName)
	}
}

func (tc *TypeChecker) checkAssign(stmt *aster.Assign) {
	lhsType := tc.inferType(stmt.Target)
	rhsType := tc.inferType(stmt.Value)

	if lhsType == nil || rhsType == nil {
		return
	}

	// Skip check for blank identifier "_"
	if ident, ok := stmt.Target.(*aster.IdentExpr); ok && ident.Name == "_" {
		return
	}

	if !lhsType.IsSameAs(rhsType) {
		tc.appendErrorf("cannot assign %s to %s", stmt.Line, rhsType.Name, lhsType.Name)
	}
}

func (tc *TypeChecker) checkStructLiteral(lit *aster.StructLiteral) *aster.Type {
	// 1. Lookup the Struct definition in the Global Table
	structName := lit.Type.Name
	structSym, exists := tc.GlobalTable.Resolve(structName)
	if !exists {
		tc.appendErrorf("undefined type: %s", lit.Line, structName)
		return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
	}

	// 2. Map fields for easy lookup during validation
	// Updated to use aster.Type for better type checking
	expectedFields := make(map[string]aster.Type)
	for _, f := range structSym.Fields {
		// Assuming f.Type is already of type aster.Type in your symbol table
		expectedFields[f.Name] = f.Type
	}

	// 3. Validate each field provided in the literal
	for _, providedField := range lit.Fields {
		expectedType, fieldExists := expectedFields[providedField.Name]

		if !fieldExists {
			tc.appendErrorf("struct %s has no field %s", providedField.Line, structName, providedField.Name)
			continue
		}

		// Use the new inferType2 which returns *aster.Type
		providedType := tc.inferType(providedField.Value)

		if providedType != nil && providedType.Name != aster.INVALID.String() {
			// Compare Names and IsArray property
			if providedType.Name != expectedType.Name || providedType.IsArray != expectedType.IsArray {
				tc.appendErrorf("type mismatch in %s.%s: expected %s, got %s",
					providedField.Line, structName, providedField.Name, expectedType.Name, providedType.Name)
			}
		}
	}

	// 4. Return the struct as an aster.Type pointer
	return &aster.Type{
		Name:    structName,
		IsArray: false,
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
			sym := &Symbol{
				Name: d.Name,
				Kind: "struct",
			}
			for _, f := range d.Fields {
				if f.Name != "" {
					sym.Fields = append(sym.Fields, aster.Field{
						Name: f.Name,
						Type: f.Type,
					})
				}
			}
			tc.GlobalTable.Define(d.Name, sym)

		case *aster.VarDeclar:
			var finalType string

			// Handle explicit type: var a int
			if d.Type != nil {
				finalType = d.Type.Name
			} else if d.Value != nil {
				// Handle type inference: var c = 10 + 10
				finalType = tc.inferType(d.Value).Name
			}

			if finalType == "" {
				finalType = aster.INVALID.String()
			}

			// Register the variable in the GlobalTable
			sym := &Symbol{
				Name: d.Name,
				Type: aster.Type{Name: finalType},
				Kind: "var",
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

		rootTable.Define(callee.Name, &Symbol{
			Name: callee.Name,
			Kind: "func",
			// We give it a special marker to avoid parameter count errors later
			Type: aster.Type{Name: aster.INVALID.String()},
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

	//if len(call.Args) != len(sym.Params) {
	//	tc.appendErrorf("too many or too few arguments in call to %s", callee.Line, callee.Name)
	//	return aster.INVALID.String()
	//}

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
	return sym.ReturnType.Name
}

func (tc *TypeChecker) checkForStmt(stmt *aster.ForStmt) {
	// 1. Create a new scope for the loop
	// This ensures loop variables (like 'i') don't leak out
	childScopeID := tc.CurrentTable.generateChildID()
	childTable := &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	// 2. Check the Initialization (Init) part (e.g., i := 0)

	if stmt.Init != nil {
		tc.checkStmt(stmt.Init)
	}

	// 3. Check the Condition (Cond) part (e.g., i < 10)
	if stmt.Cond != nil {
		condType := tc.inferType(stmt.Cond).Name
		if condType != "bool" && condType != aster.INVALID.String() {
			tc.appendErrorf("non-bool condition in for statement: got %s", stmt.Cond.GetLine(), condType)
		}
	}

	// 4. Check the Post-iteration (Post) part (e.g., i = i + 1)
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

func (tc *TypeChecker) checkStmt(stmt aster.Statement) {
	switch s := stmt.(type) {

	// 1. Variable Declarations (var i int)
	case *aster.VarDeclar:
		tc.checkVarDeclar(s)

	// 2. Short Declarations (i := 10)
	case *aster.Declar:
		tc.checkDeclar(s)

	// 3. Assignments (x = 20)
	case *aster.Assign:
		tc.checkAssign(s)

	// 4. Expression Statements (Function calls)
	case *aster.ExprStmt:
		tc.inferType(s.Expr)

	// 5. Control Flow (If / For)
	case *aster.IfStmt:
		tc.checkIfStmt(s)

	case *aster.ForStmt:
		tc.checkForStmt(s)

	// 6. Return Statement
	case *aster.ReturnStmt:
		tc.checkReturnStmt(s)

	default:
		//  Handle unknown statements if necessary
	}
}

func (tc *TypeChecker) checkIfStmt(stmt *aster.IfStmt) {
	// 1. Verify the condition is a boolean expression
	condType := tc.inferType(stmt.Cond).Name

	if condType != "bool" && condType != aster.INVALID.String() {
		tc.appendErrorf("non-bool condition in if statement: got %s", stmt.Cond.GetLine(), condType)
	}

	// 2. Check the "Then" block (The scope where condition is true)
	if stmt.Then != nil {
		tc.checkBlock(stmt.Then)
	}

	// 3. Handle the "Else" part (Optional)
	if stmt.Else != nil {
		// Else could be another IfStmt (else if) or a FrameBlock (else)
		switch e := stmt.Else.(type) {
		case *aster.IfStmt:
			// Recursive call for "else if" chain
			tc.checkIfStmt(e)
		case *aster.FrameBlock:
			// Direct check for "else" block
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

func (tc *TypeChecker) checkUnaryExpr(expr *aster.UnaryExpr) *aster.Type {
	// 1. Identify the operand's type
	operandType := tc.inferType(expr.Expr)
	if operandType == nil || operandType.Name == aster.INVALID.String() {
		return &aster.Type{Name: aster.INVALID.String()}
	}

	switch expr.Op {
	case "&":
		// Fox Rule: No multi-level pointers (PtrDepth must be 0 before taking address)
		if operandType.PtrDepth >= 1 {
			tc.appendErrorf("multi-level ptr are not allowed", expr.Line)
			return &aster.Type{Name: aster.INVALID.String()}
		}

		// Address-of: Increment the pointer depth to 1
		return &aster.Type{
			Name:     operandType.Name,
			IsArray:  operandType.IsArray,
			PtrDepth: operandType.PtrDepth + 1,
			Line:     expr.Line,
		}

	case "*":
		// Dereference: Ensure we have exactly depth 1 to strip
		if operandType.PtrDepth <= 0 {
			tc.appendErrorf("invalid indirect: %s is not a pointer", expr.Line, operandType.Name)
			return &aster.Type{Name: aster.INVALID.String()}
		}

		// Return a copy with PtrDepth 0
		return &aster.Type{
			Name:     operandType.Name,
			IsArray:  operandType.IsArray,
			PtrDepth: 0,
			Line:     expr.Line,
		}

	case "!":
		// Logical Negation: Only for bool and depth 0
		if operandType.Name != "bool" || operandType.PtrDepth > 0 || operandType.IsArray {
			tc.appendErrorf("operator '!' not defined for type %s", expr.Line, operandType.Name)
			return &aster.Type{Name: aster.INVALID.String()}
		}
		return &aster.Type{Name: "bool", IsArray: false, PtrDepth: 0}

	case "-":
		// Numeric Negation: Only for depth 0
		if operandType.PtrDepth > 0 || operandType.IsArray {
			tc.appendErrorf("cannot use '-' on pointer or array type", expr.Line)
			return &aster.Type{Name: aster.INVALID.String()}
		}
		return operandType

	default:
		return operandType
	}
}

func (tc *TypeChecker) checkMultiAssignment(left []aster.Expression, right []aster.Expression, isDefine bool, line int) {

	var expandedRightTypes []*aster.Type
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
			newSymbol := &Symbol{
				Name: ident.Name,
				Type: *rightTypeName,
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
