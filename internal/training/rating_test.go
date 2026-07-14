package training

import "testing"

func TestUpdateRating(t *testing.T) {
	if got := UpdateRating(1500, 1500, 1, 400, 3000); got != 1512 {
		t.Fatalf("got %v", got)
	}
	if got := UpdateRating(1500, 1500, 0, 400, 3000); got != 1488 {
		t.Fatalf("got %v", got)
	}
}

func TestUpdateRatingRejectsUnsupportedScore(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("UpdateRating() did not reject score 0.25")
		}
	}()
	UpdateRating(1500, 1500, 0.25, 400, 3000)
}
