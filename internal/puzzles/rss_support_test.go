package puzzles

func maximumRSSBytes(goos string, raw uint64) uint64 {
	if goos == "linux" {
		return raw << 10
	}
	return raw
}
