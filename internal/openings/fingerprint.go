package openings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type fingerprintMove struct {
	ID          string `json:"id"`
	UCI         string `json:"uci"`
	Destination string `json:"destination"`
}

func promptFingerprint(
	promptID string,
	position string,
	primary fingerprintMove,
	alternatives []fingerprintMove,
) string {
	ordered := append([]fingerprintMove(nil), alternatives...)
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].ID != ordered[right].ID {
			return ordered[left].ID < ordered[right].ID
		}
		if ordered[left].UCI != ordered[right].UCI {
			return ordered[left].UCI < ordered[right].UCI
		}
		return ordered[left].Destination < ordered[right].Destination
	})
	payload := struct {
		PromptID     string            `json:"promptId"`
		Position     string            `json:"position"`
		Primary      fingerprintMove   `json:"primary"`
		Alternatives []fingerprintMove `json:"alternatives"`
	}{
		PromptID: promptID, Position: position, Primary: primary, Alternatives: ordered,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (c *courseCompiler) compileFingerprints() {
	for _, sourcePrompt := range c.pack.Prompts {
		compiledPrompt, exists := c.compiled.Prompts[sourcePrompt.PromptID]
		if !exists {
			continue
		}
		position, positionOK := c.compiled.Positions[sourcePrompt.PositionID]
		primary, primaryOK := c.compiled.Moves[sourcePrompt.PrimaryMoveID]
		if !positionOK || position.FEN == "" || !primaryOK {
			continue
		}
		canonicalPosition, err := CanonicalPosition(position.FEN)
		if err != nil {
			continue
		}
		primaryDestination, ok := c.compiled.Positions[primary.ToPositionID]
		if !ok || primaryDestination.FEN == "" {
			continue
		}
		canonicalDestination, err := CanonicalPosition(primaryDestination.FEN)
		if err != nil {
			continue
		}
		alternatives := make([]fingerprintMove, 0, len(sourcePrompt.AcceptedAlternativeMoveIDs))
		valid := true
		for _, moveID := range sourcePrompt.AcceptedAlternativeMoveIDs {
			move, moveOK := c.compiled.Moves[moveID]
			destination, destinationOK := c.compiled.Positions[move.ToPositionID]
			if !moveOK || !destinationOK || destination.FEN == "" {
				valid = false
				break
			}
			canonical, canonicalErr := CanonicalPosition(destination.FEN)
			if canonicalErr != nil {
				valid = false
				break
			}
			alternatives = append(alternatives, fingerprintMove{ID: moveID, UCI: move.UCI, Destination: canonical})
		}
		if !valid {
			continue
		}
		compiledPrompt.SemanticFingerprint = promptFingerprint(
			sourcePrompt.PromptID,
			canonicalPosition,
			fingerprintMove{ID: primary.MoveID, UCI: primary.UCI, Destination: canonicalDestination},
			alternatives,
		)
		c.compiled.Prompts[sourcePrompt.PromptID] = compiledPrompt
	}
}
