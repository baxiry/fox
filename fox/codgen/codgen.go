package codgen

import (
	"fox/aster"
	"fox/tchecker"
	"fox/wrrap"
)

type Codegen struct {
	ctx        *wrrap.Context
	tc         *tchecker.TypeChecker
	project    *aster.Project // Changed from Package to Project
	printfFunc wrrap.Function // Pre-defined printf function
}

// NewCodegen initializes the environment for code generation
func NewCodegen(proj *aster.Project) *Codegen {
	ctx := wrrap.NewContext()

	// Add the macOS workarounds we discussed earlier
	ctx.AddDriverOption("-L/opt/homebrew/lib/gcc/current")
	ctx.AddDriverOption("-B/opt/homebrew/lib/gcc/current/gcc/aarch64-apple-darwin24/15")
	//ctx.AddDriverOption("-lemutls_w")

	cg := &Codegen{
		ctx:     ctx,
		project: proj,
	}

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

// genExpr evaluates an AST expression and returns its JIT RValue
func (cg *Codegen) genExpr(expr aster.Expression) wrrap.RValue {
	switch e := expr.(type) {
	case *aster.NumberExpr:
		t := cg.ctx.NewIntType()
		return cg.ctx.NewIntConstant(t, e.Value)

	case *aster.CallExpr:
		return cg.generateCall(e)
	}
	return wrrap.RValue{}
}

func (cg *Codegen) generateCall(e *aster.CallExpr) wrrap.RValue {
	// For now, let's assume all calls are to built-ins like printf
	callee := e.Callee.(aster.IdentExpr).Name

	if callee == "printf" {
		var args []any
		for _, arg := range e.Args {
			args = append(args, cg.genExpr(arg))
		}
		return cg.ctx.NewCall(cg.printfFunc, args...)
	}

	return wrrap.RValue{}
}

func (cg *Codegen) genCallExpr(expr *aster.CallExpr) wrrap.RValue {
	// 1. Get the callee name
	callee, _ := expr.Callee.(aster.IdentExpr)

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
		rValues = append(rValues, cg.genExpr(arg))
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

// Generate iterates through the AST and produces machine code.
func (cg *Codegen) Generate() {
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
