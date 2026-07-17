package buildinfo

import "regexp"

const (
	Name          = "Chess Trainer"
	RepositoryURL = "https://github.com/xang1234/chess"
)

var Commit = "development"

type Info struct {
	Name      string `json:"name"`
	Commit    string `json:"commit"`
	SourceURL string `json:"sourceUrl"`
}

var exactCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

func Current() Info {
	commit := Commit
	sourceURL := RepositoryURL
	if exactCommit.MatchString(commit) {
		sourceURL += "/tree/" + commit
	} else {
		commit = "development"
	}
	return Info{Name: Name, Commit: commit, SourceURL: sourceURL}
}
