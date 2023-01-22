package tictacstatemachine

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/machine"
)

var (
	ErrGameAlreadyStarted = errors.New("game already started")
	ErrGameHasAllPlayers  = errors.New("game is not waiting for player")
	ErrUniquePlayers      = errors.New("game can not have same player twice")
	ErrGameNotInProgress  = errors.New("cannot move, game is not in progress")
	ErrNotYourTurn        = errors.New("not your turn")
	ErrPositionTaken      = errors.New("position is taken")
	ErrGameFinished       = errors.New("game is finished")
	ErrInputInvalid       = errors.New("input is invalid")
)

func Transition(cmd Command, state State) (State, error) {
	return MustMatchCommandR2(
		cmd,
		func(x *CreateGameCMD) (State, error) {
			if state != nil {
				return nil, ErrGameAlreadyStarted
			}

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
			newState, ok := state.(*GameWaitingForPlayer)
			if !ok {
				return nil, ErrGameHasAllPlayers
			}

			if newState.FirstPlayerID == x.SecondPlayerID {
				return nil, ErrUniquePlayers
			}

			base := newState.TicTacToeBaseState
			base.SecondPlayerID = x.SecondPlayerID

			return &GameProgress{
				TicTacToeBaseState: base,
				MovesTaken:         map[Move]PlayerID{},
				MovesOrder:         []Move{},
				NextMovePlayerID:   newState.FirstPlayerID,
			}, nil
		},
		func(x *StartGameCMD) (State, error) {
			newState, err := Transition(&CreateGameCMD{
				FirstPlayerID: x.FirstPlayerID,
				BoardRows:     x.BoardRows,
				BoardCols:     x.BoardCols,
				WinningLength: x.WinningLength,
			}, state)

			if err != nil {
				return nil, err
			}

			return Transition(&JoinGameCMD{
				SecondPlayerID: x.SecondPlayerID,
			}, newState)
		},
		func(x *MoveCMD) (State, error) {
			if IsGameFinished(state) {
				return nil, ErrGameFinished
			}

			newState, ok := state.(*GameProgress)
			if !ok {
				return nil, ErrGameNotInProgress
			}

			if newState.NextMovePlayerID != x.PlayerID {
				return nil, ErrNotYourTurn
			}

			move, err := ParsePosition(x.Position, newState.BoardRows, newState.BoardCols)
			if err != nil {
				return nil, err
			}

			if _, ok := newState.MovesTaken[move]; ok {
				return nil, ErrPositionTaken
			}

			if newState.MovesTaken == nil {
				newState.MovesTaken = map[Move]PlayerID{}
			}

			newState.MovesTaken[x.Position] = x.PlayerID
			newState.MovesOrder = append(newState.MovesOrder, move)

			if x.PlayerID == newState.FirstPlayerID {
				newState.NextMovePlayerID = newState.SecondPlayerID
			} else {
				newState.NextMovePlayerID = newState.FirstPlayerID
			}

			// Check if there is a winner
			winseq := GenerateWiningPositions(newState.WinningLength, newState.BoardRows, newState.BoardCols)
			if seq, win := CheckIfMoveWin(newState.MovesOrder, winseq); win {
				return &GameEndWithWin{
					TicTacToeBaseState: newState.TicTacToeBaseState,
					Winner:             x.PlayerID,
					WiningSequence:     seq,
					MovesTaken:         newState.MovesTaken,
				}, nil
			} else if len(newState.MovesTaken) == (newState.BoardRows * newState.BoardCols) {
				return &GameEndWithDraw{
					TicTacToeBaseState: newState.TicTacToeBaseState,
					MovesTaken:         newState.MovesTaken,
				}, nil
			}

			return newState, nil
		},
		func(x *GiveUpCMD) (State, error) {
			newState, ok := state.(*GameProgress)
			if !ok {
				return nil, ErrGameNotInProgress
			}

			winnerID := newState.FirstPlayerID
			if x.PlayerID == newState.FirstPlayerID {
				winnerID = newState.SecondPlayerID
			}

			return &GameEndWithWin{
				TicTacToeBaseState: newState.TicTacToeBaseState,
				Winner:             winnerID,
				MovesTaken:         newState.MovesTaken,
			}, nil
		},
	)
}

func NewMachine() *machine.Machine[Command, State] {
	return machine.NewSimpleMachine(Transition)
}

func NewMachineWithState(s State) *machine.Machine[Command, State] {
	return machine.NewSimpleMachineWithState(Transition, s)
}

func ParsePosition(position Move, boardRows int, boardCols int) (Move, error) {
	var r, c int
	_, err := fmt.Sscanf(position, "%d.%d", &r, &c)
	if err != nil {
		return "", fmt.Errorf("move cannot be parsed %w; %s", ErrInputInvalid, err)
	}

	if r < 1 ||
		c < 1 ||
		r > boardRows ||
		c > boardCols {
		return "", fmt.Errorf("move position is out of bounds %w", ErrInputInvalid)
	}

	return MkMove(r, c), nil

}

func GameRules(rows int, cols int, length int) (int, int, int) {
	r, c, l := rows, cols, length

	max := 10

	if l < 3 {
		l = 3
	} else if l > max {
		l = max
	}

	if r <= l {
		r = l
	}

	if c <= l {
		c = l
	}

	return r, c, l
}

func IsGameFinished(x State) bool {
	switch x.(type) {
	case *GameEndWithDraw,
		*GameEndWithWin:
		return true
	}

	return false
}
