package wrrap

/*
#cgo CFLAGS: -I/opt/homebrew/include
#cgo LDFLAGS: -L/opt/homebrew/lib/gcc/current -lgccjit
#cgo LDFLAGS: -L/opt/homebrew/lib/gcc/current

#include <libgccjit.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// Types

type Type struct{ ptr *C.gcc_jit_type }
type Param struct{ ptr *C.gcc_jit_param }
type Function struct{ ptr *C.gcc_jit_function }
type Block struct{ ptr *C.gcc_jit_block }
type RValue struct{ ptr *C.gcc_jit_rvalue }
type LValue struct{ ptr *C.gcc_jit_lvalue }
type Context struct{ ptr *C.gcc_jit_context }

type FunctionKind int

const (
	// 0: GCC_JIT_FUNCTION_INTERNAL
	FunctionInternal FunctionKind = iota

	// 1: GCC_JIT_FUNCTION_EXPORTED (main)
	FunctionExported

	// 2: GCC_JIT_FUNCTION_IMPORTED (printf)
	FunctionImported

	// 3: GCC_JIT_FUNCTION_ALWAYS_INLINE
	FunctionAlwaysInline
)

const (
	Assemble   = C.GCC_JIT_OUTPUT_KIND_ASSEMBLER
	Executable = C.GCC_JIT_OUTPUT_KIND_EXECUTABLE
)

const (
	Plus  BinaryOp = 0 // GCC_JIT_BINARY_OP_PLUS
	Minus BinaryOp = 1 // GCC_JIT_BINARY_OP_MINUS
	Mult  BinaryOp = 2 // GCC_JIT_BINARY_OP_MULT
	Div   BinaryOp = 3 // GCC_JIT_BINARY_OP_DIV

	//Assemble   OutputKind = 0 // GCC_JIT_OUTPUT_KIND_ASSEMBLER
	//Executable OutputKind = 1 // GCC_JIT_OUTPUT_KIND_EXECUTABLE
)

type BinaryOp = C.enum_gcc_jit_binary_op
type OutputKind = C.enum_gcc_jit_output_kind

//  Context Methods

func NewContext() *Context {
	return &Context{ptr: C.gcc_jit_context_acquire()}
}

func (ctx *Context) Release() {
	C.gcc_jit_context_release(ctx.ptr)
}

func (ctx *Context) NewIntType() Type {
	return Type{ptr: C.gcc_jit_context_get_type(ctx.ptr, C.GCC_JIT_TYPE_INT)}
}

// NewGlobal creates a new global variable in the JIT context.
func (ctx *Context) NewGlobal(t Type, name string) LValue {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	// Using GCC_JIT_GLOBAL_EXPORTED makes the variable visible to other modules if needed.
	ptr := C.gcc_jit_context_new_global(ctx.ptr, nil, C.GCC_JIT_GLOBAL_EXPORTED, t.ptr, cName)
	return LValue{ptr: ptr}
}

// check RValue are nil
func (rv RValue) IsNil() bool {
	return rv.ptr == nil
}

// check LValue are nil
func (lv LValue) IsNil() bool {
	return lv.ptr == nil
}

// AsRValue converts a location (LValue) into a value (RValue).
func (lv LValue) AsRValue() RValue {
	return RValue{ptr: C.gcc_jit_lvalue_as_rvalue(lv.ptr)}
}
func (ctx *Context) NewVoidType() Type {
	return Type{ptr: C.gcc_jit_context_get_type(ctx.ptr, C.GCC_JIT_TYPE_VOID)}
}

func (ctx *Context) NewStringType() Type {
	return Type{ptr: C.gcc_jit_context_get_type(ctx.ptr, C.GCC_JIT_TYPE_CONST_CHAR_PTR)}
}

func (ctx *Context) NewParam(t Type, name string) Param {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return Param{ptr: C.gcc_jit_context_new_param(ctx.ptr, nil, t.ptr, cName)}
}

func (ctx *Context) NewFunction(name string, retType Type, isVariadic bool, fkind FunctionKind, params ...Param) Function {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cParams []*C.gcc_jit_param
	for _, p := range params {
		cParams = append(cParams, p.ptr)
	}

	var pPtr **C.gcc_jit_param
	if len(cParams) > 0 {
		pPtr = (**C.gcc_jit_param)(unsafe.Pointer(&cParams[0]))
	}

	v := 0
	if isVariadic {
		v = 1
	}

	f := C.gcc_jit_context_new_function(ctx.ptr, nil,
		C.enum_gcc_jit_function_kind(fkind),
		retType.ptr,
		cName,
		C.int(len(cParams)),
		pPtr,
		C.int(v))

	return Function{ptr: f}
}

func (ctx *Context) AddTopLevelAsm(asm string) {
	cAsm := C.CString(asm)
	defer C.free(unsafe.Pointer(cAsm))
	C.gcc_jit_context_add_top_level_asm(ctx.ptr, nil, cAsm)
}

func (ctx *Context) NewIntConstant(t Type, val int) RValue {
	return RValue{ptr: C.gcc_jit_context_new_rvalue_from_int(ctx.ptr, t.ptr, C.int(val))}
}

func (ctx *Context) NewStringConstant(str string) RValue {
	cStr := C.CString(str)
	defer C.free(unsafe.Pointer(cStr))
	return RValue{ptr: C.gcc_jit_context_new_string_literal(ctx.ptr, cStr)}
}

func (ctx *Context) NewBinaryOp(op C.enum_gcc_jit_binary_op, t Type, a, b RValue) RValue {
	return RValue{ptr: C.gcc_jit_context_new_binary_op(ctx.ptr, nil, op, t.ptr, a.ptr, b.ptr)}
}

func (ctx *Context) NewCall(f Function, args ...any) RValue {
	// Simplification for the initial experiment: We assume that args is RValue // TO_DELETE

	var cArgs []*C.gcc_jit_rvalue
	for _, arg := range args {
		if rv, ok := arg.(RValue); ok {
			cArgs = append(cArgs, rv.ptr)
		} else if lv, ok := arg.(LValue); ok {
			// تحويل LValue إلى RValue تلقائياً
			cArgs = append(cArgs, C.gcc_jit_lvalue_as_rvalue(lv.ptr))
		}
	}

	var aPtr **C.gcc_jit_rvalue
	if len(cArgs) > 0 {
		aPtr = (**C.gcc_jit_rvalue)(unsafe.Pointer(&cArgs[0]))
	}

	rv := C.gcc_jit_context_new_call(ctx.ptr, nil, f.ptr, C.int(len(cArgs)), aPtr)
	return RValue{ptr: rv}
}

func (ctx *Context) CompileToFile(kind C.enum_gcc_jit_output_kind, filename string) {
	cName := C.CString(filename)
	defer C.free(unsafe.Pointer(cName))
	C.gcc_jit_context_compile_to_file(ctx.ptr, kind, cName)
}

//  Function & Block Methods

func (f Function) NewBlock(name string) Block {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return Block{ptr: C.gcc_jit_function_new_block(f.ptr, cName)}
}

func (f Function) NewLocal(t Type, name string) LValue {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	return LValue{ptr: C.gcc_jit_function_new_local(f.ptr, nil, t.ptr, cName)}
}

func (b Block) AddAssignment(lv LValue, rv RValue) {
	C.gcc_jit_block_add_assignment(b.ptr, nil, lv.ptr, rv.ptr)
}

func (b Block) AddEval(rv RValue) {
	C.gcc_jit_block_add_eval(b.ptr, nil, rv.ptr)
}

func (b Block) EndWithReturn(rv RValue) {
	C.gcc_jit_block_end_with_return(b.ptr, nil, rv.ptr)
}

func (ctx *Context) AddDriverOption(option string) {
	cOption := C.CString(option)
	defer C.free(unsafe.Pointer(cOption))
	C.gcc_jit_context_add_driver_option(ctx.ptr, cOption)
}
