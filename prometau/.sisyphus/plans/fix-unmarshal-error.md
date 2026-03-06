# Plan: Fix JSON Type Marshaling Issue in API

## Context 
The current implementation has a custom `ShouldBind` function in `gin_server.go` that converts all JSON number types to strings, addressing login field type mismatches. However, this causes problems for APIs that require strict typing (like product operations with `category_id int64`).

## Problem Statement
1. **Original issue**: Login requests couldn't bind when `password` was sent as a number
2. **Current issue**: Product creation/update operations fail when sending integer values as strings 
3. **Root cause**: Universal type conversion without context awareness

## Solution Strategy: Context-Aware Type Conversion
Implement conditional type conversion based on the target struct type rather than universal conversion:

### 1. Enhanced ShouldBind Method in GinContext
Modify the approach to distinguish between:
- Login/Authentication objects where flexibility is desired (string conversion allowed)
- Data/Resource objects where strict typing is essential

### 2. Implementation Details
1. Identify target object type using reflection 
2. Apply type conversion only for known authentication objects
3. Use strict binding for resource management objects
4. Preserve existing behavior for other formats (form, multipart)

### 3. Specific Target Detection Methods

Option A: Struct name pattern matching
- Check if object contains `LoginRequest`, `AuthInfo`, etc. in its type name
- Enable flexible conversion only for those cases

Option B: Interface-based recognition
- Define marker interface for objects requiring flexible typing
- Implement type checker

Option C: Manual object registry
- Explicit type mapping for which objects need flexible typing

### 4. Expected Outcomes
- ✅ Login with numeric fields still works
- ✅ Product operations with int64, int, float64 fields still work
- ✅ Backward compatibility for existing clients
- ✅ Minimal impact on performance

### 5. Test Scenarios 
- [ ] Send JSON to login with passwords as numbers and strings
- [ ] Send JSON to create product with numerical fields (category_id, stock, price)
- [ ] Send mixed type data to ensure proper handling
- [ ] Verify existing functionality remains intact

## Risk Mitigation
- Keep fallback methods if type detection fails
- Ensure logging for diagnostic purposes
- Maintain current error behaviors for unsupported formats