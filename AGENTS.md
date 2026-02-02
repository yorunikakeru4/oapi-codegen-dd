## Building
- Use Makefile settings or commands for building
- All commands located in `cmd/`

## Testing
- All `pkg/runtime` functions should aim for 100% test coverage
- Run `go test -cover ./pkg/runtime/...` to check coverage

## Configuration
- Consult `examples/` for sample configurations for different use cases
- Running without config uses default settings:
  ```go run ./cmd/oapi-codegen <spec-path>```

### Key config options
- `output.use-single-file: true` - Generate all code in one file (default) vs multiple files
- `generate.client: true` - Generate HTTP client code
- `generate.validation.skip: true` - Skip Validate() method generation
- `generate.validation.response: true` - Generate Validate() for response types (useful for contract testing)
- `generate.always-prefix-enum-values: true` - Prefix enum constants with type name (default)
- `generate.default-int-type: int64` - Use int64 instead of int for integer types
- `skip-prune: true` - Keep unused types (normally pruned)
- `error-mapping` - Map response types to implement error interface (key: type name, value: json path to message)
- `filter.include/exclude` - Filter paths, tags, operation-ids, extensions

## Verifying changes
- After making changes to the code generator, ensure to run `make generate` which regenerates the code for all OpenAPI specs in `examples/`
- Verify that the generated code matches the expected output in `examples/`
- Run `make test` to ensure all unit tests pass
- Consider running integration tests in Connexions to verify end-to-end functionality:
  ```make test-integration```. That could take 5 minutes.

## Investigating integration test failures

### Running specific specs
- Re-run with `SPEC=<path-to-spec.yml> make test-integration` to focus on a specific spec
- Never run all integration tests at once - always use SPEC= to limit scope
- Example: `make test-integration SPEC=3.0/github.com/ghes-3.5.1.1.4.yml`

### Common failure types and resolutions

1. **libopenapi limitation - JSON Pointer refs to paths**
   - Error: `component '#/paths/~1api~1v1~1...' does not exist in the specification`
   - Cause: Spec uses `$ref` pointing to path elements which libopenapi can't resolve
   - Resolution: Skip spec by prefixing filename with `-` (e.g., `mv spec.yml -spec.yml`)

2. **Missing schema/component reference**
   - Error: `component '#/components/schemas/SomeName' does not exist`
   - Cause: Spec references a schema that doesn't exist (broken spec)
   - Resolution: Remove the spec from testdata

3. **x- prefixed schema names**
   - Error: `undefined: XAny` or similar
   - Cause: Spec has schemas named with `x-` prefix (e.g., `x-any`) which libopenapi treats as extensions
   - Resolution: Skip spec with `-` prefix

4. **External file references**
   - Error: `unable to open the rolodex file` or `../some-file.yaml does not exist`
   - Cause: Spec references external files that aren't present
   - Resolution: Remove the spec from testdata

5. **Build failures (undefined types, syntax errors)**
   - Check the generated code at the debug path shown in error output
   - Look for patterns in the generated code that indicate the issue
   - Fix the generator code, then re-run the specific spec

### Workflow
1. Get the failing spec path from test output
2. Run: `go run ./cmd/oapi-codegen <spec-path>` to see generation errors
3. If generation succeeds, check the debug path for the generated code
4. For libopenapi limitations, skip with `-` prefix
5. For broken specs, remove from testdata
6. For generator bugs, fix and re-test

### Creating reproducible test cases
When debugging complex issues, create a minimal reproducible example:
1. Create a new directory in `examples/` (e.g., `examples/issue-123/`)
2. Add a simplified spec that reproduces the issue
3. Add a `cfg.yaml` with the relevant configuration
4. Create a `gen_test.go` to verify the generated code compiles:
   ```go
   package example_test

   import (
       "testing"
       _ "github.com/yorunikakeru4/oapi-codegen-dd/v3/examples/issue-123"
   )

   func TestCompiles(t *testing.T) {}
   ```
5. Run `go generate ./examples/issue-123/...` to generate code
6. Run `go test ./examples/issue-123/...` to verify it compiles
7. After fixing, ask the user whether to keep the example or remove it

### File organization
- `testdata/specs` - Specs being tested (if missing: run `make fetch-specs` to download)
- Never put generated files in project root - use `/tmp` for testing

## Code Generation Architecture

### Two-pass schema processing
The generator uses a two-pass approach for component schemas:
1. **Pass 1 (preRegisterSchemaNames)**: Registers all schema names and refs in TypeTracker BEFORE processing. This ensures cross-references can find the correct (potentially renamed) type names.
2. **Pass 2 (generateSchemaDefinitions)**: Generates full type definitions using pre-registered names.

This is critical for handling `x-go-name` extensions and name conflicts correctly.

### TypeTracker
Central registry for managing type names and references:
- `registerName(name)` - Reserve a type name
- `registerRef(ref, name)` - Map a $ref to a type name
- `LookupByRef(ref)` - Find type name for a $ref
- `LookupByName(name)` - Find TypeDefinition by name
- `generateUniqueName(name)` - Generate unique name if conflict exists (appends numbers)

### Circular reference handling
- `ParseOptions.visited` map tracks visited schema paths to prevent infinite recursion
- When a circular reference is detected, the generator returns a reference to the already-registered type
- Pre-registration in pass 1 ensures the type name exists before it's referenced

### Enum generation
- Enums are generated for schemas with `enum` values
- Only comparable types can have enum constants (primitives like string, int, bool)
- Non-comparable types (time.Time, uuid.UUID, arrays, structs) skip enum constant generation
- Use `isComparableType()` to check if a type can be used as an enum constant
- `nonConstantTypes` map lists Go types that cannot be constants

### Response type handling
- Response types can be configured to implement the `error` interface via `output.response-type-suffix` and error mapping
- When a response type has error mapping, it cannot be an alias (aliases don't support methods)
- Response schemas are processed similarly to component schemas but with different SpecLocation

### Union types (oneOf/anyOf)
- Union types are generated as structs with pointer fields for each variant
- Use `ContainsUnions()` method on schemas to check if they contain union elements
- Union types get custom JSON marshaling/unmarshaling to handle the polymorphism

### Name conflict resolution
- When multiple schemas would generate the same Go type name, the generator appends numbers (e.g., `Foo`, `Foo2`, `Foo3`)
- This happens at registration time via `generateUniqueName()`
- Conflicts can arise from: same names in different paths, `x-go-name` collisions, inline type extraction
