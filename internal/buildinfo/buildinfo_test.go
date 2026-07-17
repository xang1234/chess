package buildinfo

import "testing"

func TestCurrentUsesOnlyAnExactLowercaseCommit(t *testing.T) {
	validCommit := "0123456789abcdef0123456789abcdef01234567"
	tests := []struct {
		name       string
		commit     string
		wantCommit string
		wantSource string
	}{
		{
			name:       "exact lowercase commit",
			commit:     validCommit,
			wantCommit: validCommit,
			wantSource: RepositoryURL + "/tree/" + validCommit,
		},
		{name: "development", commit: "development", wantCommit: "development", wantSource: RepositoryURL},
		{name: "uppercase", commit: "0123456789ABCDEF0123456789ABCDEF01234567", wantCommit: "development", wantSource: RepositoryURL},
		{name: "abbreviated", commit: "0123456789ab", wantCommit: "development", wantSource: RepositoryURL},
		{name: "malformed", commit: "not-a-commit", wantCommit: "development", wantSource: RepositoryURL},
	}

	original := Commit
	t.Cleanup(func() { Commit = original })
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Commit = test.commit
			got := Current()
			if got.Name != Name || got.Commit != test.wantCommit || got.SourceURL != test.wantSource {
				t.Fatalf("Current() = %#v, want name %q commit %q source %q", got, Name, test.wantCommit, test.wantSource)
			}
		})
	}
}
