package domain

import "time"

type Coverage struct {
	First         time.Time
	Last          time.Time
	LastIndexedAt time.Time
	Gazettes      int
	Acts          int
	LastRun       FetchRun
}
