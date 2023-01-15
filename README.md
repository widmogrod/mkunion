# mkunion
Improves work with unions in golang by generating beautiful code (in other languages referred as sum types, variants, discriminators, tagged unions)

Project generates code for you, so you don't have to write it by hand. 
It's a good idea to use it when you have a lot of unions in your codebase.

## What it offers?
- Visitor interface with appropriate methods added to each union type
- Default implementation of Visitor that simplifies work with unions
- Reducer that can do recursive traversal (depth and breadth first) & default implementation of Reducer, fantastic for traversing ASTs

## What it's useful for?
- **Custom DSL**. When you want to create your own DSL, you can use this library to create AST for it. (./examples/ast)
- **State machine**. When you need to manage state of your application, you can use this library to create states and transitions as unions. (./examples/state)

Have fun! I hope you will find it useful.

## Usage
### Install mkunion
Make sure that you have installed mkunion and is in GOPATH/bin
```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@v1.7
```

### Create your first union
Create your first union. In our example it's a simple tree with Branch and Leaf nodes
```go
package example

//go:generate mkunion -name=Tree -types=Branch,Leaf
type Branch struct{ L, R Tree }
type Leaf   struct{ Value int }
```

With version `1.6` you can generate union types without specifying variant types names, like so:
```go
package example

//go:generate mkunion -name=Tree
type (
    Branch struct{ L, R Tree }
    Leaf   struct{ Value int }
)
```

This is now recomended way of generating unions.

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
example/tree_example_mkunion_tree_default_reducer.go
example/tree_example_mkunion_tree_default_visitor.go
example/tree_example_mkunion_tree_reducer_bfs.go
example/tree_example_mkunion_tree_reducer_dfs.go
example/tree_example_mkunion_tree_visitor.go
```
Don't commit generated files to your repository. They are generated on the fly.
In your CI/CD process you need to run go generate before testing & building your project.


### Use generated code
With our example you may want to **sum all values in tree**.


To be precise, we want to sum values that `Leaf` struct holds.
For example, such tree needs to be summed to 10:
```go
tree := &Branch{
    L: &Leaf{Value: 1},
    R: &Branch{
        L: &Branch{
            L: &Leaf{Value: 2},
            R: &Leaf{Value: 3},
        },
        R: &Leaf{Value: 4},
    },
}
```

To sum up values in a tree we can do it in 3 ways. 
In all `mkunion` will help us to do it in a clean way.

#### 1. Implement tree reducer with help of `Match` function
This approach is familiar with everyone who use functional programming.

- In this approach you're responsible for defining how you want to travers tree. 
  We will go with depth-first traversal.
- `MustMatchTree` function will do type checking, you need to handle all cases.

```go
func MyReduceDepthFirstTree[A any](x Tree, aggregate func (int, A) A, init A) A {
    // MustMatchTree is generated function my mkunion
    return MustMatchTree(
	    x, 
	    func (x *Leaf) B {
	        return aggregate(x.Value, init)
	    },
	    func (x *Branch) B {
	        // Note: here you define traversal order
	        // Right branch first, left branch second
	        return MyReduceDepthFirstTree(x.L, aggregate, MyReduceDepthFirstTree(x.R, f, init))
	    }, 
    )
}
```

You use this function like this:
```go
result := MyReduceDepthFirstTree(tree, func (x, y int) int {
    return x + y
}, 0)
assert.Equal(t, 10, result)
```


### 2. Leverage generated default reduction with traversal strategies (depth first, breadth first)
You should use this approach
 - When you need to traverse tree in different way than a depth first, like breadth first without writing your own code
 - When you need to stop traversing of a tree at some point. For example, when you want to find a value in a tree, or meet some condition.

To demonstrate different traversal strategies, we will reduce a tree to a structure that will hold not only result of sum, but also order of nodes visited

```go
// This structure will hold order of nodes visited, and resulting sum
type orderAgg struct {
    Order  []int
    Result int
}

// This is how we define reducer function for traversal of tree

var red TreeReducer[orderAgg] = &TreeDefaultReduction[orderAgg]{
    PanicOnFallback:      false,
    DefaultStopReduction: false,
    OnLeaf: func(x *Leaf, agg orderAgg) (orderAgg, bool) {
        return orderAgg{
            Order:  append(agg.Order, x.Value),
            Result: agg.Result + x.Value,
        }, false
    },
}

// Dept first traversal
result := ReduceTreeDepthFirst(red, tree, orderAgg{})
assert.Equal(t, 10, result.Result)
assert.Equal(t, []int{1, 2, 3, 4}, result.Order) // notice that order is different!

// Breadth first traversal
result = ReduceTreeBreadthFirst(red, tree, orderAgg{})
assert.Equal(t, 10, result.Result)
assert.Equal(t, []int{1, 4, 2, 3}, result.Order) // notice that order is different!
```

Note:
- You can see that generated code knows how to traverse union recursively. 
  - You can write flat code and don't worry about recursion. 
- Generator assumes that if in structure is reference to union type `Tree`, then it's recursive. 
  - Such code can also work on slices. You can take a look at `example/where_predicate_example.go` to see something more complex


#### 3. Implement visitor interface
This is most open way to traverse tree.
- You have to implement `TreeVisitor` interface that was generated for you by `mkunion` tool.
- You have to define how traversal should happen

This approach is better when you want to hold state or references in `sumVisitor` struct.
In simple example this is not necessary, but in more complex cases you may store HTTP client, database connection or something else.

```go
// assert that sumVisitor implements TreeVisitor interface
var _ TreeVisitor = (*sumVisitor)(nil)

// implementation of sumVisitor
type sumVisitor struct{}

func (s sumVisitor) VisitBranch(v *Branch) any {
    return v.L.Accept(s).(int) + v.R.Accept(s).(int)
}

func (s sumVisitor) VisitLeaf(v *Leaf) any {
    return v.Value
}
```

You can use `sumVisitor` like this:
```go
assert.Equal(t, 10, tree.Accept(&sumVisitor{}))
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

### V1.2.x
- [x] Add breadth-first reducer traversal

### V1.3.x
- [x] Use go:embed for templates

### V1.4.x
- [x] Add function generation like `Match` to simplify work with unions
- [x] Benchmark implementation of `Match` vs Reducer (depth-first has close performance, but breadth-first is much slower)

### V1.5.x
- [x] Add support for multiple go:generate mkunion in one file

### V1.6.x
- [x] Add variant types inference
- [x] Add `Unwrap` method to OneOf

### V1.7.x
- [x] `MustMatch*R2` function return tuple as result
- [x] Introduce recursive schema prototype (`github.com/widmogrod/mkunion/x/schema` package)
- [x] Integrate with schema for json serialization/deserialization
- [x] `mkunion` can skip extensions `-skip-extensions=<generator_name>` to be generated
- [x] Remove OneOf to be the same as variant! (breaking change)

### V1.8.x
- [ ] Add state machine generation
- [ ] Allow to change visitor name form Visit* to i.e Handle*
- [ ] Allow extending (embedding) base Visitor interface with external interface
- [ ] Schema Registry should reject registration of names that are already registered!

### V2.x.x
- [ ] Add support for generic union types

