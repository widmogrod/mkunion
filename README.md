# mkunion
Improves work with unions in golang by generating beautiful code (in other languages referred as sum types, variants, discriminators, tagged unions)

Project generates code for you, so you don't have to write it by hand. 
It's a good idea to use it when you have a lot of unions in your codebase.

What is offers?
- Visitor interface with appropriate methods added to each union type
- Default implementation of Visitor that simplifies work with unions
- Reducer that can do recursive traversal & default implementation of Reducer, fantastic for traversing ASTs

Have fun! I hope you will find it useful.

## Usage
### Install mkunion
Make sure that you have installed mkunion and is in GOPATH/bin
```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@v1
```

### Create your first union
Create your first union. In our example it's a simple tree with Branch and Leaf nodes
```go
package example

//go:generate mkunion -name=Tree -types=Branch,Leaf
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
)
```

### Generate code
Run 
```
go generate ./...
```

Go will generate few files for you in the same location as union defnition
```
// source file
example/tree_example.go
// generated file
example/tree_example_mkunion_default_visitor.go
example/tree_example_mkunion_reducer.go
example/tree_example_mkunion_visitor.go
```
Don't commit generated files to your repository. They are generated on the fly.
In your CI/CD process you need to run go generate before testing & building your project.


### Use generated code
With our example you may want to sum all values in tree.

```go
tree := &Branch{
    L: &Leaf{Value: 1},
    R: &Branch{
        L: &Leaf{Value: 2},
        R: &Leaf{Value: 3},
    },
}

var red TreeReducer[int] = &TreeDefaultReduction[int]{
    OnBranch: func(x *Branch, agg int) (result int, stop bool) {
        // don't do anything, but continue traversing
        return agg, false
    },
    OnLeaf: func(x *Leaf, agg int) (int, bool) {
        // add value to accumulator
        return agg + x.Value, false
    },
}

result := ReduceTree(red, tree, 0)
assert.Equal(t, 6, result)
```

> Note: You can see that generated code knows how to traverse union recursively. 
> You can write flat code and don't worry about recursion.

> Generator assumes that if in structure is reference to union type `Tree`, then it's recursive.
> Such code can also work on slices. You can take a look at `example/where_predicate_example.go` to see something more complex


You may decide that you want to write down your own visitor for that, then you can do it like this:
```go
var _ TreeVisitor = (*sumVisitor)(nil)

type sumVisitor struct{}

func (s sumVisitor) VisitBranch(v *Branch) any {
    return v.L.Accept(s).(int) + v.R.Accept(s).(int)
}

func (s sumVisitor) VisitLeaf(v *Leaf) any {
    return v.Value
}
```

> Note: Naturally your visitor can be more complex, but it's up to you.

You can use `sumVisitor` like this:
```go
assert.Equal(t, 6, tree.Accept(&sumVisitor{}))
```

## More examples 
Please take a look at `./example` directory. It contains more examples of generated code.

Have fun! I hope you will find it useful.

## Development & contribution
When you want to contribute to this project, go for it! 
Unit test are must have for any PR.

Other than that, nothing special is required. 
You may want to create issue to describe your idea before you start working on it.
That will help other developers to understand your idea and give you feedback.

```
go generate ./...
go test ./...
```

## Roadmap ideas
### V1.0.x
- [x] Add visitor generation for unions
- [x] Add support for depth-first traversal
- [x] Add support for slice []{Variant} type traversal

### V1.1.x
- [x] Add support for map[any]{Variant} type
- [ ] Add breath-first reducer traversal

### V1.2.x
- [ ] Add support for not-stop able reducer

### V1.3.x
- [ ] Add support for multiple go:generate mkunion in one file

### V2.x.x
- [ ] Add support for generic union types

## Knows bugs
- [ ] Multiple go:generates mkunion in one file overwrite generated code. 
  Solution: split it to multiple files
