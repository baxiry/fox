package tcheck

import (
	"fmt"
	"fox/aster"
)

// Type Checker
type TypeChecker struct {
	GlobalTable     *SymbolTable
	CurrentTable    *SymbolTable
	CurrentRetTypes []string
	Errors          []string
}

// Sympol
type Symbol struct {
	Name        string
	Type        aster.Type
	ScopeID     string
	Kind        string        // "var", "func", "struct"
	Params      []aster.Param // For functions
	ReturnTypes []aster.Type  // Changed: Slice of types for multiple returns
	Fields      []aster.Field // For structs
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
		return "error"
	}

	// 3. Determine result type
	switch expr.Op {
	case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
		return "bool" // Logical ops always return bool

	default:
		return leftType // Arithmetic ops return the same type
	}
}

func (tc *TypeChecker) inferType(expr aster.Expression) string {
	switch e := expr.(type) {
	// ... (Existing cases)

	case aster.CallExpr:
		types := tc.inferReturnTypes(e)
		if len(types) > 1 {
			tc.Errors = append(tc.Errors, "multiple-value context error")
			return "error"
		}
		if len(types) == 0 {
			return "void" // Or handle functions with no return
		}
		return types[0]

	default:
		return "unknown"
	}
}

func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) []string {
	if call, ok := expr.(aster.CallExpr); ok {
		funcSym, exists := tc.GlobalTable.Resolve(call.Callee.(aster.IdentExpr).Name)
		if !exists {
			tc.Errors = append(tc.Errors, fmt.Sprintf("undefined function: %s", call.Callee))
			return []string{"error"}
		}

		// Return all types from the slice we just updated in Symbol
		var types []string
		for _, t := range funcSym.ReturnTypes {
			types = append(types, t.Name)
		}
		return types
	}

	// For non-call expressions, it's always a single type
	return []string{tc.inferType(expr)}
}
func (tc *TypeChecker) checkVarDeclar(decl *aster.VarDeclar) {
	var finalType string

	// 1. If type is explicit (var i int = 10)
	if decl.Type != nil {
		finalType = decl.Type.Name

		// If there is an assignment, check type compatibility
		if decl.Value != nil {
			valueType := tc.inferType(decl.Value)
			if finalType != valueType {
				tc.Errors = append(tc.Errors, fmt.Sprintf("cannot use %s as %s in assignment", valueType, finalType))
			}
		}
	} else {
		// 2. Type inference (var i = 10)
		if decl.Value == nil {
			tc.Errors = append(tc.Errors, fmt.Sprintf("variable %s: missing type or expression", decl.Name))
			return
		}
		finalType = tc.inferType(decl.Value)
	}

	// 3. Register the symbol in the current table
	sym := &Symbol{
		Name:    decl.Name,
		Type:    aster.Type{Name: finalType}, // Simplified for now
		ScopeID: tc.CurrentTable.ScopeID,
	}

	err := tc.CurrentTable.Define(decl.Name, sym)
	if err != nil {
		tc.Errors = append(tc.Errors, err.Error())
	}
}

func (tc *TypeChecker) checkFieldAccess(expr aster.FieldAccessExpr) string {
	objType := tc.inferType(expr.Object)
	if objType == "error" {
		return "error"
	}

	// Auto-dereference: if type is "*User", look up "User"
	actualTypeName := objType
	if len(objType) > 0 && objType[0] == '*' {
		actualTypeName = objType[1:]
	}

	structSym, exists := tc.GlobalTable.Resolve(actualTypeName)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("type %s is not defined", actualTypeName))
		return "error"
	}

	// Check fields within the resolved struct symbol
	for _, field := range structSym.Fields {
		if field.Name == expr.Field {
			return field.Type // Found the field, return its type
		}
	}

	tc.Errors = append(tc.Errors, fmt.Sprintf("type %s has no field %s", actualTypeName, expr.Field))
	return "error"
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	// 1. Create a new child SymbolTable for the function
	childScopeID := tc.CurrentTable.generateChildID()
	childTable := &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  tc.CurrentTable,
		ScopeID: childScopeID,
	}

	// 2. Switch context to the new function scope
	previousTable := tc.CurrentTable
	tc.CurrentTable = childTable

	// 3. Define parameters in the new scope
	// Parameters are the first inhabitants of the function frame
	for _, param := range fn.Params {
		sym := &Symbol{
			Name:    param.Name,
			Type:    param.Type,
			ScopeID: childScopeID,
		}
		err := tc.CurrentTable.Define(param.Name, sym)
		if err != nil {
			tc.Errors = append(tc.Errors, err.Error())
		}
	}

	// 4. Check function body (The FrameBlock)
	// We will use checkBlock to process the statements inside
	if fn.Body != nil {
		tc.checkBlock(fn.Body)
	}

	// 5. Restore context back to the parent scope
	tc.CurrentTable = previousTable
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

	for _, stmt := range block.Stmts {
		switch s := stmt.(type) {
		case *aster.VarDeclar:
			tc.checkVarDeclar(s)

		case aster.Declar: // For := operator
			tc.checkDeclar(s)

		case aster.Assign: // For = operator
			tc.checkAssign(s)

		case aster.ExprStmt:
			tc.inferType(s.Expr)

		// Add other statement types as needed
		default:
			// For now, skip unknown statements
		}
	}
}

func (tc *TypeChecker) checkDeclar(decl aster.Declar) {
	// 1. := always requires an expression on the right
	if decl.Value == nil {
		tc.Errors = append(tc.Errors, "syntax error: := must have a value on the right")
		return
	}

	// 2. Infer type from the value (Value is mandatory here)
	finalType := tc.inferType(decl.Value)
	if finalType == "error" {
		return
	}

	// 3. Register the symbol (Assuming Name is an IdentExpr)
	varName := ""
	if ident, ok := decl.Name.(aster.IdentExpr); ok {
		varName = ident.Name
	} else {
		tc.Errors = append(tc.Errors, "invalid left-hand side in := declaration")
		return
	}

	sym := &Symbol{
		Name:    varName,
		Type:    aster.Type{Name: finalType},
		ScopeID: tc.CurrentTable.ScopeID,
	}

	// 4. Reuse Define to prevent redeclaration in the same scope
	err := tc.CurrentTable.Define(varName, sym)
	if err != nil {
		tc.Errors = append(tc.Errors, err.Error())
	}
}

func (tc *TypeChecker) checkAssign(asgn aster.Assign) {
	targetType := tc.inferType(asgn.Target)

	var rhsTypes []aster.Type
	if call, ok := asgn.Value.(aster.CallExpr); ok {
		funcSym, exists := tc.GlobalTable.Resolve(call.Callee.(aster.IdentExpr).Name)
		if exists && funcSym.Kind == "func" {
			// Direct access to the slice, no string manipulation needed
			rhsTypes = funcSym.ReturnTypes
		}
	} else {
		// Wrap single type into a slice for consistency
		singleType := tc.inferType(asgn.Value)
		rhsTypes = []aster.Type{{Name: singleType}}
	}

	// 3. Validation
	if len(rhsTypes) > 1 {
		msg := fmt.Sprintf("assignment mismatch: %d values but only 1 target", len(rhsTypes))
		tc.Errors = append(tc.Errors, msg)
		return
	}

	if len(rhsTypes) == 1 {
		currentRhs := rhsTypes[0].Name
		if targetType != "error" && currentRhs != "error" && targetType != currentRhs {
			msg := fmt.Sprintf("cannot assign %s to %s", currentRhs, targetType)
			tc.Errors = append(tc.Errors, msg)
		}
	}
}

func (tc *TypeChecker) checkStructLiteral(lit aster.StructLiteral) string {
	// 1. Lookup the Struct definition in the Global Table
	structName := lit.Type.Name
	structSym, exists := tc.GlobalTable.Resolve(structName)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("undefined type: %s", structName))
		return "error"
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
			tc.Errors = append(tc.Errors, fmt.Sprintf("struct %s has no field %s", structName, providedField.Name))
			continue
		}

		// Infer the type of the value being assigned to the field
		providedType := tc.inferType(providedField.Value)

		// Strict Check: Ensure types match exactly
		if providedType != expectedType {
			msg := fmt.Sprintf("type mismatch in %s.%s: expected %s, got %s",
				structName, providedField.Name, expectedType, providedType)
			tc.Errors = append(tc.Errors, msg)
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

// check
func (tc *TypeChecker) Check(a *aster.AST) {
	// 1. Register Structs first (so variables can use them as types)
	for _, s := range a.Structs {
		tc.GlobalTable.Define(s.Name, &Symbol{
			Name:   s.Name,
			Fields: s.Fields,
			// ScopeID for globals is usually "0" or empty
			ScopeID: "0",
		})
	}

	// 2. Register and check Global Variables
	tc.checkGlobalVars(a.Vars)

	// 3. Register Function signatures
	for _, f := range a.Funcs {
		tc.GlobalTable.Define(f.FuncName, &Symbol{
			Name:    f.FuncName,
			Type:    aster.Type{Name: "func"},
			ScopeID: "0",
		})
	}

	// 4. Finally, check function bodies
	for _, f := range a.Funcs {
		tc.checkFuncDecl(&f)
	}
}

func (tc *TypeChecker) checkCallExpr(call *aster.CallExpr) string {
	// 1. Assert that Callee is an IdentExpr to get the Name
	callee, ok := call.Callee.(aster.IdentExpr)
	if !ok {
		tc.Errors = append(tc.Errors, "invalid call: expected a function name")
		return "error"
	}

	// 2. Resolve the function name in the SymbolTable
	sym, exists := tc.CurrentTable.Resolve(callee.Name)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("undefined function: %s", callee.Name))
		return "error"
	}

	// 3. Ensure the symbol is actually a function
	if sym.Kind != "func" {
		tc.Errors = append(tc.Errors, fmt.Sprintf("%s is not a function", callee.Name))
		return "error"
	}

	// 4. Check arguments count
	if len(call.Args) != len(sym.Params) {
		msg := fmt.Sprintf("too many or too few arguments in call to %s", callee.Name)
		tc.Errors = append(tc.Errors, msg)
		return "error"
	}

	// 5. Validate each argument type against parameter type
	for i, arg := range call.Args {
		expectedType := sym.Params[i].Type.Name
		providedType := tc.inferType(arg)

		if expectedType != providedType {
			msg := fmt.Sprintf("cannot use %s as %s in argument to %s",
				providedType, expectedType, callee.Name)
			tc.Errors = append(tc.Errors, msg)
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
		if condType != "bool" {
			msg := fmt.Sprintf("non-bool condition in for statement: got %s", condType)
			tc.Errors = append(tc.Errors, msg)
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
	case aster.Declar:
		tc.checkDeclar(s)

	// 3. Assignments (x = 20)
	case aster.Assign:
		tc.checkAssign(s)

	// 4. Expression Statements (Function calls)
	case aster.ExprStmt:
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
	if condType != "bool" {
		msg := fmt.Sprintf("non-bool condition in if statement: got %s", condType)
		tc.Errors = append(tc.Errors, msg)
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

func (tc *TypeChecker) checkReturnStmt(ret *aster.ReturnStmt) {
	// 1. Match the count of return values
	// Example: fn() int, bool must return exactly two values
	if len(ret.Results) != len(tc.CurrentRetTypes) {
		msg := fmt.Sprintf("return count mismatch: expected %d values, got %d",
			len(tc.CurrentRetTypes), len(ret.Results)) // results : values
		tc.Errors = append(tc.Errors, msg)
		return
	}

	// 2. Validate each expression type
	for i, expr := range ret.Results {
		providedType := tc.inferType(expr)
		expectedType := tc.CurrentRetTypes[i]

		// Strict Type Checking (No implicit conversion)
		if providedType != expectedType {
			msg := fmt.Sprintf("cannot use type %s as %s in return statement",
				providedType, expectedType)
			tc.Errors = append(tc.Errors, msg)
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
		return "error"
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
			return "error"
		}
		return operandType[1:]

	case "!":
		// Logical Negation: Only for boolean types
		if operandType != "bool" {
			msg := fmt.Sprintf("operator '!' not defined for type %s", operandType)
			tc.Errors = append(tc.Errors, msg)
			return "error"
		}
		return "bool"

	case "-":
		// Numeric Negation: Keep the numeric type (int/float)
		return operandType

	default:
		return operandType
	}
}
