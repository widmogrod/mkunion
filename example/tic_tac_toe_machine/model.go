package tictacstatemachine

// Value objects, values that restrict cardinality of the state.
type (
	PlayerID = string
	Move     = string
)

// Commands that trigger state transitions.
//
//go:tag mkunion:"Command"
type (
	CreateGameCMD struct {
		FirstPlayerID PlayerID
		BoardRows     int
		BoardCols     int
		WinningLength int
	}
	JoinGameCMD  struct{ SecondPlayerID PlayerID }
	StartGameCMD struct {
		FirstPlayerID  PlayerID
		SecondPlayerID PlayerID
		BoardRows      int
		BoardCols      int
		WinningLength  int
	}
	MoveCMD struct {
		PlayerID PlayerID
		Position Move
	}
	GiveUpCMD struct {
		PlayerID PlayerID
	}
)

// State of the game.
// Commands are used to update or change state
//
//go:tag mkunion:"State"
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

type TicTacToeBaseState struct {
	FirstPlayerID  PlayerID
	SecondPlayerID PlayerID
	BoardRows      int
	BoardCols      int
	WinningLength  int
}
