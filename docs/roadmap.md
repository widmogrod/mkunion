# Roadmap
## Learning and adoption

- [ ] **docs**: document simple state machine and how to use `mkunion` for it
- [ ] **feature**: `mkunion watch ./...` command that watches for changes in files and runs faster than `go generate ./...` that executes each go:generate separately
- [ ] **docs**: document other packages in `x/` directory
- [ ] **docs**: document typescript types generation and end-to-end typs concepts (from backend to frontend)
- [ ] **feature**: expose functions to extract `go:tag` metadata
- [ ] **docs**: describe philosophy of "data as resource" and how it translates to some of library concepts

## Long tern experiments and prototypes

- [ ] **experiment**: generate other (de)serialization formats (e.g. grpc, sql, graphql)
- [ ] **prototype**: http & gRPC client for end-to-end types. 
- [ ] **experiment**: allow to derive behaviour for types, like derive(Map), would generated union type with Map() method  