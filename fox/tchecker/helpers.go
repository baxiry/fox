package tchecker

import (
	"fmt"
	"fox/aster"
	"fox/symbols"
)

// cloneType creates a deep copy of a Type to avoid circular references and field mismatching.
func (tc *TypeChecker) cloneType(t *symbols.Type) *symbols.Type {
	if t == nil {
		return nil
	}
	return &symbols.Type{
		Name:     t.Name,
		PtrDepth: t.PtrDepth,
		IsArray:  t.IsArray,
		Size:     t.Size,
	}
}

// Helper to build primitive types consistently without repetition
func newType(name string, ptrDepth int, isArray bool) *symbols.Type {
	return &symbols.Type{Name: name, PtrDepth: ptrDepth, IsArray: isArray}
}

// Function to convert program parameters
func mapParamsToSymbols(asterParams []aster.Param) []symbols.Param {
	result := make([]symbols.Param, len(asterParams))
	for i, param := range asterParams {
		result[i] = symbols.Param{
			Name: param.Name,
			Type: (*symbols.Type)(param.Type),
		}
	}
	return result
}

// Function to convert corrected structure fields
func mapFieldsToSymbols(asterFields []aster.Field) []symbols.StructField {
	result := make([]symbols.StructField, len(asterFields))
	for i, field := range asterFields {
		result[i] = symbols.StructField{
			Name: field.Name,
			Type: (*symbols.Type)(field.Type),
		}
	}
	return result
}

// Correct implementation: enforce line as the first parameter
func (tc *TypeChecker) appendErrorf(format string, line int, args ...any) {
	msg := fmt.Sprintf(format, args...)
	tc.Errors = append(tc.Errors, fmt.Sprintf("line %d: %s", line, msg))
}
