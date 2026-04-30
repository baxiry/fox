package codgen

import (
	"fmt"
	"fox/aster"
	"fox/tchecker"
	"fox/wrrap"
	"strings"
)

type Codegen struct {
	ctx        *wrrap.Context
	unit       *aster.AST
	tc         *tchecker.TypeChecker
	project    *aster.Project // Changed from Package to Project
	printfFunc wrrap.Function
	// Stores global variables: key is name, value is the JIT LValue
	globals map[string]wrrap.LValue
	funcs   map[string]wrrap.Function
}

func (cg *Codegen) genFunctionBody(f *aster.Func) {
	fn, exists := cg.funcs[f.FuncName]
	if !exists {
		return
	}

	intType := cg.ctx.NewIntType()

	block := fn.NewBlock("entry")

	if f.Body == nil {
		fmt.Println("DEBUG: Function body is nil!")
		return
	}

	fmt.Printf("DEBUG: Processing %d statements in %s\n", len(f.Body.Stmts), f.FuncName)
	for _, stmt := range f.Body.Stmts {
		cg.genStmt(&block, stmt)
	}

	block.EndWithReturn(cg.ctx.NewIntConstant(intType, 0))
}

// genAssign generates code for variable assignments.
func (cg *Codegen) genAssign(block *wrrap.Block, stmt *aster.Assign) {
	// For hello_world, we assume a single target like: msg = "value"
	if len(stmt.Targets) > 0 && len(stmt.Values) > 0 {
		// Correctly identifying the target (LValue)
		targetIdent, ok := stmt.Targets[0].(*aster.IdentExpr)
		if !ok {
			return
		}

		target, exists := cg.globals[targetIdent.Name]
		if !exists {

			fmt.Printf("DEBUG: Codegen Error: Global variable %s not found!\n", targetIdent.Name)
			return
		}

		// FIX: Passing 'block' as the first argument to genExpr
		val := cg.genExpr(block, stmt.Values[0])

		// Perform the JIT assignment
		block.AddAssignment(target, val)
	}
}

// genStmt routes aster.Statement to specific generator functions.
func (cg *Codegen) genStmt(block *wrrap.Block, stmt aster.Statement) {
	switch s := stmt.(type) {
	case *aster.Assign:
		cg.genAssign(block, s)

	// case *aster.ExprStmt: cg.genExpr(block, s.Expr)
	case *aster.ExprStmt:
		val := cg.genExpr(block, s.Expr)
		block.AddEval(val)

	}
}

// genExpr now accepts the block and handles IdentExpr for globals.
func (cg *Codegen) genExpr(block *wrrap.Block, expr aster.Expression) wrrap.RValue {
	switch e := expr.(type) {

	case *aster.StringExpr:
		text := e.Literal

		if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
			text = text[1 : len(text)-1]
		}

		text = strings.ReplaceAll(text, "\\n", "\n")
		text = strings.ReplaceAll(text, "\\t", "\t")
		return cg.ctx.NewStringConstant(text + "\x00")

	case *aster.IdentExpr:
		if lv, exists := cg.globals[e.Name]; exists {
			return lv.AsRValue()
		}

	case *aster.CallExpr:
		return cg.generateCall(block, e)
	}

	return wrrap.RValue{}
}

// genFunctionSignature declares the function's name and return type.
func (cg *Codegen) genFunctionSignature(f *aster.Func) {
	// For simplicity in hello_world, we assume return type is int for main
	retType := cg.ctx.NewIntType()

	// Create the function signature without the body
	fn := cg.ctx.NewFunction(f.FuncName, retType, false)
	cg.funcs[f.FuncName] = fn

}

// genGlobalVar allocates memory for global variables in the JIT context.
func (cg *Codegen) genGlobalVar(decl *aster.VarDeclar) {
	var jitType wrrap.Type

	// Basic type mapping for now
	switch decl.Type.Name {
	case "string":
		jitType = cg.ctx.NewStringType()
	case "int":
		jitType = cg.ctx.NewIntType()
	default:
		jitType = cg.ctx.NewIntType() // Fallback
	}

	// Create and store the global variable
	name := decl.Name
	cg.globals[name] = cg.ctx.NewGlobal(jitType, name)
}

// NewCodegen initializes the environment for code generation
func NewCodegen(proj *aster.Project) *Codegen {
	ctx := wrrap.NewContext()

	// Add the macOS workarounds we discussed earlier
	ctx.AddDriverOption("-L/opt/homebrew/lib/gcc/current")
	ctx.AddDriverOption("-B/opt/homebrew/lib/gcc/current/gcc/aarch64-apple-darwin24/15")
	//ctx.AddDriverOption("-lemutls_w")

	var firstUnit *aster.AST
	if len(proj.Packages) > 0 && len(proj.Packages[0].Files) > 0 {
		firstUnit = &aster.AST{
			Decls: proj.Packages[0].Files[0].Decls,
		}
	}

	cg := &Codegen{
		ctx:     ctx,
		project: proj,
		unit:    firstUnit,
	}

	cg.globals = make(map[string]wrrap.LValue)
	cg.funcs = make(map[string]wrrap.Function)
	cg.setupBuiltIns()
	return cg
}

func (cg *Codegen) setupBuiltIns() {
	// Define the types needed for printf signature
	intType := cg.ctx.NewIntType()
	charPtr := cg.ctx.NewStringType()

	// Define printf(char* format, ...)
	formatParam := cg.ctx.NewParam(charPtr, "format")
	cg.printfFunc = cg.ctx.NewFunction("printf", intType, true, formatParam)
}

// generateCall handles both built-in and user-defined function calls.
func (cg *Codegen) generateCall(block *wrrap.Block, e *aster.CallExpr) wrrap.RValue {
	var name string
	if ident, ok := e.Callee.(*aster.IdentExpr); ok {
		name = ident.Name
	}

	if name == "printf" {
		var args []interface{}
		for _, arg := range e.Args {
			args = append(args, cg.genExpr(block, arg))
		}
		return cg.ctx.NewCall(cg.printfFunc, args...)
	}

	sym, exists := cg.tc.GlobalTable.Resolve(name)
	if exists && !sym.IsBuiltIn {
	}

	return wrrap.RValue{}
}

func (cg *Codegen) genCallExpr(expr *aster.CallExpr) wrrap.RValue {
	// 1. Get the callee name
	callee, _ := expr.Callee.(*aster.IdentExpr)

	// 2. Lookup the symbol to check if it's a Built-in
	sym, _ := cg.tc.GlobalTable.Resolve(callee.Name)

	if sym.IsBuiltIn {
		// Handle special built-in logic
		return cg.genBuiltInCall(callee.Name, expr.Args)
	}

	// 3. Normal function call logic (for user-defined functions)
	// ... logic for wrrap.NewCall using the compiled function pointer
	return wrrap.RValue{} // This fixes the missing return error
}

func (cg *Codegen) genBuiltInCall(name string, args []aster.Expression) wrrap.RValue {
	// Convert Fox arguments to wrrap.RValues
	var rValues []any
	for _, arg := range args {
		rValues = append(rValues, cg.genExpr(nil, arg))
	}

	switch name {
	case "printf":
		// Use the printf wrapper we defined in wrrap/wrrap.go
		// We assume printfFunc was previously initialized in our Context
		return cg.ctx.NewCall(cg.printfFunc, rValues...)

		// Add other built-ins like println or panic here
	}

	return wrrap.RValue{}
}

func (cg *Codegen) Generate() {
	if cg.unit == nil {
		fmt.Println("CRITICAL: cg.unit is nil, skipping generation")
		return
	}

	for _, decl := range cg.unit.Decls {
		if f, ok := decl.(*aster.Func); ok {
			fmt.Printf("DEBUG: Found function decl: %s\n", f.FuncName)
			cg.genFunctionSignature(f)
		}
	}

	for _, decl := range cg.unit.Decls {
		if f, ok := decl.(*aster.Func); ok {
			fmt.Printf("DEBUG: Generating body for: %s\n", f.FuncName)
			cg.genFunctionBody(f)
		}
	}

	cg.ctx.CompileToFile(wrrap.Assemble, "output.s")
	cg.ctx.CompileToFile(wrrap.Executable, "output")
}

/*
func GenerateCode() {
	ctx := wrrap.NewContext()
	defer ctx.Release()

	// The golden path we discovered
	const gccInternalPath = "/opt/homebrew/lib/gcc/current/gcc/aarch64-apple-darwin24/15"

	// 1. Tell GCC where to find its internal sub-programs (like collect2)
	ctx.AddDriverOption("-B" + gccInternalPath)

	// 2. Add the directory to the library search path for the linker
	ctx.AddDriverOption("-L" + gccInternalPath)

	intType := ctx.NewIntType()
	voidType := ctx.NewVoidType()
	_ = voidType

	stringType := ctx.NewStringType() // char*

	// printf(char* format, ...)
	printfParam := ctx.NewParam(stringType, "format")
	printfFunc := ctx.NewFunction("printf", intType, true, printfParam)

	// Defining the main function

	mainFunc := ctx.NewFunction("main", intType, false)

	block := mainFunc.NewBlock("entry")

	// Calculation: x := 10 + 20

	const10 := ctx.NewIntConstant(intType, 10)

	const20 := ctx.NewIntConstant(intType, 20)

	sum := ctx.NewBinaryOp(wrrap.Plus, intType, const10, const20)

	// Defining a local variable x and storing the result in it
	localX := mainFunc.NewLocal(intType, "x")

	block.AddAssignment(localX, sum)

	// Calling printf to print the result
	fmtStr := ctx.NewStringConstant("%d\n")

	block.AddEval(ctx.NewCall(printfFunc, fmtStr, localX))

	// End Function
	block.EndWithReturn(ctx.NewIntConstant(intType, 0))

	ctx.CompileToFile(wrrap.Assemble, "out.s")

	ctx.CompileToFile(wrrap.Executable, "hello_fox")
}
*/
