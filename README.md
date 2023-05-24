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
go install github.com/widmogrod/mkunion/cmd/mkunion@v1.15
```

### Create your first union
Create your first union. In our example it's a simple tree with Branch and Leaf nodes
```go
package example

//go:generate mkunion -name=Tree
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
    return v.L.AcceptTree(s).(int) + v.R.AcceptTree(s).(int)
}

func (s sumVisitor) VisitLeaf(v *Leaf) any {
    return v.Value
}
```

You can use `sumVisitor` like this:
```go
assert.Equal(t, 10, tree.AcceptTree(&sumVisitor{}))
```

### Use `mkunion` to simplify state management
Let's build our intuition first and crete simple state machine that increments counter using `github.com/widmogrod/mkunion/x/machine` package

Let's import the package
```go
import "github.com/widmogrod/mkunion/x/machine"
```

Let's define our state machine
```go
m := NewSimpleMachineWithState(func(cmd string, state int) (int, error) {
  switch cmd {
  case "inc":
    return state + 1, nil
  case "dec":
    return state - 1, nil
  default:
    return 0, fmt.Errorf("unknown cmd: %s", cmd)
  }
}, 10)
```

Now to increment or decrement counter we can do it like this:
```go
err := m.Handle("inc")
assert.NoError(t, err)
assert.Equal(t, 11, m.State())
```

Simple, right?

### Let's use `mkunion` crete more complex state machine
We learn how API of machine looks like. Let's complicate above example and use `mkunion` to express distinct commands and states.

We will build state machine to manage Tic Tac Toe game.
I will not explain rules of Tic Tac Toe, and focus on how to use `mkunion` to model state transitions.

You can find full example in [example](example/tic_tac_toe_machine/model.go)

- When we want to play a game, we need to start it first. `CreateGameCMD` is command that defines rules of the game
- To allow other player to join the game we have `JoinGameCMD`
- And lastly we need a command to make a move `MakeMoveCMD`

Here is how we define those interactions:
```go
//go:generate mkunion -name=Command
type (
	CreateGameCMD struct {
		FirstPlayerID PlayerID
		BoardRows     int
		BoardCols     int
		WinningLength int
	}
	JoinGameCMD  struct{ 
		SecondPlayerID PlayerID 
	}
	MoveCMD struct {
		PlayerID PlayerID
		Position Move
	}
)
```

Next we need to rules of the game.
- We cannot start a game without two players. `GameWaitingForPlayer` state will be used to indicate his.
- When we have two players, we can start a game. `GameInProgress` state will be used to indicate his. This state allows to make moves.
- When we have a winner, we can end a game. `GameEndWithWin` or `GameEndWithDraw` state will be used to indicate his. This state does not allow to make moves.

```go
//go:generate mkunion -name=State
type (
	GameWaitingForPlayer struct {
		TicTacToeBaseState
	}

	GameProgress struct {
		TicTacToeBaseState

		NextMovePlayerID Move
		MovesTaken       map[Move]PlayerID
		MovesOrder       []Move
	}

	GameEndWithWin struct {
		TicTacToeBaseState

		Winner         PlayerID
		WiningSequence []Move
		MovesTaken     map[Move]PlayerID
	}
	GameEndWithDraw struct {
		TicTacToeBaseState

		MovesTaken map[Move]PlayerID
	}
)
```

Now we have to connect those rules by state transition.

This is how transition function looks like. 
Implementation is omitted for brevity, but whole code can be found in [machine.go](example/tic_tac_toe_machine/machine.go)

```go
func Transition(cmd Command, state State) (State, error) {
	return MustMatchCommandR2(
		cmd,
		func(x *CreateGameCMD) (State, error) {
			if state != nil {
				return nil, ErrGameAlreadyStarted
			}

			// validate game rules
			rows, cols, length := GameRules(x.BoardRows, x.BoardCols, x.WinningLength)

			return &GameWaitingForPlayer{
				TicTacToeBaseState: TicTacToeBaseState{
					FirstPlayerID: x.FirstPlayerID,
					BoardRows:     rows,
					BoardCols:     cols,
					WinningLength: length,
				},
			}, nil
		},
		func(x *JoinGameCMD) (State, error) {
			// omitted for brevity
		},
		func(x *MoveCMD) (State, error) {
          // omitted for brevity
		},
	)
}
```
We define `Transition` that use `MustMatchCommandR2` that was generated by `mkunion` to manage state transition
Now we will use `github.com/widmogrod/mkunion/x/machine` package to provide unifed API for state machine

```go
m := machine.NewMachineWithState(Transition, nil)
err := m.Handle(&CreateGameCMD{
    FirstPlayerID: "player1",
    BoardRows:     3,
    BoardCols:     3,
    WinningLength: 3,
})
assert.NoError(t, err)
assert.Equal(t, &GameWaitingForPlayer{
    TicTacToeBaseState: TicTacToeBaseState{
        FirstPlayerID: "player1",
        BoardRows:     3,
        BoardCols:     3,
        WinningLength: 3,
    },
}, m.State())
```

This is it. We have created state machine that manages Tic Tac Toe game.

Now with power of `x/schema` transiting golang records and union types over network like JSON is easy.

```go
// deserialise client commands
schemaCommand, err := schema.FromJSON(data)
cmd, err := schema.ToGo(schemaCommand)

// apply command to state (you may want to load it from database, s/schema package can help with that, it has DynamoDB schema support)
m := machine.NewMachineWithState(Transition, nil)
err = m.Handle(cmd)

// serialise state to send to client
schemaState := schema.FromGo(m.State())
data, err := schema.ToJSON(schemaState)
```

This is all. I hope you will find this useful.


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
- [x] Introduce `github.com/widmogrod/mkunion/x/machine` for simple state machine construction

### V1.9.x
- [x] Introduce schema helper functions like Get(schema, location), As[int](schema, default), Reduce[A](schema, init, func(schema, A) A) A
- [x] Allow to have union with one element, this is useful for domain model that is not yet fully defined
- [x] `x/schema` breaking change. `ToGo` returns any and error. Use `MustToGo` to get panic on error

### V1.10.x
- [x] Introduce `Match*` and `Match*R2` functions, that offer possibility to specif behaviour when value is `nil`

### V1.14.x
- [x] Introduce `Match*R0` and `MustMatch*R0` functions, that allow matching but don't return any value

### V1.15.x
- [x] Union interface has method `Accept{Varian}` instead of just `Accept`. Thanks to that is possible to use the same type in multiple unions. Such feature is beneficial for domain modelling.
- [x] CLI `mkunion` change flag `-types` to `-variants`

### V1.16.x
- [ ] Allow to change visitor name form Visit* to i.e Handle*
- [ ] Allow extending (embedding) base Visitor interface with external interface
- [ ] Schema Registry should reject registration of names that are already registered!
- [ ] Add configurable behaviour how schema should act when field is missing, but schema has a value for it

### V2.x.x
- [ ] Add support for generic union types

