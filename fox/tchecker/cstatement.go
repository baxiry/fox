package tchecker

import (
	"fox/aster"
	"fox/symbols"
	"strings"
)

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

func (tc *TypeChecker) checkMatchStmt(s *aster.MatchStmt) {
	if s == nil || s.Object == nil {
		return
	}

	objType := tc.inferType(s.Object)
	if objType == nil {
		return
	}

	isErrorEnvelope := strings.HasPrefix(objType.Name, "_Result_")

	for _, c := range s.Cases {
		isErrorCase := false
		if len(c.Conditions) > 0 {
			if ident, ok := c.Conditions[0].(*aster.IdentExpr); ok && ident.Name == "Error" {
				isErrorCase = true
			}
		}

		if isErrorEnvelope {
			if isErrorCase {
				originalType := objType.Name
				objType.Name = "Error"

				if c.Body != nil {
					tc.checkBlock(c.Body)
				}

				objType.Name = originalType
				continue
			} else {
				originalType := objType.Name
				objType.Name = strings.TrimPrefix(originalType, "_Result_")

				if c.Body != nil {
					tc.checkBlock(c.Body)
				}

				objType.Name = originalType
				continue
			}
		}

		if c.Body != nil {
			tc.checkBlock(c.Body)
		}
	}

	if s.Else != nil {
		if isErrorEnvelope {
			originalType := objType.Name
			objType.Name = strings.TrimPrefix(originalType, "_Result_")

			tc.checkBlock(s.Else)

			objType.Name = originalType
		} else {
			tc.checkBlock(s.Else)
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

func (tc *TypeChecker) checkReturnStmt(stmt *aster.ReturnStmt) {
	if tc.CurrFn == nil {
		tc.appendErrorf("return statement outside function", stmt.Line)
		return
	}

	retSig := tc.CurrentRetTypes
	if retSig == nil && tc.CurrFn.RetSig != nil {
		retSig = tc.CurrFn.RetSig
	}

	valuesCount := len(stmt.Results)

	// 1. Void function handling
	if retSig == nil || len(retSig.Fields) == 0 {
		if valuesCount > 0 {
			tc.appendErrorf("too many arguments to return: expected 0, got %d", stmt.Line, valuesCount)
		}
		return
	}

	expectedFields := retSig.Fields
	expectedCount := len(expectedFields)

	// 2. Empty return handling (naked return)
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

	// Ensure all returned expressions are fully visited and checked
	for _, actualExpr := range stmt.Results {
		if actualExpr != nil {
			tc.checkExpr(actualExpr)
		}
	}

	// 3. Single returned expression in multi-return function (shorthand expansion)
	if valuesCount == 1 {
		actualExpr := stmt.Results[0]
		actualType := tc.inferType(actualExpr)
		if actualType == nil || actualType.Name == aster.INVALID.String() {
			return
		}

		// Handle error value return shortcut
		if (actualType.Name == "error" || actualType.Name == "Error") && retSig.HasError {
			tc.expandReturnWithZeros(stmt, expectedFields, actualExpr, true)
			return
		}

		// Handle first field type matching shortcut
		if expectedCount > 1 && actualType.IsSameAs(expectedFields[0].Type) {
			tc.expandReturnWithZeros(stmt, expectedFields, actualExpr, false)
			return
		}

		// Single expected return value matching
		if expectedCount == 1 {
			tc.checkTypeMatch(expectedFields[0].Type, actualType, stmt.Line)
			return
		}
	}

	// 4. Argument count matching for multi-return
	if valuesCount > expectedCount {
		tc.appendErrorf("too many arguments to return: expected %d, got %d", stmt.Line, expectedCount, valuesCount)
		return
	}

	if valuesCount < expectedCount {
		tc.appendErrorf("not enough arguments to return: expected %d, got %d", stmt.Line, expectedCount, valuesCount)
		return
	}

	// 5. Explicit 1:1 type matching for each returned expression
	for i, actualExpr := range stmt.Results {
		actualType := tc.inferType(actualExpr)
		if actualType == nil || actualType.Name == aster.INVALID.String() {
			continue
		}
		expectedType := expectedFields[i].Type
		tc.checkTypeMatch(expectedType, actualType, stmt.Line)
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
