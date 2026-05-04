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
	CurrentRetTypes []*aster.Type
	CurrentLine     int
	Errors          []string
}

// Sympol
type Symbol struct {
	Name        string
	Type        aster.Type
	ScopeID     string
	Kind        string            // "var", "func", "struct"
	Params      []aster.Param     // For functions
	ReturnTypes []aster.ReturnSig // Changed: Slice of types for multiple returns
	Fields      []aster.Field     // For structs
	IsShared    bool
	IsBuiltIn   bool
	IsVariadic  bool
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

		ReturnTypes: []aster.ReturnSig{
			{Type: aster.Type{Name: "void"}},
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
			ReturnTypes: []aster.ReturnSig{
				{
					Name: "", // Built-ins usually don't have named returns
					Type: retTypeName,
					Line: 0, // Standard for injected symbols
				},
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
			tc.appendErrorf("undefined variable: %s", e.Line, e.Name)
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
		types := tc.inferReturnTypes(e)
		if len(types) == 0 {
			return &aster.Type{Name: "void", IsArray: false}
		}
		if len(types) > 1 {
			tc.appendErrorf("multiple-value call in single-value context", e.Line)
			return &aster.Type{Name: aster.INVALID.String(), IsArray: false}
		}
		// Assuming types[0] is still a string for now, wrap it
		return &aster.Type{Name: types[0].Name, IsArray: false}

	case *aster.StructLiteral:
		// Now this returns *aster.Type correctly
		return tc.checkStructLiteral(*e)

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

	case *aster.UnaryExpr:
		return tc.checkUnaryExpr(e)

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

func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) []*aster.Type {
	// 1. If it's a function call, we look up its return types in the symbol table
	if call, ok := expr.(*aster.CallExpr); ok {
		// Run checkCallExpr to validate arguments and report errors
		_ = tc.checkCallExpr(call)

		// Resolve the function name
		if callee, ok := call.Callee.(*aster.IdentExpr); ok {
			sym, exists := tc.CurrentTable.Resolve(callee.Name)
			if !exists {
				// Return a pointer to an INVALID type if function not found
				return []*aster.Type{{Name: aster.INVALID.String(), IsArray: false}}
			}

			// If the function has no return types defined
			if len(sym.ReturnTypes) == 0 {
				return []*aster.Type{{Name: "void", IsArray: false}}
			}

			// Precision: Extract pointers to the types from ReturnTypes
			var result []*aster.Type
			for i := range sym.ReturnTypes {
				// We take the address of the type stored in the symbol table
				// to maintain consistency across the checker
				result = append(result, &sym.ReturnTypes[i].Type)
			}
			return result
		}
	}

	// 2. For non-call expressions (e.g., variables, literals),
	// we infer the single type and return it as a slice of one pointer.
	singleType := tc.inferType(expr)
	if singleType == nil {
		return []*aster.Type{{Name: "unknown", IsArray: false}}
	}

	return []*aster.Type{singleType}
}

func (tc *TypeChecker) registerFunctions(ast *aster.AST) {
	for _, decl := range ast.Decls {
		if f, ok := decl.(*aster.Func); ok {

			sym := &Symbol{
				Name:        f.FuncName,
				Kind:        "func",
				ReturnTypes: f.Returns,
				Params:      f.Params,
			}

			tc.GlobalTable.Define(f.FuncName, sym)
		}
	}
}

func (tc *TypeChecker) checkVarDeclar(decl *aster.VarDeclar) {
	var finalType *aster.Type

	// 1. If type is explicit (var i int = 10)
	if decl.Type != nil {
		finalType = decl.Type
		if decl.Value != nil {
			valueType := tc.inferType(decl.Value)
			// Safety: Skip check if inferred type is nil or INVALID
			if valueType != nil && valueType.Name != aster.INVALID.String() {
				// Added PtrDepth comparison for the new pointer rules
				if finalType.Name != valueType.Name ||
					finalType.IsArray != valueType.IsArray ||
					finalType.PtrDepth != valueType.PtrDepth {
					tc.appendErrorf("cannot use %s as %s in assignment", decl.Line, valueType.Name, finalType.Name)
				}
			}
		}
	} else {
		// 2. Type inference (var i = 10)
		if decl.Value == nil {
			tc.appendErrorf("variable %s: missing expression for type inference", decl.Line, decl.Name)
			return
		}

		finalType = tc.inferType(decl.Value)

		if finalType == nil || finalType.Name == aster.INVALID.String() {
			return
		}
	}

	// 3. Register the symbol in the current table
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
	tc.CurrentRetTypes = []*aster.Type{}
	for _, ret := range fn.Returns {
		tc.CurrentRetTypes = append(tc.CurrentRetTypes, &ret.Type)
	}

	sym, _ := tc.GlobalTable.Resolve(fn.FuncName)
	tc.CurrentFunction = sym

	childScopeID := tc.CurrentTable.generateChildID()
	childTable := &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	// 3. Define parameters in the new scope
	for _, param := range fn.Params {
		// --- Start of added logic for type validation ---
		typeName := param.Type.Name
		isBuiltin := typeName == "int" || typeName == "string" || typeName == "bool"

		// Validate if the type exists (for custom types like X, Y, Z)
		if !isBuiltin {
			if _, exists := tc.GlobalTable.Resolve(typeName); !exists {
				tc.appendErrorf("undefined type: %s", param.Type.Line, typeName)
			}
		}
		// --- End of added logic ---

		paramSym := &Symbol{
			Name:    param.Name,
			Type:    param.Type,
			ScopeID: childScopeID,
		}

		// FIXED: Use param.Name as the key, not param.Type.Name
		err := tc.CurrentTable.Define(param.Name, paramSym)
		if err != nil {
			tc.Errors = append(tc.Errors, err.Error())
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

	expected := tc.CurrentFunction.ReturnTypes
	actualExprs := stmt.Results

	// 1. checke number of values
	if len(expected) != len(actualExprs) {
		tc.Errors = append(tc.Errors, fmt.Sprintf(
			"line %d: function %s expects %d return values, got %d",
			stmt.Line, tc.CurrentFunction.Name, len(expected), len(actualExprs),
		))
		return
	}

	// 2. checking match types
	for i, expr := range actualExprs {
		expectedTypeName := expected[i].Type.Name
		actualTypeName := tc.inferType(expr)

		if actualTypeName.Name == "" || actualTypeName.Name == aster.INVALID.String() {
			continue
		}

		if expectedTypeName != actualTypeName.Name {
			fmt.Println("actual type : ", actualTypeName)
			tc.appendErrorf(
				"cannot use %s as type %s in return argument",
				stmt.Line, actualTypeName, expectedTypeName,
			)
		}
	}
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
	// 1. Basic syntax check
	if len(decl.Values) == 0 {
		tc.appendErrorf("syntax error: := must have values on the right", decl.Line)
		return
	}

	// 2. Collect and unpack all types
	var rightSideTypes []*aster.Type
	for _, valExpr := range decl.Values {
		rightSideTypes = append(rightSideTypes, tc.inferReturnTypes(valExpr)...)
	}

	numNames := len(decl.Names)
	numValues := len(rightSideTypes)

	// 3. APPLY FOX FLEXIBILITY RULE:
	// If values > names, we only care if the ignored values contain an 'error'
	if numValues > numNames {
		// Rule: Check the ignored part for 'error' type
		for i := numNames; i < numValues; i++ {
			if rightSideTypes[i].Name == "error" {
				tc.appendErrorf("must handle or explicitly ignore 'error' at return position %d", decl.Line, i+1)
			}
		}
		// Slice the right side to match the names
		rightSideTypes = rightSideTypes[:numNames]
	} else if numValues < numNames {
		// Rule: Under-assignment is still a hard error (The Left is essential)

		tc.appendErrorf("assignment mismatch: %d names on the left but only %d values provided",
			decl.Line, numNames, numValues)

		// We still proceed by padding with INVALID to silence further errors
		for len(rightSideTypes) < numNames {
			rightSideTypes = append(rightSideTypes, &aster.Type{Name: aster.INVALID.String()})
		}
	}

	// 4. Register each name with its type (Safe now because lengths match)
	for i, nameExpr := range decl.Names {
		ident, ok := nameExpr.(*aster.IdentExpr)
		if !ok {
			continue
		}

		varName := ident.Name
		if varName == "_" {
			continue
		}

		inferredType := rightSideTypes[i]

		if inferredType.Name == "" || inferredType.Name == "unknown" {
			inferredType.Name = aster.INVALID.String()
		}

		if inferredType.Name == "error" {
			continue
		}

		sym := &Symbol{
			Name:    varName,
			Type:    *inferredType,
			ScopeID: tc.CurrentTable.ScopeID,
		}

		if err := tc.CurrentTable.Define(varName, sym); err != nil {
			tc.appendErrorf(err.Error(), ident.Line)
		}
	}
}

func (tc *TypeChecker) checkAssign(asgn *aster.Assign) {

	// 1. Ensure the right side is not empty
	if len(asgn.Values) == 0 {

		tc.Errors = append(tc.Errors, "syntax error: assignment must have values on the right")
		return
	}

	// 2. Collect and flatten all types from the right-hand side
	var rightSideTypes []*aster.Type
	for _, valExpr := range asgn.Values {
		// This handles both single values and multiple returns from functions
		rightSideTypes = append(rightSideTypes, tc.inferReturnTypes(valExpr)...)
	}

	// 3. Structural check: Count of variables on the left must match types on the right
	if len(asgn.Targets) != len(rightSideTypes) {

		msg := fmt.Sprintf("line %d: assignment mismatch: %d variables but %d values provided",
			asgn.Line, len(asgn.Targets), len(rightSideTypes))
		tc.Errors = append(tc.Errors, msg)
		return
	}

	// 4. Validate each variable and its type consistency

	// Replace step 4, 6, and 7 with this robust logic:
	for i := 0; i < len(asgn.Targets); i++ {
		lhsType := tc.inferType(asgn.Targets[i])
		rhsType := tc.inferType(asgn.Values[i])

		// 1. Nil Safety
		if lhsType == nil || rhsType == nil {
			continue
		}

		// 2.  INVALID
		if lhsType.Name == aster.INVALID.String() || rhsType.Name == aster.INVALID.String() {
			continue
		}

		// 3.
		if lhsType.Name != rhsType.Name ||
			lhsType.PtrDepth != rhsType.PtrDepth ||
			lhsType.IsArray != rhsType.IsArray {

			tc.appendErrorf("cannot assign %s to %s", asgn.Line, rhsType.Name, lhsType.Name)
		}
	}

}

func (tc *TypeChecker) checkStructLiteral(lit aster.StructLiteral) *aster.Type {
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
				if f.Name != "" { // تأكد من وجود اسم للحقل
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

	// 6. Handle the new ReturnTypes slice
	if len(sym.ReturnTypes) == 0 {
		return "void"
	}

	if len(sym.ReturnTypes) > 1 {
		// In a single-value context, returning multiple values is an error
		// This string will be caught by the caller (like in assignments or binary exprs)
		return "multi_value_error"
	}

	// Return the single available type name
	return sym.ReturnTypes[0].Name
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
		retTypes := tc.inferReturnTypes(expr)
		expandedRightTypes = append(expandedRightTypes, retTypes...)
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
