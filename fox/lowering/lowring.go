package lowering

import (
	"fox/aster"
)

type Desugarer struct {
	ast *aster.AST
}

func NewDesugarer(ast *aster.AST) *Desugarer {
	return &Desugarer{ast: ast}
}

func (d *Desugarer) Run() *aster.AST {
	return d.ast
}

func (d *Desugarer) lowerFunctionReturn(fn *aster.Func) *aster.Struct {
	_ = fn
	return &aster.Struct{}
}
