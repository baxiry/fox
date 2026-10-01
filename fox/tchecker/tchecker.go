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
