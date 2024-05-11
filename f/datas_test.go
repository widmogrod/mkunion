package f

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
)

func ExampleEitherToJSON() {
	var either Either[int, string] = &Right[int, string]{Value: "hello"}
	result, _ := shared.JSONMarshal(either)
	fmt.Println(string(result))
	// Output: {"$type":"f.Right","f.Right":{"Value":"hello"}}
}
