
# TODO: Frontend Compiler Security Layer

## Compile-Time Safety Guard for Flat/True Tagged Unions

- Implement Strict Initialization Checking inside the `TypeChecker` to ensure all fields are verified before any runtime use
- Enforce Compile-Time Bounds Checking to definitely block any raw access to `union` variants outside of an explicit `match` block
- Prevent memory leakage or uninitialized register parsing by creating a formal control flow analysis matrix for dynamic assignability
- Validate that all declared pointer depths `PtrDepth` match the physical structural expectations during type inference routines

