package symbols

import (
	"fmt"
)

type Type struct {
	Name     string
	PtrDepth int
	Size     int
	IsArray  bool
}

// Param tracks function parameter definitions context
type Param struct {
	Name string
	Type *Type
}

// ReturnSig tracks multi-return type signatures
type ReturnSig struct {
	Type *Type
}

// StructField mirrors the verified field specs inside a struct type
type StructField struct {
	Name string
	Type *Type
}

// Symbol acts as the unique canonical metadata repository for types, functions, and variables
type Symbol struct {
	Name       string
	Type       *Type
	ScopeID    string
	Kind       string        // "var", "func", "struct"
	Params     []Param       // Allocated for functions tracking context
	ReturnType *ReturnSig    // Multi-return tracking signature
	Fields     []StructField // Static structure members dimensions layout
	IsShared   bool
	IsBuiltIn  bool
	IsVariadic bool
}

// SymbolTable manages hierarchical domain visibility matrix layers with parent links
type SymbolTable struct {
	Symbols map[string]*Symbol
	Parent  *SymbolTable
	ScopeID string // Hierarchical metrics track boundaries e.g., "1.1", "1.2"
	NextID  int    // Counter tracking block dynamic sequence layout instances
}

// NewSymbolTable initializes an absolute clean lookup tracking block scope
func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Symbols: make(map[string]*Symbol),
		Parent:  parent,
	}
}

// Resolve retrieves a symbol by name, searching up the scope chain recursively (O(1) amortized)
func (st *SymbolTable) Resolve(name string) (*Symbol, bool) {
	// 1. Get symbol from the current scope's symbol table
	symb, exists := st.Symbols[name]
	if exists {
		return symb, true
	}

	// 2. Recursively search in parent symbols if not found locally
	if st.Parent != nil {
		return st.Parent.Resolve(name)
	}

	return nil, false
}

// Define adds a new symbol to the current scope's symbol table with unique strict guard validation
func (st *SymbolTable) Define(name string, sym *Symbol) error {
	if _, exists := st.Symbols[name]; exists {
		return fmt.Errorf("symbol %s already defined in this scope", name)
	}

	// Ensure the symbol's type is stored accurately
	st.Symbols[name] = sym
	return nil
}

func (st *SymbolTable) GenerateChildID() string {
	st.NextID++ // Increment internal counter
	if st.ScopeID == "" {
		return fmt.Sprintf("%d", st.NextID)
	}
	return fmt.Sprintf("%s.%d", st.ScopeID, st.NextID)
}

func (t *Type) IsSameAs(other *Type) bool {
	if t == nil || other == nil {
		return false
	}
	return t.Name == other.Name &&
		t.PtrDepth == other.PtrDepth &&
		t.IsArray == other.IsArray
}
