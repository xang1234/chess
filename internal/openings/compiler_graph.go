package openings

import (
	"fmt"
	"strings"
)

func (c *courseCompiler) compileGraph() {
	c.detectCycles()
	root, rootOK := c.compiled.Positions[c.pack.RootPositionID]
	if !rootOK || c.rules == nil {
		return
	}
	if _, err := CanonicalPosition(c.pack.RootFEN); err != nil {
		c.addDiagnostic("invalid_root_fen", "rootFen", err.Error())
		return
	}
	if _, err := c.rules.LegalMoves(c.pack.RootFEN); err != nil {
		c.addDiagnostic("invalid_root_fen", "rootFen", err.Error())
		return
	}
	root.FEN = c.pack.RootFEN
	c.compiled.Positions[root.ID] = root

	queue := []string{root.ID}
	queued := map[string]bool{root.ID: true}
	derivedBy := map[string]string{root.ID: "rootFen"}
	for len(queue) > 0 {
		positionID := queue[0]
		queue = queue[1:]
		position := c.compiled.Positions[positionID]
		for _, moveID := range c.compiled.Outgoing[positionID] {
			move := c.compiled.Moves[moveID]
			path := c.basePath(c.movePaths, moveID, "move")
			nextFEN, err := c.rules.ApplyUCI(position.FEN, move.UCI)
			if err != nil {
				c.addDiagnostic("illegal_move", path+".uci", fmt.Sprintf("cannot apply %q from %s: %v", move.UCI, positionID, err))
				continue
			}
			san, err := c.rules.SAN(position.FEN, move.UCI)
			if err != nil {
				c.addDiagnostic("illegal_move", path+".uci", fmt.Sprintf("cannot derive SAN for %q: %v", move.UCI, err))
				continue
			}
			move.SAN = san
			c.compiled.Moves[moveID] = move

			destination := c.compiled.Positions[move.ToPositionID]
			if destination.FEN == "" {
				destination.FEN = nextFEN
				c.compiled.Positions[destination.ID] = destination
				derivedBy[destination.ID] = moveID
				if !queued[destination.ID] {
					queued[destination.ID] = true
					queue = append(queue, destination.ID)
				}
				continue
			}
			existingCanonical, existingErr := CanonicalPosition(destination.FEN)
			nextCanonical, nextErr := CanonicalPosition(nextFEN)
			if existingErr != nil || nextErr != nil {
				c.addDiagnostic("invalid_derived_fen", path+".toPositionId", fmt.Sprintf("cannot compare derived position: %v %v", existingErr, nextErr))
				continue
			}
			if existingCanonical != nextCanonical {
				c.addDiagnostic(
					"inconsistent_transposition",
					path+".toPositionId",
					fmt.Sprintf("move %q derives %q but %q previously derived %q", moveID, nextCanonical, derivedBy[destination.ID], existingCanonical),
				)
			}
		}
	}

	for _, position := range c.pack.Positions {
		compiled, ok := c.compiled.Positions[position.PositionID]
		if ok && compiled.FEN == "" {
			path := c.basePath(c.positionPaths, position.PositionID, "position")
			c.addDiagnostic("unreachable_position", path+".positionId", fmt.Sprintf("position %q is not reachable from the root", position.PositionID))
		}
	}
}

func (c *courseCompiler) detectCycles() {
	colors := map[string]uint8{}
	var visit func(string)
	visit = func(positionID string) {
		colors[positionID] = 1
		for _, moveID := range c.compiled.Outgoing[positionID] {
			move := c.compiled.Moves[moveID]
			switch colors[move.ToPositionID] {
			case 0:
				visit(move.ToPositionID)
			case 1:
				path := c.basePath(c.movePaths, moveID, "move")
				c.addDiagnostic("cycle", path+".toPositionId", fmt.Sprintf("move %q closes a cycle at position %q", moveID, move.ToPositionID))
			}
		}
		colors[positionID] = 2
	}
	for _, position := range c.pack.Positions {
		if _, exists := c.compiled.Positions[position.PositionID]; exists && colors[position.PositionID] == 0 {
			visit(position.PositionID)
		}
	}
}

func (c *courseCompiler) validateDepthsAndRoles() {
	for _, selected := range []Depth{DepthQuick, DepthStandard, DepthReference} {
		selectedRank, _ := depthRank(selected)
		reachable := map[string]bool{}
		if _, rootOK := c.compiled.Positions[c.pack.RootPositionID]; rootOK {
			reachable[c.pack.RootPositionID] = true
		}
		queue := []string{}
		if reachable[c.pack.RootPositionID] {
			queue = append(queue, c.pack.RootPositionID)
		}
		for len(queue) > 0 {
			positionID := queue[0]
			queue = queue[1:]
			for _, moveID := range c.compiled.Outgoing[positionID] {
				move := c.compiled.Moves[moveID]
				minimumRank, valid := depthRank(move.MinimumDepth)
				if !valid || minimumRank > selectedRank {
					continue
				}
				if !reachable[move.ToPositionID] {
					reachable[move.ToPositionID] = true
					queue = append(queue, move.ToPositionID)
				}
			}
		}
		c.reachableByDepth[selected] = reachable
		for _, sourceMove := range c.pack.Moves {
			move, exists := c.compiled.Moves[sourceMove.MoveID]
			if !exists {
				continue
			}
			minimumRank, valid := depthRank(move.MinimumDepth)
			if valid && minimumRank <= selectedRank && !reachable[move.FromPositionID] {
				path := c.basePath(c.movePaths, move.MoveID, "move")
				c.addDiagnostic("depth_dependency", path+".minimumDepth", fmt.Sprintf("%s move depends on a parent hidden at %s depth", move.MinimumDepth, selected))
			}
		}
	}

	for _, sourcePosition := range c.pack.Positions {
		position, exists := c.compiled.Positions[sourcePosition.PositionID]
		if !exists || position.FEN == "" {
			continue
		}
		fields := strings.Fields(position.FEN)
		if len(fields) < 2 {
			continue
		}
		learnerToMove := (fields[1] == "w" && c.pack.Perspective == PerspectiveWhite) ||
			(fields[1] == "b" && c.pack.Perspective == PerspectiveBlack)
		repertoire := 0
		for _, moveID := range c.compiled.Outgoing[position.ID] {
			move := c.compiled.Moves[moveID]
			path := c.basePath(c.movePaths, moveID, "move")
			if learnerToMove {
				if move.TrainingRole == RoleOpponent {
					c.addDiagnostic("role_perspective", path+".trainingRole", "opponent role is not allowed when the learner is to move")
				}
				if move.TrainingRole == RoleRepertoire {
					repertoire++
					if repertoire > 1 {
						c.addDiagnostic("multiple_repertoire_moves", path+".trainingRole", fmt.Sprintf("position %q has more than one repertoire move", position.ID))
					}
				}
			} else if move.TrainingRole != RoleOpponent {
				c.addDiagnostic("role_perspective", path+".trainingRole", "only opponent role is allowed when the opponent is to move")
			}
		}
	}
}

func (c *courseCompiler) basePath(paths map[string]string, id, fallback string) string {
	path := paths[id]
	if path == "" {
		return fallback
	}
	if separator := strings.LastIndexByte(path, '.'); separator >= 0 {
		return path[:separator]
	}
	return path
}
