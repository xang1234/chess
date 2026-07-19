package importing

type Format string

type SourceIDOrigin string

const (
	SourceIDFixed    SourceIDOrigin = "fixed"
	SourceIDEmbedded SourceIDOrigin = "embedded"
	SourceIDPath     SourceIDOrigin = "path"
)

type Inspection struct {
	Path             string         `json:"path"`
	Filename         string         `json:"filename"`
	Format           Format         `json:"format"`
	FormatLabel      string         `json:"formatLabel"`
	SourceID         string         `json:"sourceId"`
	SourceIDOrigin   SourceIDOrigin `json:"sourceIdOrigin"`
	SourceName       string         `json:"sourceName,omitempty"`
	URL              string         `json:"url,omitempty"`
	Attribution      string         `json:"attribution,omitempty"`
	ReplacesExisting bool           `json:"replacesExisting"`
}

type Phase string

const (
	PhaseDetecting  Phase = "detecting"
	PhaseParsing    Phase = "parsing"
	PhaseSealing    Phase = "sealing"
	PhaseActivating Phase = "activating"
)

type Progress struct {
	Phase      Phase `json:"phase"`
	RowsRead   int64 `json:"rowsRead"`
	BytesRead  int64 `json:"bytesRead"`
	TotalBytes int64 `json:"totalBytes"`
}

type ProgressSink func(Progress)

type Rejection struct {
	Ordinal int64  `json:"ordinal"`
	Reason  string `json:"reason"`
}

type Report struct {
	Accepted   int64            `json:"accepted"`
	Duplicates int64            `json:"duplicates"`
	Rejected   int64            `json:"rejected"`
	Examples   []Rejection      `json:"examples"`
	Counts     map[string]int64 `json:"counts,omitempty"`
}
