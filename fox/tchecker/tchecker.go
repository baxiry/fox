package tchecker

import (
	"fmt"
	"fox/aster"
)

// Type Checker
type TypeChecker struct {
	GlobalTable     *SymbolTable
	CurrentTable    *SymbolTable
	CurrentFunction *Symbol
	CurrentRetTypes []string
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
	return &TypeChecker{
		GlobalTable:  global,
		CurrentTable: global,
		Errors:       make([]string, 0),
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

func (tc *TypeChecker) checkBinaryExpr(expr *aster.BinaryExpr) string {
	// 1. Get types of both sides
	leftType := tc.inferType(expr.Left)
	rightType := tc.inferType(expr.Right)

	// 2. Strict check: No implicit conversion
	if leftType != rightType {
		msg := fmt.Sprintf("type error: mismatch between %s and %s", leftType, rightType)
		tc.Errors = append(tc.Errors, msg)
		return aster.INVALID.String()
	}

	// 3. Determine result type
	switch expr.Op {
	case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
		return "bool" // Logical ops always return bool

	default:
		return leftType // Arithmetic ops return the same type
	}
}

// Correct implementation: enforce line as the first parameter
func (tc *TypeChecker) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: %s", line, msg))
}

func (tc *TypeChecker) inferType(expr aster.Expression) string {
	if expr == nil {
		return "unknown"
	}

	switch e := expr.(type) {

	// Literals (String, Number, Bool keep your original logic)
	case aster.StringExpr:
		return "string"
	case aster.UnaryExpr:
		return "bool"

	case aster.NumberExpr:
		return "int"

	case aster.BoolExpr:
		return "bool"

	case aster.CallExpr:
		types := tc.inferReturnTypes(e)
		if len(types) == 0 {
			return "void"
		}
		if len(types) > 1 {
			tc.appendErrorf("multiple-value call in single-value context", e.Line)
			return aster.INVALID.String()
		}
		return types[0]

	case aster.IdentExpr:
		sym, exists := tc.CurrentTable.Resolve(e.Name)
		if !exists {
			tc.appendErrorf("undefined variable: %s", e.Line, e.Name)

			rootTable := tc.CurrentTable
			for rootTable.Parent != nil {
				rootTable = rootTable.Parent
			}

			rootTable.Define(e.Name, &Symbol{
				Name:    e.Name,
				Type:    aster.Type{Name: aster.INVALID.String()},
				ScopeID: rootTable.ScopeID,
			})

			return aster.INVALID.String()
		}
		return sym.Type.Name

	case aster.FieldAccessExpr:
		// Check the left-hand side first (e.g., 'data' in 'data.x')
		leftType := tc.inferType(e.Object)
		//if leftType == "" || leftType == "INVALID" || leftType == "unknown" {
		//	fmt.Printf("DEBUG: FieldAccess failed at line %d. Object Type is: '%s'\n", e.Line, leftType)
		//}

		if leftType == aster.INVALID.String() {
			return aster.INVALID.String() // Silent return if target is already undefined
		}
		return tc.checkFieldAccess(e)

	case aster.StructLiteral:
		return tc.checkStructLiteral(e)

	case aster.BinaryExpr:
		leftType := tc.inferType(e.Left)
		rightType := tc.inferType(e.Right)

		// 3. Propagation: If either side is "error", don't report more errors here
		if leftType == aster.INVALID.String() || rightType == aster.INVALID.String() {
			return aster.INVALID.String()
		}

		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			return "bool"
		case "&&", "||":
			return "bool"
		}

		if leftType == rightType {
			return leftType
		}
		return leftType

	default:
		return "unknown"
	}
}

func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) []string {
	if call, ok := expr.(aster.CallExpr); ok {
		_ = tc.checkCallExpr(&call)

		callee, _ := call.Callee.(aster.IdentExpr)
		sym, exists := tc.CurrentTable.Resolve(callee.Name)
		if !exists {
			return []string{aster.INVALID.String()}
		}

		var names []string
		for _, field := range sym.ReturnTypes {
			// .
			typeName := field.Type.Name
			if typeName == "" {
				typeName = aster.INVALID.String()
			}
			names = append(names, typeName)
		}

		if len(names) == 0 {
			return []string{"void"}
		}
		return names
	}
	return []string{tc.inferType(expr)}
}

func (tc *TypeChecker) registerFunctions(ast *aster.AST) {
	for _, f := range ast.Funcs {
		sym := &Symbol{
			Name:        f.FuncName,
			Kind:        "func",
			ReturnTypes: f.Returns,
			Params:      f.Params,
		}

		tc.GlobalTable.Define(f.FuncName, sym)
	}
}

func (tc *TypeChecker) checkVarDeclar(decl *aster.VarDeclar) {
	var finalType string

	// 1. If type is explicit (var i int = 10)
	if decl.Type != nil {
		finalType = decl.Type.Name

		if decl.Value != nil {
			valueType := tc.inferType(decl.Value)
			// Safety: Skip check if inferred type is already an error/invalid
			if valueType != aster.INVALID.String() && finalType != valueType {
				tc.appendErrorf("cannot use %s as %s in assignment", decl.Line, valueType, finalType)
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

	// 3. Register the symbol in the current table
	sym := &Symbol{
		Name:    decl.Name,
		Type:    aster.Type{Name: finalType},
		ScopeID: tc.CurrentTable.ScopeID,
	}

	err := tc.CurrentTable.Define(decl.Name, sym)
	if err != nil {
		// IMPORTANT: We report the redeclaration error,
		// but we DON'T stop or overwrite the first definition.
		// This keeps the TypeChecker stable using the first 'i' found.
		tc.appendErrorf("var `%s` redeclared in this block", decl.Line, decl.Name)
	}
}

func (tc *TypeChecker) checkFieldAccess(expr aster.FieldAccessExpr) string {
	objType := tc.inferType(expr.Object)
	if objType == "error" || objType == "unknown" {
		return aster.INVALID.String()
	}

	actualTypeName := objType
	if len(objType) > 0 && objType[0] == '*' {
		actualTypeName = objType[1:]
	}

	structSym, exists := tc.GlobalTable.Resolve(actualTypeName)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("type %s is not defined", actualTypeName))
		return aster.INVALID.String()
	}

	if structSym.Kind != "struct" {
		tc.Errors = append(tc.Errors, fmt.Sprintf("%s is not a struct", actualTypeName))
		return aster.INVALID.String()
	}

	for _, field := range structSym.Fields {
		if field.Name == expr.Field {

			return field.Type
		}
	}

	tc.appendErrorf("type %s has no field %s", structSym.Type.Line, actualTypeName, expr.Field)
	return aster.INVALID.String()
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	tc.CurrentRetTypes = []string{}
	for _, ret := range fn.Returns {
		tc.CurrentRetTypes = append(tc.CurrentRetTypes, ret.Type.Name)
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

		if actualTypeName == "" || actualTypeName == aster.INVALID.String() {
			continue
		}

		if expectedTypeName != actualTypeName {
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
	var rightSideTypes []string
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
			if rightSideTypes[i] == "error" {
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
			rightSideTypes = append(rightSideTypes, aster.INVALID.String())
		}
	}

	// 4. Register each name with its type (Safe now because lengths match)
	for i, nameExpr := range decl.Names {
		ident, ok := nameExpr.(aster.IdentExpr)
		if !ok {
			continue
		}

		varName := ident.Name
		if varName == "_" {
			continue
		}

		inferredType := rightSideTypes[i]

		if inferredType == "" || inferredType == "unknown" {
			inferredType = aster.INVALID.String()
		}

		if inferredType == "error" {
			continue
		}

		sym := &Symbol{
			Name:    varName,
			Type:    aster.Type{Name: inferredType},
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
	var rightSideTypes []string
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
	for i, nameExpr := range asgn.Targets {

		if ident, ok := nameExpr.(aster.IdentExpr); ok && ident.Name == "_" {
			continue
		}
		// 1. inferType will handle both 'i' and 'data.x'
		// It will report "undefined variable" if it's not there
		lhsType := tc.inferType(nameExpr)
		rhsType := rightSideTypes[i]

		// 2. Only compare types if both were successfully inferred
		if lhsType != aster.INVALID.String() && rhsType != aster.INVALID.String() {
			if lhsType != rhsType {
				tc.appendErrorf("cannot assign %s to %s", nameExpr.GetLine(), rhsType, lhsType)
			}
		}
	}

}

func (tc *TypeChecker) checkStructLiteral(lit aster.StructLiteral) string {
	// 1. Lookup the Struct definition in the Global Table
	structName := lit.Type.Name
	structSym, exists := tc.GlobalTable.Resolve(structName)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("undefined type: %s", structName))
		return aster.INVALID.String()
	}

	// 2. Map fields for easy lookup during validation
	expectedFields := make(map[string]string)
	for _, f := range structSym.Fields {
		expectedFields[f.Name] = f.Type
	}

	// 3. Validate each field provided in the literal
	for _, providedField := range lit.Fields {
		expectedType, fieldExists := expectedFields[providedField.Name]

		if !fieldExists {
			tc.appendErrorf("struct %s has no field %s", providedField.Line, structName, providedField.Name)
			continue
		}

		providedType := tc.inferType(providedField.Value)

		if providedType != aster.INVALID.String() && providedType != expectedType {
			tc.appendErrorf("type mismatch in %s.%s: expected %s, got %s",
				providedField.Line, structName, providedField.Name, expectedType, providedType)
		}
	}

	// 4. Return the struct name as its type
	return structName
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

	for _, f := range a.Funcs {
		tc.checkFuncDecl(f)
	}
}

func (tc *TypeChecker) checkGlobalVarsAndStructs(ast *aster.AST) {
	// 1. Register all Structs first
	for _, s := range ast.Structs {
		sym := &Symbol{
			Name: s.Name,
			Kind: "struct",
		}
		for _, f := range s.Fields {
			if f.Name == "" {
			}
			sym.Fields = append(sym.Fields, aster.Field{
				Name: f.Name,
				Type: f.Type,
			})
		}
		tc.GlobalTable.Define(s.Name, sym)
	}

	// 2. NEW: Register and infer types for Global Variables
	for _, v := range ast.Vars {
		var finalType string

		// Handle explicit type: var a int
		if v.Type != nil {
			finalType = v.Type.Name
		} else if v.Value != nil {
			// Handle type inference: var c = 10 + 10
			finalType = tc.inferType(v.Value)
		}

		if finalType == "" {
			finalType = aster.INVALID.String()
		}
		// Register the variable in the GlobalTable
		sym := &Symbol{
			Name: v.Name,
			Type: aster.Type{Name: finalType},
			Kind: "var",
		}

		// Now functions can resolve these variables through the global scope
		tc.GlobalTable.Define(v.Name, sym)
	}
}

func (tc *TypeChecker) checkCallExpr(call *aster.CallExpr) string {
	// 1. Assert that Callee is an IdentExpr to get the Name
	callee, ok := call.Callee.(aster.IdentExpr)
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
	if len(call.Args) != len(sym.Params) {
		tc.appendErrorf("too many or too few arguments in call to %s", callee.Line, callee.Name)
		return aster.INVALID.String()
	}

	// 5. Validate each argument type against parameter type

	for i, arg := range call.Args {
		expectedType := sym.Params[i].Type.Name
		providedType := tc.inferType(arg)

		if expectedType != providedType {
			tc.appendErrorf("cannot use %s as %s in argument to %s", callee.Line, providedType, expectedType, callee.Name)
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
		condType := tc.inferType(stmt.Cond)
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
	condType := tc.inferType(stmt.Cond)

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
		if ident, ok := arg.(aster.IdentExpr); ok {
			sym, exists := tc.CurrentTable.Resolve(ident.Name)
			if exists {
				//  Tagging for future lock-detection warnings
				sym.IsShared = true
			}
		}
	}
}

func (tc *TypeChecker) checkUnaryExpr(expr *aster.UnaryExpr) string {
	// 1. Identify the operand's type (e.g., "int" or "*int")
	operandType := tc.inferType(expr.Expr)
	if operandType == "error" {
		return aster.INVALID.String()
	}

	switch expr.Op {
	case "&":
		// Address-of: Elevate to pointer
		return "*" + operandType

	case "*":
		// Dereference: Ensure it's a pointer before stripping '*'
		if len(operandType) == 0 || operandType[0] != '*' {
			msg := fmt.Sprintf("invalid indirect: %s is not a pointer", operandType)
			tc.Errors = append(tc.Errors, msg)
			return aster.INVALID.String()
		}
		return operandType[1:]

	case "!":
		// Logical Negation: Only for boolean types
		if operandType != "bool" {
			msg := fmt.Sprintf("operator '!' not defined for type %s", operandType)
			tc.Errors = append(tc.Errors, msg)
			return aster.INVALID.String()
		}
		return "bool"

	case "-":
		// Numeric Negation: Keep the numeric type (int/float)
		return operandType

	default:
		return operandType
	}
}

func (tc *TypeChecker) checkMultiAssignment(left []aster.Expression, right []aster.Expression, isDefine bool, line int) {
	var expandedRightTypes []string

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

		ident, isIdent := leftExpr.(aster.IdentExpr)
		if isIdent && ident.Name == "_" {
			continue
		}

		if isDefine && isIdent {
			// Create a Symbol pointer as required by your Define method
			newSymbol := &Symbol{
				Name: ident.Name,
				Type: aster.Type{Name: rightTypeName},
			}
			tc.GlobalTable.Define(ident.Name, newSymbol)
		} else {
			leftTypeName := tc.inferType(leftExpr)
			if leftTypeName != rightTypeName {
				tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: cannot assign %s to %s",
					line, rightTypeName, leftTypeName))
			}
		}
	}
}
