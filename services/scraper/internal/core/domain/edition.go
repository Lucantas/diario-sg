// Package domain contém as entidades do scraper. Não depende de nenhuma
// outra camada.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Edition é uma edição do Diário Oficial disponível na fonte.
type Edition struct {
	Number      string
	PublishedAt time.Time
	URL         string
}

// StoragePath é o caminho determinístico da edição no bucket. Ser
// determinístico é o que torna a coleta idempotente.
func (e Edition) StoragePath() string {
	id := sanitize(e.Number)
	if id == "" {
		sum := sha256.Sum256([]byte(e.URL))
		id = hex.EncodeToString(sum[:])[:16]
	}
	return fmt.Sprintf("gazettes/%s/edicao-%s.pdf", e.PublishedAt.Format("2006/01/02"), id)
}

// MarkerPath indica que a edição foi armazenada E o evento publicado.
func (e Edition) MarkerPath() string { return e.StoragePath() + ".published" }

// FetchedEdition é o fato de domínio "edição coletada com sucesso".
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
