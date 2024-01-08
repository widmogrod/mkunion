package schemaless

//go:generate go run ../../../cmd/mkunion/main.go

//go:tag serde:"json"
type ExampleRecord struct {
	Name string
	Age  int
}

// refactored exampleUpdateRecords that use Save
var exampleUpdateRecords = Save(
	Record[ExampleRecord]{
		ID:   "123",
		Type: "ExampleRecord",
		Data: ExampleRecord{
			Name: "John",
			Age:  20,
		},
	},
	Record[ExampleRecord]{
		ID:   "124",
		Type: "ExampleRecord",
		Data: ExampleRecord{
			Name: "Jane",
			Age:  30,
		},
	},
	Record[ExampleRecord]{
		ID:   "313",
		Type: "ExampleRecord",
		Data: ExampleRecord{
			Name: "Alice",
			Age:  39,
		},
	},
	Record[ExampleRecord]{
		ID:   "1234",
		Type: "ExampleRecord",
		Data: ExampleRecord{
			Name: "Bob",
			Age:  40,
		},
	},
	Record[ExampleRecord]{
		ID:   "3123",
		Type: "ExampleRecord",
		Data: ExampleRecord{
			Name: "Zarlie",
			Age:  39,
		},
	},
)
