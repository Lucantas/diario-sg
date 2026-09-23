package domain

import "time"

type FetchRun struct {
	ID            string
	Source        string
	RequestedFrom time.Time
	RequestedTo   time.Time
	Found         int
	Stored        int
	Skipped       int
	Failed        int
	Error         string
	StartedAt     time.Time
	FinishedAt    time.Time
}

func (r FetchRun) Valid() bool {
	return r.ID != "" && r.Source != "" &&
		r.Found >= 0 && r.Stored >= 0 && r.Skipped >= 0 && r.Failed >= 0 &&
		!r.FinishedAt.Before(r.StartedAt) && !r.RequestedTo.Before(r.RequestedFrom)
}
