package tictacstatemachine

import (
	"fmt"
	"io"
	"sort"
)

func MkMove(x, y int) Move {
	return fmt.Sprintf("%d.%d", x, y)
}

func Wining() [][]Move {
	return GenerateWiningPositions(3, 3, 3)
}

func Wining3x3() [][]Move {
	horizontalWins := [][]Move{
		{"1.1", "1.2", "1.3"},
		{"2.1", "2.2", "2.3"},
		{"3.1", "3.2", "3.3"},
	}
	verticalWins := [][]Move{
		{"1.1", "2.1", "3.1"},
		{"1.2", "2.2", "3.2"},
		{"1.3", "2.3", "3.3"},
	}
	diagonalWins := [][]Move{
		{"1.1", "2.2", "3.3"},
		{"1.3", "2.2", "3.1"},
	}

	return append([][]Move{},
		append(horizontalWins,
			append(verticalWins, diagonalWins...)...)...)
}

var winningPositionsCache = map[string][][]Move{}

func CachedGenerateWiningPositions(rows, cols, winseq int) [][]Move {
	key := fmt.Sprintf("%d-%d-%d", rows, cols, winseq)
	if v, ok := winningPositionsCache[key]; ok {
		return v
	}
	winningPositionsCache[key] = GenerateWiningPositions(winseq, rows, cols)
	return winningPositionsCache[key]
}

func GenerateWiningPositions(inline int, rows, columns int) [][]Move {
	var winseq [][]Move
	// horizontal column shiftC of winning sequences
	maxShiftC := (columns - inline)
	maxShiftR := (rows - inline)
	for shift := 0; shift <= maxShiftC; shift++ {
		for row := 1; row <= rows; row++ {
			var seq []Move
			for col := 1 + shift; col <= inline+shift; col++ {
				seq = append(seq, Move(fmt.Sprintf("%d.%d", row, col)))
			}
			winseq = append(winseq, seq)
		}
	}

	// vertical row shiftC of winning sequences
	for shift := 0; shift <= maxShiftR; shift++ {
		for col := 1; col <= columns; col++ {
			var seq []Move
			for row := 1 + shift; row <= inline+shift; row++ {
				seq = append(seq, Move(fmt.Sprintf("%d.%d", row, col)))
			}
			winseq = append(winseq, seq)
		}
	}

	// diagonal shift of winning sequences
	for shiftR := 0; shiftR <= maxShiftR; shiftR++ {
		for shiftC := 0; shiftC <= maxShiftC; shiftC++ {
			var seqLeftBottom []Move
			for i := 1; i <= inline; i++ {
				seqLeftBottom = append(seqLeftBottom, Move(fmt.Sprintf("%d.%d", i+shiftR, i+shiftC)))
			}
			winseq = append(winseq, seqLeftBottom)

			var seqRightBottom []Move
			for i := 1; i <= inline; i++ {
				seqRightBottom = append(seqRightBottom, Move(fmt.Sprintf("%d.%d", i+shiftR, inline-i+shiftC+1)))
			}
			winseq = append(winseq, seqRightBottom)

		}
	}

	return winseq
}

func CheckIfMoveWin(moves []Move, winseq [][]Move) ([]Move, bool) {
	moves1 := append([]Move{}, moves...)

	a_moves, b_moves := make([]Move, 0), make([]Move, 0)
	for idx, m := range moves1 {
		if idx%2 == 0 {
			a_moves = append(a_moves, m)
		} else {
			b_moves = append(b_moves, m)
		}
	}

	sort.Strings(a_moves)
	sort.Strings(b_moves)

	for _, seq := range winseq {
		if wonpos, won := is_win_seq(a_moves, seq); won {
			return wonpos, true
		}
		if wonpos, won := is_win_seq(b_moves, seq); won {
			return wonpos, true
		}
	}

	return nil, false
}

func is_win_seq(moves, seq []Move) ([]Move, bool) {
	if len(moves) < len(seq) {
		return nil, false
	}

	for i := 0; i < len(seq); i++ {
		var foundInMoves bool
		for j := 0; j < len(moves); j++ {
			if seq[i] == moves[j] {
				foundInMoves = true
				break
			}
		}
		if !foundInMoves {
			return nil, false
		}
	}

	return seq, true
}

func PrintGameRC(w io.Writer, movesTaken map[Move]PlayerID, rows, cols int) {
	fmt.Fprintln(w)
	for i := 1; i <= rows; i++ {
		for j := 1; j <= cols; j++ {
			move := MkMove(i, j)
			if playerID, ok := movesTaken[move]; ok {
				fmt.Fprint(w, playerID)
			} else {
				fmt.Fprint(w, "_")
			}
			fmt.Fprint(w, " | ")
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func ToMovesTaken(moves []Move) map[Move]PlayerID {
	mm := map[Move]PlayerID{}
	for i, m := range moves {
		if i%2 == 0 {
			mm[m] = "O"
		} else {
			mm[m] = "X"
		}
	}
	return mm
}
