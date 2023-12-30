package typedful

import "github.com/widmogrod/mkunion/x/storage/schemaless"

//go:generate go run ../../../../cmd/mkunion/main.go serde

//go:tag serde:"json"
type User struct {
	Name string
	Age  int
}

//go:tag serde:"json"
type UsersCountByAge struct {
	Count int
}

func AgeRangeKey(age int) string {
	if age < 20 {
		return "byAge:0-20"
	} else if age < 30 {
		return "byAge:20-30"
	} else if age < 40 {
		return "byAge:30-40"
	} else {
		return "byAge:40+"
	}
}

var exampleUserRecords = schemaless.Save(
	schemaless.Record[User]{
		ID:   "1",
		Type: "user",
		Data: User{
			Name: "John",
			Age:  20,
		},
	},
	schemaless.Record[User]{
		ID:   "2",
		Type: "user",
		Data: User{
			Name: "Jane",
			Age:  30,
		},
	},
	schemaless.Record[User]{
		ID:   "3",
		Type: "user",
		Data: User{
			Name: "Alice",
			Age:  39,
		},
	},
)
