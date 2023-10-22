package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
)

type Execution struct {
	FlowID    string
	Status    State
	Location  string
	StartTime int64
	EndTime   int64
	Variables map[string]schema.Schema
}

//go:generate go run ../../cmd/mkunion/main.go -name=Command
type (
	Run struct {
		//FlowID string
		Flow  Worflow
		Input schema.Schema
	}
	Callback struct {
		CallbackID string
		//Flow       Worflow
		Result schema.Schema
		//Fail       schema.Schema
	}
	TryRecover struct{}
)

//go:generate go run ../../cmd/mkunion/main.go -name=State
type (
	NextOperation struct {
		Result    schema.Schema
		BaseState BaseState
	}
	Done struct {
		Result    schema.Schema
		BaseState BaseState
	}
	Error struct {
		Code    string
		Reason  string
		Retried int64
		//MaxRetries int64
		BaseState BaseState
	}
	Await struct {
		CallbackID string
		Timeout    int64
		//Timeout    time.Duration
		BaseState BaseState
	}
)

type BaseState struct {
	Flow       Worflow // Flow is a reference to the flow that describes execution
	RunID      string  // RunID is a unique identifier of the execution
	StepID     string  // StepID is a unique identifier of the step in the execution
	Variables  map[string]schema.Schema
	ExprResult map[string]schema.Schema

	// Default values
	DefaultMaxRetries int64
}

func GetRunID(state State) string {
	return MustMatchState(
		state,
		func(x *NextOperation) string {
			return x.BaseState.RunID
		},
		func(x *Done) string {
			return x.BaseState.RunID
		},
		func(x *Error) string {
			return x.BaseState.RunID
		},
		func(x *Await) string {
			return x.BaseState.RunID
		},
	)
}

//go:generate go run ../../cmd/mkunion/main.go -name=Worflow
type (
	Flow struct {
		Name string // name of the flow
		Arg  string // name of the argument, which will hold the input to the flow
		Body []Expr
	}
	FlowRef struct {
		FlowID string
	}
)

//go:generate go run ../../cmd/mkunion/main.go -name=Expr
type (
	End struct {
		ID     string
		Result Reshaper
	}
	Assign struct {
		ID    string
		VarOk string
		// if VarErr is not empty, then error will be assigned to this variable
		// to give chance to handle it, before it will be returned to the caller
		// otherwise, any error will stop execution of the program
		VarErr string
		Val    Expr
	}
	Apply struct {
		ID    string
		Name  string
		Args  []Reshaper
		Await *ApplyAwaitOptions
	}
	Choose struct {
		ID   string
		If   Predicate
		Then []Expr
		Else []Expr
	}
	//// Try allows to define a rules to recover from error in certain expression block
	//// Concept of this expression is experimental and similar to try/catch in other languages.
	//// ```
	//// try {
	////   // do something
	//// } catch (e) {
	////   // handle error
	//// }
	////
	//// It could be replaced, with enforcing that error or success is always returned as value of function
	//// and then using Choose to decide what to do next.
	//// ```
	//// res = ReserveInventory(...)
	//// if res.ok {
	////   // do something
	//// } else {
	////   // handle error
	//// }
	////
	//// Alternatively, it could be replaced with let and if.
	//// Let has special property, that if second variable error is not defined, it will stop execution of program
	//// but if it is defined, it will continue execution, and problemw will be accessible as variable to handle
	//// ```
	//// let res, err = await ReserveInventory(...)
	//// if err {
	////   // handle error
	//// }
	//// // do something
	//// ```
	//Try struct {
	//	ID    string
	//	Try   []Expr
	//	Catch []Expr
	//}

	//// Resume is like reverse callback, or Apply with await.
	//// Resume waits for caller to provide a result, and then continue execution.
	//// ```
	//// 	let res, err = await ReserveInventory(...)
	//// 	if err {
	//// 		// timeout reached
	//// 		_ = CancelReservation(...)
	//// 		return({"status": "timeout"})
	////	}
	////
	//// let approved = await Int > 2
	//// let approved = await {name: String > 0 and // , age: Int > 0}
	//// if !approved {
	////   _ = CancelReservation(...)
	////   return({status: "rejected"})
	//// }
	////
	//Resume struct {
	//	ID      string
	//	Timeout time.Duration
	//	//Caller  schema.Schema
	//	//Callee  schema.Schema
	//	Options ResumeOptions
	//}
)

type ResumeOptions struct {
	Timeout int64
	//Timeout time.Duration
}

type ApplyAwaitOptions struct {
	Timeout int64
	//Timeout time.Duration
}

//go:generate go run ../../cmd/mkunion/main.go -name=Reshaper
type (
	GetValue struct {
		Path string
	}
	SetValue struct {
		Value schema.Schema
	}
)

//go:generate go run ../../cmd/mkunion/main.go -name=Predicate
type (
	And struct {
		L []Predicate
	}
	Or struct {
		L []Predicate
	}
	Not struct {
		P Predicate
	}
	Compare struct {
		Operation string
		Left      Reshaper
		Right     Reshaper
	}
)
