package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/openings"

	"github.com/corentings/chess/v2"
)

type validationOutput struct {
	CourseID       string                  `json:"courseId"`
	ContentVersion string                  `json:"contentVersion"`
	Counts         map[string]int64        `json:"counts"`
	Coverage       openings.CoverageReport `json:"coverage"`
}

type validationFailure struct {
	Diagnostics []openings.Diagnostic `json:"diagnostics"`
}

type sanlineOutput struct {
	StartFEN string        `json:"startFen"`
	Moves    []sanlineMove `json:"moves"`
	FinalFEN string        `json:"finalFen"`
}

type sanlineMove struct {
	Index int    `json:"index"`
	SAN   string `json:"san"`
	UCI   string `json:"uci"`
	FEN   string `json:"fen"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n       coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "sanline":
		return runSANLine(args[1:], stdout, stderr)
	default:
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n       coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n")
		return 2
	}
	compiled, err := openings.ValidateCoursePackFile(
		context.Background(),
		args[0],
		chessrules.Rules{},
	)
	if err != nil {
		var validationErr *openings.ValidationError
		if errors.As(err, &validationErr) {
			if encodeErr := writeIndentedJSON(stderr, validationFailure{
				Diagnostics: validationErr.Diagnostics,
			}); encodeErr != nil {
				_, _ = fmt.Fprintf(stderr, "write diagnostics: %v\n", encodeErr)
			}
		} else {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 1
	}
	if err := writeIndentedJSON(stdout, validationOutput{
		CourseID:       compiled.Pack.CourseID,
		ContentVersion: compiled.Pack.ContentVersion,
		Counts:         openings.StructuralCounts(compiled),
		Coverage:       compiled.Coverage,
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "write validation result: %v\n", err)
		return 1
	}
	return 0
}

func runSANLine(args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 || args[0] != "--fen" || strings.TrimSpace(args[1]) == "" {
		_, _ = io.WriteString(stderr, "usage: coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
	startFEN := args[1]
	sanMoves := args[2:]
	fenOption, err := chess.FEN(startFEN)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid FEN: %v\n", err)
		return 1
	}
	game := chess.NewGame(fenOption)
	uciNotation := chess.UCINotation{}
	algebraic := chess.AlgebraicNotation{}
	result := sanlineOutput{StartFEN: startFEN, Moves: make([]sanlineMove, 0, len(sanMoves))}
	for index, san := range sanMoves {
		san = strings.TrimSpace(san)
		if san == "" {
			_, _ = fmt.Fprintf(stderr, "move %d is empty\n", index+1)
			return 1
		}
		move, err := algebraic.Decode(game.Position(), san)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "move %d %q: %v\n", index+1, san, err)
			return 1
		}
		uci := uciNotation.Encode(game.Position(), move)
		if err := game.PushMove(san, nil); err != nil {
			_, _ = fmt.Fprintf(stderr, "move %d %q: %v\n", index+1, san, err)
			return 1
		}
		result.Moves = append(result.Moves, sanlineMove{
			Index: index + 1,
			SAN:   san,
			UCI:   uci,
			FEN:   game.Position().String(),
		})
	}
	result.FinalFEN = game.Position().String()
	if err := writeIndentedJSON(stdout, result); err != nil {
		_, _ = fmt.Fprintf(stderr, "write sanline result: %v\n", err)
		return 1
	}
	return 0
}

func writeIndentedJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
