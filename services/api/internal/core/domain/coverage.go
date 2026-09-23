package domain

import "time"

type Coverage struct {
	Source        string
	First         time.Time
	Last          time.Time
	LastIndexedAt time.Time
	Gazettes      int
	Acts          int
	LastRun       FetchRun
}
