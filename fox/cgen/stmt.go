package cgen

/*
func (cg *Codegen) genStmt(stmt aster.Statement) {
	switch s := stmt.(type) {
	case *aster.IfStmt:
		cg.genIfStmt(s)
	case *aster.MatchStmt:
		cg.genMatchStmt(s)
	case *aster.Declar:
		cg.genDeclarStmt(s)
	case *aster.VarDeclar:
		cg.genVarDeclarStmt(s)
	case *aster.Assign:
		cg.genAssignStmt(s)
	case *aster.ExprStmt:
		cg.genExprStmt(s)
	case *aster.ReturnStmt:
		cg.genReturnStmt(s)
	case *aster.ForStmt:
		cg.genForStmt(s)
	}
}

func (cg *Codegen) genIfStmt(s *aster.IfStmt) {
	cg.writeIndent()
	cg.builder.WriteString("if (")
	cg.genExpr(s.Cond)
	cg.builder.WriteString(") ")

	savedFunc := cg.CurrentFunction

	if s.Then != nil {
		cg.genBlock(s.Then)
	}

	cg.CurrentFunction = savedFunc

	if s.Else != nil {
		cg.builder.WriteString(" else ")
		if block, ok := s.Else.(*aster.FrameBlock); ok {
			cg.genBlock(block)
		} else {
			cg.writeIndent()
			cg.builder.WriteString("{\n")
			cg.indent++
			cg.writeIndent()
			cg.genStmt(s.Else)
			cg.indent--
			cg.writeIndent()
			cg.builder.WriteString("}\n")
		}
	} else {
		cg.builder.WriteString("\n")
	}

	cg.CurrentFunction = savedFunc
}

func (cg *Codegen) genMatchStmt(s *aster.MatchStmt) {
	isErrorEnvelope := false
	var objectIdentName string
	if ident, ok := s.Object.(*aster.IdentExpr); ok && ident.Type != nil {
		objectIdentName = ident.Name
		if strings.HasPrefix(ident.Type.Name, "_Result_") {
			isErrorEnvelope = true
		}
	}

	cg.writeIndent()
	if isErrorEnvelope {
		cg.builder.WriteString("switch (")
		cg.genExpr(s.Object)
		cg.builder.WriteString(".header.error_flag) {\n")
	} else {
		cg.builder.WriteString("switch (")
		cg.genExpr(s.Object)
		cg.builder.WriteString("->_tag) {\n")
	}

	for _, c := range s.Cases {
		tagValue := 0
		isErrorCase := false

		if len(c.Conditions) > 0 {
			if ident, ok := c.Conditions[0].(*aster.IdentExpr); ok {
				if isErrorEnvelope {
					if ident.Name == "Error" {
						isErrorCase = true
						tagValue = 1
					}
				} else {
					switch ident.Name {
					case "Active":
						tagValue = 1
					case "Inactive":
						tagValue = 2
					}
				}
			}
		}

		if isErrorEnvelope && !isErrorCase {
			tagValue = 0
		}

		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "case %d:\n", tagValue)
		cg.indent++

		if c.Body != nil {
			numStmts := len(c.Body.Stmts)
			for idx, subStmt := range c.Body.Stmts {
				if idx == numStmts-1 {
					if exprStmt, ok := subStmt.(*aster.ExprStmt); ok && exprStmt.Expr != nil {
						if call, ok := exprStmt.Expr.(*aster.CallExpr); ok {
							cg.writeIndent()

							isPrintfCall := false
							if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Name == "printf" {
								isPrintfCall = true
							}

							if isErrorEnvelope && !isErrorCase && isPrintfCall {
								fmt.Fprintf(&cg.builder, "printf(")
								if len(call.Args) > 0 {
									cg.genExpr(call.Args[0])
									for k := 1; k < len(call.Args); k++ {
										cg.builder.WriteString(", ")
										if argIdent, ok := call.Args[k].(*aster.IdentExpr); ok && argIdent.Name == objectIdentName {
											fmt.Fprintf(&cg.builder, "%s.value.success", objectIdentName)
										} else {
											cg.genExpr(call.Args[k])
										}
									}
								}
								cg.builder.WriteString("); break;\n")
								continue
							}

							cg.genCall(call)
							cg.builder.WriteString("; break;\n")
							continue
						}
					}
					cg.genStmt(subStmt)
				} else {
					cg.genStmt(subStmt)
				}
			}
		}

		cg.indent--
	}

	if s.Else != nil {
		cg.writeIndent()
		cg.builder.WriteString("default:\n")
		cg.indent++

		for _, subStmt := range s.Else.Stmts {
			if exprStmt, ok := subStmt.(*aster.ExprStmt); ok && exprStmt.Expr != nil {
				if call, ok := exprStmt.Expr.(*aster.CallExpr); ok {
					isPrintfCall := false
					if callIdent, ok := call.Callee.(*aster.IdentExpr); ok && callIdent.Name == "printf" {
						isPrintfCall = true
					}

					if isErrorEnvelope && isPrintfCall {
						cg.writeIndent()
						fmt.Fprintf(&cg.builder, "printf(")
						if len(call.Args) > 0 {
							cg.genExpr(call.Args[0])
							for k := 1; k < len(call.Args); k++ {
								cg.builder.WriteString(", ")
								if argIdent, ok := call.Args[k].(*aster.IdentExpr); ok && argIdent.Name == objectIdentName {
									fmt.Fprintf(&cg.builder, "%s.value.success", objectIdentName)
								} else {
									cg.genExpr(call.Args[k])
								}
							}
						}
						cg.builder.WriteString("); break;\n")
						continue
					}
				}
			}
			cg.writeIndent()
			cg.genStmt(subStmt)
		}

		cg.indent--
	}

	cg.writeIndent()
	cg.builder.WriteString("}\n")
}

func (cg *Codegen) genDeclarStmt(s *aster.Declar) {
	cg.writeIndent()
	if ident, ok := s.Name.(*aster.IdentExpr); ok {
		typeName := "int32_t"

		// 1. Prioritize type set by TypeChecker / AST resolution
		if ident.Type != nil && ident.Type.Name != "" && ident.Type.Name != "INVALID" && ident.Type.Name != "invalid" {
			typeName = cg.mapType(ident.Type)
		} else if s.Value != nil {
			// Fallback checks
			if call, ok := s.Value.(*aster.CallExpr); ok {
				if callIdent, ok := call.Callee.(*aster.IdentExpr); ok {
					// ابحث عن الدالة في الـ AST لتعرف ما إذا كانت ترجع أكثر من حقل
					targetFunc := cg.findFunc(callIdent.Name)
					if targetFunc != nil && targetFunc.Return != nil && len(targetFunc.Return.Fields) > 1 {
						typeName = cg.getEnvelopeName(targetFunc.FuncName)
					} else if callIdent.Type != nil {
						typeName = cg.mapType(callIdent.Type)
					}
				}
			} else if lit, ok := s.Value.(*aster.StructLiteral); ok && lit.Type != nil {
				typeName = lit.Type.Name
			}
		}

		fmt.Fprintf(&cg.builder, "%s %s = ", typeName, ident.Name)
		cg.genExpr(s.Value)
		cg.builder.WriteString(";\n")
	}
}

func (cg *Codegen) genVarDeclarStmt(s *aster.VarDeclar) {
	cg.writeIndent()
	cType := cg.mapType(s.Type)
	if s.Type != nil && s.Type.IsArray {
		fmt.Fprintf(&cg.builder, "%s %s[%d];\n", cType, s.Name, s.Type.Size)
	} else {
		if s.Value != nil {
			if structLit, ok := s.Value.(*aster.StructLiteral); ok && s.Type.PtrDepth > 0 {
				cg.genHeapStructLiteral(structLit, s.Name)
			} else {
				fmt.Fprintf(&cg.builder, "%s %s = ", cType, s.Name)
				cg.genExpr(s.Value)
				cg.builder.WriteString(";\n")
			}
		} else {
			fmt.Fprintf(&cg.builder, "%s %s;\n", cType, s.Name)
		}
	}
}

func (cg *Codegen) genAssignStmt(s *aster.Assign) {
	cg.writeIndent()
	cg.genExpr(s.Target)
	cg.builder.WriteString(" = ")
	cg.genExpr(s.Value)
	cg.builder.WriteString(";\n")
}

func (cg *Codegen) genExprStmt(s *aster.ExprStmt) {
	cg.writeIndent()
	cg.genExpr(s.Expr)
	if _, isDecl := s.Expr.(*aster.Declar); !isDecl {
		cg.builder.WriteString(";\n")
	}
}

func (cg *Codegen) genReturnStmt(s *aster.ReturnStmt) {
	// إذا كانت الدالة تحتوي على مخرجات (Return Fields)
	if cg.CurrentFunction != nil && cg.CurrentFunction.Return != nil && len(cg.CurrentFunction.Return.Fields) > 0 {

		// 1. إرجاع قيمة واحدة عادية بدون أخطاء (Zero Overhead)
		if len(cg.CurrentFunction.Return.Fields) == 1 && !cg.CurrentFunction.Return.HasError {
			cg.builder.WriteString("return ")
			if len(s.Results) > 0 {
				cg.genExpr(s.Results[0])
			}
			cg.builder.WriteString(";\n")
			return
		}

		// 2. إرجاع متعدد (باستخدام Envelope Struct)
		envelopeName := cg.getEnvelopeName(cg.CurrentFunction.FuncName)

		// حالة خاصة: إذا كان التعبير المرجح هو استدعاء دالة أخرى ترجع نفس الـ Envelope مباشرة
		// مثال: return getOtherUserAndObj();
		if len(s.Results) == 1 && len(cg.CurrentFunction.Return.Fields) > 1 {
			// يمكنك إرجاع النتيجة مباشرة بدون تعبئة غلاف جديد
			cg.writeIndent()
			cg.builder.WriteString("return ")
			cg.genExpr(s.Results[0])
			cg.builder.WriteString(";\n")
			return
		}

		cg.writeIndent()
		fmt.Fprintf(&cg.builder, "%s __ret_env = {0};\n", envelopeName)

		if len(s.Results) == 0 {
			// حالة الـ Named Returns الضمنية (مثال: return;)
			for _, field := range cg.CurrentFunction.Return.Fields {
				if field.Name != "" {
					cg.writeIndent()
					fmt.Fprintf(&cg.builder, "__ret_env.%s = %s;\n", field.Name, field.Name)
				}
			}
		} else {
			// إرجاع صريح مقترن بالترتيب (مثال: return user, obj;)
			for i, expr := range s.Results {
				if i < len(cg.CurrentFunction.Return.Fields) {
					fieldName := cg.CurrentFunction.Return.Fields[i].Name
					cg.writeIndent()
					fmt.Fprintf(&cg.builder, "__ret_env.%s = ", fieldName)
					cg.genExpr(expr)
					cg.builder.WriteString(";\n")
				}
			}
		}

		cg.writeIndent()
		cg.builder.WriteString("return __ret_env;\n")
	} else {
		// void return
		cg.writeIndent()
		cg.builder.WriteString("return;\n")
	}
}
func (cg *Codegen) genForStmt(s *aster.ForStmt) {
	cg.writeIndent()
	cg.builder.WriteString("for (")

	if s.Init != nil {
		switch initStmt := s.Init.(type) {
		case *aster.Assign:
			cg.genExpr(initStmt.Target)
			cg.builder.WriteString(" = ")
			cg.genExpr(initStmt.Value)
		case *aster.Declar:
			cg.builder.WriteString("int32_t ")
			cg.genExpr(initStmt.Name)
			cg.builder.WriteString(" = ")
			cg.genExpr(initStmt.Value)
		}
	}

	cg.builder.WriteString("; ")
	if s.Cond != nil {
		cg.genExpr(s.Cond)
	}
	cg.builder.WriteString("; ")
	if s.Post != nil {
		if assign, ok := s.Post.(*aster.Assign); ok {
			cg.genExpr(assign.Target)
			cg.builder.WriteString(" = ")
			cg.genExpr(assign.Value)
		}
	}
	cg.builder.WriteString(") ")
	cg.genBlock(s.Body)
}
*/
