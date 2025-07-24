# Roadmap
## Learning and adoption

- [x] **docs**: document simple state machine and how to use `mkunion` for it
- [x] **feature**: `mkunion watch ./...` command that watches for changes in files and runs faster than `go generate ./...`
- [x] **feature**: `go:tag mkmatch` to generate pattern matching functions
- [x] **feature**: `go:tag mkmatch` with better type parameters validation
- [x] **docs**: document how to write custom pattern matching functions
- [ ] **docs**: document other packages in `x/` directory
- [ ] **docs**: document typescript types generation and end-to-end typs concepts (from backend to frontend)
- [ ] **feature**: expose functions to extract `go:tag` metadata
- [ ] **docs**: describe philosophy of "data as resource" and how it translates to some of library concepts
- [x] **feature**: remove need to provide name for `//go:tag mkmatch:"<name>"` and allow to have only `//go:tag mkmatch`
- [ ] **feature**: allow to specify type param name in `go:tag mkunion:"Tree[A]"` with validation of expected number and name of type parameters
- [ ] **bug fix**: prevent dot imports adding same type into registry but in different place and causing panic

## Long tern experiments and prototypes

- [ ] **experiment**: generate other (de)serialization formats (e.g. grpc, sql, graphql)
- [ ] **prototype**: http & gRPC client for end-to-end types.
- [ ] **experiment**: allow to derive behaviour for types, like derive(Map), would generated union type with Map() method
- [ ] **experiment**: consider adding explicit discriminator type names like `example.Branch[int]` instead of `example.Branch`. This may complicate TypeScript codegen but it could increase end-to-end type safety.
- [ ] **refactor**: `x/storage` instead of generic, leverage schema information to remove lookup of schemas (overhead), eventually generate storage code
