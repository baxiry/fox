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

// Correct implementation: enforce line as the first parameter
func (tc *TypeChecker) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: %s", line, msg))
}

func (tc *TypeChecker) inferType(expr aster.Expression) string {
	switch e := expr.(type) {

	// Literals
	case aster.StringExpr:
		return "string"

	case aster.NumberExpr:
		return "int" // TODO: Handle float logic later

	case aster.BoolExpr:
		return "bool"

	case aster.CallExpr:
		types := tc.inferReturnTypes(e)
		if len(types) == 0 {
			return "void"
		}
		if len(types) > 1 {
			fmt.Println("line: ", e.Line)
			tc.appendErrorf("multiple-value call in single-value context", e.Line)
			return "error"
		}
		return types[0]

	case aster.IdentExpr:
		sym, exists := tc.CurrentTable.Resolve(e.Name)
		if !exists {
			fmt.Println("line: ", e.Line)
			tc.appendErrorf("undefined variable: %s", e.Line, e.Name)
			return "error"
		}
		return sym.Type.Name

	case aster.FieldAccessExpr:
		return tc.checkFieldAccess(e)

	case aster.StructLiteral:
		return tc.checkStructLiteral(e)

	// Binary Operations (a + b, i < 10, etc.)
	case aster.BinaryExpr:
		leftType := tc.inferType(e.Left)
		rightType := tc.inferType(e.Right)

		// 1. Comparison operators always return "bool"
		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=":
			// Ideally, check if leftType and rightType are compatible here
			return "bool"
		case "&&", "||":
			return "bool"
		}

		// 2. Arithmetic operators (+, -, *, /) return the type of the operands
		// For now, we assume simple type consistency (e.g., int + int = int)
		if leftType == rightType {
			return leftType
		}

		// If types are mismatched, it's an error but we return leftType to continue
		return leftType

	default:
		return "unknown"
	}
}

func (tc *TypeChecker) inferReturnTypes(expr aster.Expression) []string {
	if call, ok := expr.(aster.CallExpr); ok {
		// Use the already existing checkCallExpr logic
		// This will populate tc.Errors if types mismatch
		_ = tc.checkCallExpr(&call)

		// Now fetch the symbol to return the types
		callee, _ := call.Callee.(aster.IdentExpr)
		sym, exists := tc.CurrentTable.Resolve(callee.Name)
		if !exists {
			return []string{"error"}
		}

		var names []string
		for _, t := range sym.ReturnTypes {
			names = append(names, t.Name)
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

		// If there is an assignment, check type compatibility
		if decl.Value != nil {
			valueType := tc.inferType(decl.Value)
			if finalType != valueType {
				fmt.Println("LINE: ", decl.Line)
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
		Type:    aster.Type{Name: finalType}, // Simplified for now
		ScopeID: tc.CurrentTable.ScopeID,
	}

	err := tc.CurrentTable.Define(decl.Name, sym)
	if err != nil {
		tc.appendErrorf("", decl.Line, err.Error())
	}
}

func (tc *TypeChecker) checkFieldAccess(expr aster.FieldAccessExpr) string {
	objType := tc.inferType(expr.Object)
	if objType == "error" || objType == "unknown" {
		return "error"
	}

	actualTypeName := objType
	if len(objType) > 0 && objType[0] == '*' {
		actualTypeName = objType[1:]
	}

	structSym, exists := tc.GlobalTable.Resolve(actualTypeName)
	if !exists {
		tc.Errors = append(tc.Errors, fmt.Sprintf("type %s is not defined", actualTypeName))
		return "error"
	}

	if structSym.Kind != "struct" {
		tc.Errors = append(tc.Errors, fmt.Sprintf("%s is not a struct", actualTypeName))
		return "error"
	}

	for _, field := range structSym.Fields {
		if field.Name == expr.Field {
			return field.Name
		}
	}

	tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: type %s has no field %s", 0, actualTypeName, expr.Field))
	return "error"
}

func (tc *TypeChecker) checkFuncDecl(fn *aster.Func) {
	tc.CurrentRetTypes = []string{}
	for _, ret := range fn.Returns {
		tc.CurrentRetTypes = append(tc.CurrentRetTypes, ret.Type.Name)
	}

	// Retrieve the function code from the global table
	sym, _ := tc.GlobalTable.Resolve(fn.FuncName)
	// Store the current func.
	tc.CurrentFunction = sym

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
		err := tc.CurrentTable.Define(param.Type.Name, sym)
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

	// reset func
	tc.CurrentFunction = nil // reset

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

		if expectedTypeName != actualTypeName {
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

func (tc *TypeChecker) checkDeclar(decl aster.Declar) {
	// 1. Basic syntax check: Ensure the right side is not empty
	if len(decl.Values) == 0 {
		tc.Errors = append(tc.Errors, "syntax error: := must have values on the right")
		return
	}

	// 2. Collect and unpack all types from the right-hand side
	var rightSideTypes []string
	for _, valExpr := range decl.Values {
		// The "..." operator flattens return types from functions or literals
		rightSideTypes = append(rightSideTypes, tc.inferReturnTypes(valExpr)...)
	}

	// 3. Structural check: Count of identifiers must match count of types
	if len(decl.Names) != len(rightSideTypes) {
		tc.Errors = append(tc.Errors, fmt.Sprintf(
			"assignment mismatch: %d names on the left but %d values on the right",
			len(decl.Names), len(rightSideTypes),
		))
		return
	}

	// 4. Register each name into the current scope
	for i, nameExpr := range decl.Names {
		ident, ok := nameExpr.(aster.IdentExpr)
		if !ok {
			tc.Errors = append(tc.Errors, "invalid identifier on the left side of :=")
			continue
		}

		varName := ident.Name

		// 5. Blank identifier "_": skip symbol table registration
		if varName == "_" {
			continue
		}

		// 6. Safety check: Don't define variables with "error" type
		inferredType := rightSideTypes[i]
		if inferredType == "error" {
			continue
		}

		// Create the symbol and add it to the table
		sym := &Symbol{
			Name:    varName,
			Type:    aster.Type{Name: inferredType},
			ScopeID: tc.CurrentTable.ScopeID,
		}

		// Define handles redeclaration checks internally
		if err := tc.CurrentTable.Define(varName, sym); err != nil {
			tc.Errors = append(tc.Errors, err.Error())
		}
	}
}

func (tc *TypeChecker) checkAssign(asgn aster.Assign) {

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
	for i, nameExpr := range asgn.Targets {
		ident, ok := nameExpr.(aster.IdentExpr)
		if !ok {
			fmt.Println(ident.Line)
			tc.appendErrorf("invalid left-hand side in assignment", ident.Line)
			continue
		}

		varName := ident.Name

		// 5. Blank identifier "_": skip validation as it can accept any type
		if varName == "_" {
			continue
		}

		// 6. Resolve variable: it must be declared before assignment (=)
		existingSym, exists := tc.CurrentTable.Resolve(varName)
		if !exists {
			fmt.Println("Line:", ident.Line)
			tc.appendErrorf("undefined variable: %s", ident.Line, varName)
			continue
		}

		// 7. Type Consistency: Ensure new value type matches the declared variable type
		inferredRhsType := rightSideTypes[i]
		if inferredRhsType != "error" && existingSym.Type.Name != inferredRhsType {
			tc.Errors = append(tc.Errors, fmt.Sprintf(
				"Line %d cannot assign %s to variable %s of type %s", asgn.Line,
				inferredRhsType, varName, existingSym.Type.Name,
			))
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
			msg := fmt.Sprintf("Line ? type mismatch in %s.%s: expected %s, got %s",
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

		// Map AST fields to Symbol fields
		for _, f := range s.Fields {
			// If your Symbol doesn't have a 'Fields' slice yet,
			// make sure to add it to your Symbol struct definition.
			sym.Fields = append(sym.Fields, aster.Field{
				Name: f.Name,
				Type: f.Type,
			})
		}

		// Add to GlobalTable so functions can resolve this type
		tc.GlobalTable.Define(s.Name, sym)
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
		tc.Errors = append(tc.Errors, fmt.Sprintf("Line ? , %s is not a function", callee.Name))
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
