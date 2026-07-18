package puzzles

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/corentings/chess/v2"
)

const (
	maxLucasFNSLineBytes     = 1 << 20
	maxLucasFNSMetadataBytes = 64 * 1024
)

var (
	lucasFNSDifficultyPattern = regexp.MustCompile(
		`(?i)\bdifficulty[[:space:]]*[:=-]?[[:space:]]*(\*+|[0-9]+\b)`,
	)
	lucasFNSCamelBoundary   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	lucasFNSAcronymBoundary = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	lucasFNSPunctuation     = regexp.MustCompile(`[^A-Za-z0-9]+`)
	lucasFNSGenericThemes   = map[string]struct{}{
		"training": {},
		"tactics":  {},
		"puzzles":  {},
	}
)

type lucasFNSAdapter struct {
	rules chessrules.Rules
}

func NewLucasFNSAdapter(rules chessrules.Rules) PuzzleAdapter {
	return lucasFNSAdapter{rules: rules}
}

func (lucasFNSAdapter) Format() ImportFormat {
	return FormatLucasFNS
}

func (a lucasFNSAdapter) Inspect(
	ctx context.Context,
	path string,
) (ImportInspection, bool, error) {
	if err := ctx.Err(); err != nil {
		return ImportInspection{}, false, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ImportInspection{}, false, err
	}
	defer file.Close()

	scanner := newLucasFNSLineScanner(contextReader{ctx: ctx, reader: file})
	lineNumber := int64(0)
	for scanner.Scan() {
		lineNumber++
		if err := ctx.Err(); err != nil {
			return ImportInspection{}, false, err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.SplitN(line, "|", 3)
		if len(fields) != 3 {
			continue
		}
		if _, err := normalizeFEN(a.rules, strings.TrimSpace(fields[0])); err != nil {
			continue
		}
		return ImportInspection{
			SourceID:       path,
			SourceIDOrigin: SourceIDPath,
		}, true, nil
	}
	if err := scanner.Err(); err != nil {
		return ImportInspection{}, false, fmt.Errorf(
			"inspect Lucas FNS line %d: %w",
			lineNumber+1,
			err,
		)
	}
	return ImportInspection{}, false, nil
}

func (a lucasFNSAdapter) NewDecoder(
	reader io.Reader,
	inspection ImportInspection,
) (PuzzleDecoder, error) {
	if reader == nil {
		return nil, errors.New("Lucas FNS reader is required")
	}
	sourceID := strings.TrimSpace(inspection.SourceID)
	if sourceID == "" {
		return nil, errors.New("Lucas FNS source ID is required")
	}
	filename := strings.TrimSpace(inspection.Filename)
	if filename == "" {
		filename = filepath.Base(inspection.Path)
	}
	return &lucasFNSDecoder{
		rules:    a.rules,
		sourceID: sourceID,
		theme:    lucasFNSFilenameTheme(filename),
		scanner:  newLucasFNSLineScanner(reader),
	}, nil
}

func newLucasFNSLineScanner(reader io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxLucasFNSLineBytes)
	return scanner
}

type lucasFNSDecoder struct {
	rules      chessrules.Rules
	sourceID   string
	theme      string
	scanner    *bufio.Scanner
	lineNumber int64
	closed     bool
}

func (d *lucasFNSDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.closed {
		return DecodedRecord{}, io.EOF
	}
	for d.scanner.Scan() {
		d.lineNumber++
		if err := ctx.Err(); err != nil {
			return DecodedRecord{}, err
		}
		line := strings.TrimSpace(d.scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		puzzle, err := d.decodeLine(line)
		if err != nil {
			return lucasFNSRejection(d.lineNumber, err), nil
		}
		return DecodedRecord{Puzzle: &puzzle}, nil
	}
	if err := d.scanner.Err(); err != nil {
		return DecodedRecord{}, fmt.Errorf(
			"scan Lucas FNS line %d: %w",
			d.lineNumber+1,
			err,
		)
	}
	return DecodedRecord{}, io.EOF
}

func (d *lucasFNSDecoder) decodeLine(line string) (TrainingPuzzle, error) {
	fields := strings.SplitN(line, "|", 3)
	if len(fields) != 3 {
		return TrainingPuzzle{}, errors.New("line must contain FEN, description, and movetext fields")
	}
	fen := strings.TrimSpace(fields[0])
	description := strings.TrimSpace(fields[1])
	movetext := strings.TrimSpace(fields[2])
	if movetext == "" {
		return TrainingPuzzle{}, errors.New("movetext is required")
	}

	normalizedFEN, err := normalizeFEN(d.rules, fen)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("normalize FEN: %w", err)
	}
	solver, err := solverFromFEN(normalizedFEN)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	game, err := parseLucasFNSMovetext(normalizedFEN, movetext)
	if err != nil {
		return TrainingPuzzle{}, err
	}

	total := 0
	solution, err := lucasFNSMoveNodes(game.GetRootMove().Children(), 1, &total)
	if err != nil {
		return TrainingPuzzle{}, err
	}
	if len(solution) == 0 {
		return TrainingPuzzle{}, errors.New("movetext must contain at least one legal solver move")
	}
	core, err := finalizeCore(d.rules, normalizedFEN, solver, solution)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("validate Lucas FNS move tree: %w", err)
	}

	metadata := map[string]any{"description": description}
	if match := lucasFNSDifficultyPattern.FindStringSubmatch(description); len(match) == 2 {
		metadata["sourceDifficulty"] = match[1]
	}
	encodedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return TrainingPuzzle{}, fmt.Errorf("serialize Lucas FNS metadata: %w", err)
	}
	if len(encodedMetadata) > maxLucasFNSMetadataBytes {
		return TrainingPuzzle{}, fmt.Errorf(
			"metadata exceeds maximum of %d bytes",
			maxLucasFNSMetadataBytes,
		)
	}
	var themes []string
	if d.theme != "" {
		themes = []string{d.theme}
	}

	return TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			SourceID:   d.sourceID,
			SourceKind: string(FormatLucasFNS),
			ExternalID: strconv.FormatInt(d.lineNumber, 10),
			SourceFEN:  normalizedFEN,
			Metadata:   metadata,
			Themes:     themes,
			Ordinal:    d.lineNumber,
		},
	}, nil
}

func parseLucasFNSMovetext(fen string, movetext string) (*chess.Game, error) {
	rawPGN := fmt.Sprintf("[SetUp \"1\"]\n[FEN %q]\n\n%s", fen, movetext)
	scanner := chess.NewScanner(strings.NewReader(rawPGN))
	scanned, err := scanner.ScanGame()
	if errors.Is(err, io.EOF) {
		return nil, errors.New("movetext is required")
	}
	if err != nil {
		return nil, fmt.Errorf("scan Lucas FNS movetext: %w", err)
	}
	tokens, _, err := tokenizeTacticalPGNGame(scanned)
	if err != nil {
		return nil, fmt.Errorf("parse Lucas FNS movetext: %w", err)
	}
	if err := validateLucasFNSTerminalResult(tokens); err != nil {
		return nil, err
	}
	game, err := parseTacticalPGNTokens(tokens)
	if err != nil {
		return nil, fmt.Errorf("parse Lucas FNS movetext: %w", err)
	}
	if _, err := scanner.ScanGame(); err == nil {
		return nil, errors.New("movetext must contain exactly one PGN game")
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("scan trailing Lucas FNS movetext: %w", err)
	}
	return game, nil
}

func validateLucasFNSTerminalResult(tokens []chess.Token) error {
	for index, token := range tokens {
		if token.Type != chess.RESULT {
			continue
		}
		if index+1 != len(tokens) {
			return fmt.Errorf(
				"movetext contains token %q after result %q",
				tokens[index+1].Value,
				token.Value,
			)
		}
		return nil
	}
	return nil
}

func lucasFNSMoveNodes(
	moves []*chess.Move,
	depth int,
	total *int,
) ([]domain.MoveNode, error) {
	nodes := make([]domain.MoveNode, 0, len(moves))
	notation := chess.UCINotation{}
	for _, move := range moves {
		if depth > maxSolutionDepth {
			return nil, fmt.Errorf("solution depth exceeds maximum of %d", maxSolutionDepth)
		}
		*total++
		if *total > maxSolutionNodes {
			return nil, fmt.Errorf("solution exceeds maximum of %d nodes", maxSolutionNodes)
		}
		node := domain.MoveNode{
			UCI: strings.ToLower(notation.Encode(nil, move)),
		}
		if len(move.Children()) > 0 {
			children, err := lucasFNSMoveNodes(move.Children(), depth+1, total)
			if err != nil {
				return nil, err
			}
			node.Children = children
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func lucasFNSFilenameTheme(filename string) string {
	stem := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	stem = lucasFNSAcronymBoundary.ReplaceAllString(stem, "$1-$2")
	stem = lucasFNSCamelBoundary.ReplaceAllString(stem, "$1-$2")
	stem = lucasFNSPunctuation.ReplaceAllString(stem, "-")
	stem = strings.ToLower(strings.Trim(stem, "-"))
	if _, generic := lucasFNSGenericThemes[stem]; generic {
		return ""
	}
	return stem
}

func (d *lucasFNSDecoder) Close() error {
	d.closed = true
	return nil
}

func lucasFNSRejection(ordinal int64, err error) DecodedRecord {
	rejection := Rejection{Ordinal: ordinal, Reason: err.Error()}
	return DecodedRecord{Rejection: &rejection}
}
