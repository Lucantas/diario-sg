package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

const SourceDiarioPrefeitura = "diario_prefeitura"

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

func NewRunID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
