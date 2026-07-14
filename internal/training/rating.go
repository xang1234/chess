package training

import "math"

const ratingKFactor = 24

func UpdateRating(current, puzzle float64, score float64, minimum, maximum float64) float64 {
	if score != 0 && score != 0.5 && score != 1 {
		panic("rating score must be 0, 0.5, or 1")
	}
	if minimum > maximum {
		panic("minimum rating exceeds maximum rating")
	}
	expected := 1 / (1 + math.Pow(10, (puzzle-current)/400))
	updated := math.Round(current + ratingKFactor*(score-expected))
	return min(max(updated, minimum), maximum)
}
