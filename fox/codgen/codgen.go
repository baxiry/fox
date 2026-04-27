package codgen

import "fox/wrrap"

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
	printfFunc := ctx.NewFunction("printf", intType, true, printfParam) // true تعني variadic

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
