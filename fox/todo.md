
# TODO: Frontend Compiler Security Layer

## Compile-Time Safety Guard for Flat/True Tagged Unions

- Implement Strict Initialization Checking inside the `TypeChecker` to ensure all fields are verified before any runtime use
- Enforce Compile-Time Bounds Checking to definitely block any raw access to `union` variants outside of an explicit `match` block
- Prevent memory leakage or uninitialized register parsing by creating a formal control flow analysis matrix for dynamic assignability
- Validate that all declared pointer depths `PtrDepth` match the physical structural expectations during type inference routines





# TODO: Exporting & Visibility System (Capitalized Keywords Approach)

* Phase 1: Lexer (Lexical Analysis)
* Add capitalized export keywords to the reserved Keywords map.
   * Targeted keywords: Fn (Public Function), Struct (Public Struct), and Var (Public Variable).
   * Generate corresponding token types: TOKEN_PUB_FN, TOKEN_PUB_STRUCT, and TOKEN_PUB_VAR.
* Phase 2: Parser (Syntactic Analysis)
* Update the parseFunction logic to accept a visibility boolean based on the received token (fn passes false, Fn passes true).
   * Update parseStruct and parseVarDeclar with the same logic to inject an IsPublic flag into the AST nodes.
* Phase 3: TypeChecker & Symbol Table (Semantic Analysis)
* Add an IsPublic bool field to the Symbol struct inside the symbol table configuration.
   * Update the Resolve function scope-checking mechanism to block access to symbols where IsPublic == false if the resolution request originates from outside the current Package.

