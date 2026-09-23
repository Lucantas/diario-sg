package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"
)

type Edition struct {
	Source      string
	Number      string
	PublishedAt time.Time
	URL         string
}

func (e Edition) StoragePath() string {
	if e.Source == SourceDiarioCamara {
		day := e.PublishedAt.Format(time.DateOnly)
		return fmt.Sprintf("raw/%s/%s/%s.pdf", SourceDiarioCamara, e.PublishedAt.Format("2006/01/02"), day)
	}
	id := sanitize(e.Number)
	if id == "" {
		sum := sha256.Sum256([]byte(e.URL))
		id = hex.EncodeToString(sum[:])[:16]
	}
	return fmt.Sprintf("gazettes/%s/edicao-%s.pdf", e.PublishedAt.Format("2006/01/02"), id)
}

func (e Edition) MarkerPath() string { return e.StoragePath() + ".published" }

type FetchedEdition struct {
	Edition
	StoragePath    string
	ChecksumSHA256 string
	FetchedAt      time.Time
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '/' || r == ' ':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
