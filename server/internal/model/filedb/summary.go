package filedb

type SummaryPart struct {
	Index   int      `json:"index"`
	Titles  []string `json:"titles"`
	Content string   `json:"content"`
}

type Summary struct {
	Pars    []SummaryPart `json:"pars"`
	Summary string        `json:"summary"`
}
