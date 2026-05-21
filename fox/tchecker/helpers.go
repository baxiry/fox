package tchecker

import (
	"fox/aster"
	"fox/symbols"
)

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
