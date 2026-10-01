package tchecker

import (
	"fox/aster"
	"fox/symbols"
)

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

	// Handle functions with no return type
	if sym.Type == nil {
		return &symbols.Type{Name: "void"}
	}

	// Return the function's registered return type (handles both single and multi-return types)
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

// checkExpr visits, decorates nodes with their resolved types, and validates individual expressions.
func (tc *TypeChecker) checkExpr(expr aster.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {

	case *aster.CallExpr:
		// Traverse and validate arguments
		for _, arg := range e.Args {
			tc.checkExpr(arg)
		}

		resolvedType := tc.checkCallExpr(e)
		if calleeIdent, ok := e.Callee.(*aster.IdentExpr); ok {
			calleeIdent.Type = resolvedType
		}

	case *aster.IdentExpr:
		if tc.CurrentTable != nil {
			sym, exists := tc.CurrentTable.Resolve(e.Name)
			if !exists {
				tc.appendErrorf("undefined identifier: %s", e.Line, e.Name)
				e.Type = newType(aster.INVALID.String(), 0, false)
			} else if sym.Type != nil {
				e.Type = &symbols.Type{
					Name:     sym.Type.Name,
					PtrDepth: sym.Type.PtrDepth,
					IsArray:  sym.Type.IsArray,
					Size:     sym.Type.Size,
				}
			}
		}

	case *aster.IndexExpr:
		tc.checkExpr(e.Target)
		targetType := tc.inferType(e.Target)
		if targetType != nil && !targetType.IsArray {
			tc.appendErrorf("cannot index into non-array type", e.Line)
		}

	case *aster.BinaryExpr:
		tc.checkExpr(e.Left)
		tc.checkExpr(e.Right)

		leftType := tc.inferType(e.Left)
		rightType := tc.inferType(e.Right)

		if leftType != nil && rightType != nil &&
			leftType.Name != aster.INVALID.String() && rightType.Name != aster.INVALID.String() {

			switch e.Op {
			case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
				// Comparison/Logical expressions always valid if operands are valid
			default:
				if leftType.Name != rightType.Name || leftType.IsArray != rightType.IsArray {
					tc.appendErrorf("type mismatch: %s and %s", e.Line, leftType.Name, rightType.Name)
				}
			}
		}

	case *aster.UnaryExpr:
		tc.checkUnaryExpr(e)

	case *aster.BoolExpr, *aster.IntExpr, *aster.StringExpr:
		// Literals require type inference attachment
		tc.inferType(e)

	default:
		tc.inferType(expr)
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
