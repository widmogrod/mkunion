---
title: Persisting union in database
---
## Persisting state in database

At this point, we have implemented and tested the Order Service state machine.

The next thing that we need to address on our road to production is to persist state in a database.

MkUnion aims to support you in this task by providing you with the `x/storage/schemaless` package that will take care of:

- mapping Go structs to database representation and back from the database to a struct.
- leveraging optimistic concurrency control to resolve conflicts
- providing you with a simple API to work with the database
- and more

Below is a test case that demonstrates a complete example of initializing a database,
building a state using `NewMachine`, and saving and loading state from the database.

```go title="example/state/machine_database_test.go"
--8<-- "example/state/machine_database_test.go:16:46"
```

```mermaid
sequenceDiagram
    participant R as Request
    participant Store as Store

    activate R
    R->>R: Validate(request) -> error
    
    R->>Store: Load state from database by request.ObjectId
    activate Store
    Store->>R: Ok(State)
    deactivate Store
    
    R->>R: Create machine with state
    R->>R: Apply command to state
    
    R->>Store: Save state in database under request.ObjectId
    activate Store
    Store->>R: Ok()
    deactivate Store
    
    deactivate R
```

Example implementation of such a sequence diagram:

```go
func Handle(rq Request, response Response) { // Assuming Resopnse was a typo for Response
	ctx := rq.Context()
	
	// extract `objectId` and `command` from the request and perform some validation
    id := rq.ObjectId
	command := rq.Command
	
    // Load the state from the store
    state, err := store.Find(ctx, id)
	if err != nil { /*handle error*/ }

    machine := NewSimpleMachineWithState(Transition, state)
    newState, err := machine.Apply(command, state) // cmd was used before, assuming command
    if err != nil { /*handle error*/ }
	
    err = store.Save(ctx, newState) // Assuming missing =
    if err != nil { /*handle error*/ }
	
	// serialize the response
	response.Write(newState)
}
```

## Error as state. Self-healing systems.
In a request-response situation, handling errors is easy, but what if something goes wrong in some long-lived process?
How should errors be handled in such a situation? Without making what we've learned about state machines useless or hard to use?

One solution is to treat errors as state.
In such a case, our state machines will never return an error, but instead will return a new state that will represent an error.

When we introduce an explicit command responsible for correcting `RecoverableError`, we can create self-healing systems.
Thanks to that, even in situations where errors are unknown, we can retroactively introduce self-healing logic that corrects states.

Since there is always only one error state, it makes such state machines easy to reason about.

```go
//go:generate mkunion -name State
type (
    // ...
    RecoverableError struct {
        ErrCode int
        PrevState State
        RetryCount int
    }
)

//go:generate mkunion -name Command
type (
    // ...
    CorrectStateCMD struct {}
)
```

Now, we have to implement the recoverable logic in our state machine.
The example above shows how to do it in the `Transition` function.

Here is an example implementation of such a transition function:

```go
func Transition(cmd Command, state State) (State, error) {
return MustMatchCommandR2(
    cmd,
    /* ... */
    func(cmd *CorrectStateCMD) (State, error) {
        switch state := state.(type) {
        case *RecoverableError:
            state.RetryCount = state.RetryCount + 1
			
            // here we can do some self-healing logic
            if state.ErrCode == DuplicateServiceUnavailable { // Assuming DuplicateServiceUnavailable is a defined error code
                newState, err := Transition(&MarkAsDuplicateCMD{}, state.PrevState) // Assuming MarkAsDuplicateCMD is a defined command
                 if err != nil {
                    // we failed to correct the error, so we return an error state 
                     return &RecoverableError{
                        ErrCode:    0, // Consider setting a more specific error code from 'err'
                        PrevState:  state.PrevState,
                        RetryCount: state.RetryCount,
                    }, nil
                }
				
                 // we managed to fix the state, so we return new state
                 return newState, nil
             } else {
                 // log information that we have a new error code that we don't know how to handle
             }
			
            // try to correct the error in the next iteration
            return state, nil
        }
        return state, nil // Added default return for the switch
    },
) // Assuming MustMatchCommandR2 is a function that requires this closing parenthesis
}
```

Now, to correct states, we have to select all states that are in an error state from the database.
It can be used in many ways; the example below uses an abstraction called `TaskQueue` that is responsible for running tasks in the background.

This abstraction guarantees that all records (historical and new ones) will be processed.
You can think of it as a queue that is populated by records from the database that meet SQL query criteria.

You can use a CRON job and poll the database.

```go
//go:generate mms deployyml -type=TaskQueue -name=CorrectMSPErrors -autoscale=1,10 -memory=128Mi -cpu=100m -timeout=10s -schedule="0 0 * * *"
func main()
    sql := "SELECT * FROM ObjectState WHERE RecoverableError.RetryCount < 3" // Assuming ObjectState is the table
    store := datalayer.DefaultStore() // Assuming datalayer.DefaultStore() is available
    queue := TaskQueueFrom("correct-msp-errors", sql, store) // Assuming TaskQueueFrom is available
    queue.OnTask(func (ctx context.Context, task Task) error { // Assuming Task type is defined
        state := task.State()
        cmd := &CorrectStateCMD{}
        machine := NewSimpleMachineWithState(Transition, state)
        newState, err := machine.Apply(cmd, state) // machine.Apply might need context
        if err != nil {
            return err
        }
        return task.Save(ctx, newState) // Assuming task.Save is available
    })
    err := queue.Run(ctx) // Assuming ctx is defined
    if err != nil {
        log.Panic(err)
    }
}
```


## State machines and command queues and workflows
What if a command would initiate a state "to process" and save it in the database?
What if a task queue would take such a state and process it?
Wouldn't this be something like a command queue?

Under what conditions should background processes be used to transition these states?

### Processors per state
It's like micromanaging the TaskQueue, where each state has its own logic and knows what command to apply to a given state.
This could be a good starting point when there isn't a lot of good tooling.

### Processor for state machine
With good tooling, the transition of states can be declared in one place,
and deployment to the task queue could be done automatically.

Note that only some of the transitions need to happen in the background; others can be done in a request-response manner.

### Processor for state machine with workflow
A state machine could be generalized to a workflow.
We can think about it as a set of generic Commands and States (like a Turing machine).

States like `Pending`, `Completed`, `Failed`.
Commands like `Process`, `Retry`, `Cancel`.

And a workflow DSL with commands like: `Invoke`, `Choose`, `Assign`.
Where a function is some ID string, and functions need to be either
pulled from a registry or called remotely (`InvokeRemote`).
Some operations would require a callback (`InvokeAndAwait`).

Then a background processor would be responsible for executing such a workflow (using a task queue).
The program would be responsible for defining the workflow and registering functions.

Such programs could also be optimized for deployment,
if some function would be better to run on the same machine that makes an RPC call,
like a function making an RPC call to a database and caching the result in memory or in a cache cluster dedicated to a specific BFF.



