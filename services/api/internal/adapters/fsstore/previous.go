package fsstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type PreviousDump struct{ dir string }

func NewPreviousDump(dir string) *PreviousDump { return &PreviousDump{dir: dir} }

func (p *PreviousDump) Table(_ context.Context, name string) ([]string, [][]string, error) {
	file, err := p.manifestEntry(name + ".csv.gz")
	if err != nil {
		return nil, nil, err
	}
	gz, err := os.ReadFile(filepath.Join(p.dir, file.Name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, domain.ErrNoPreviousDump
	}
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(gz)
	if hex.EncodeToString(sum[:]) != file.SHA256 {
		return nil, nil, fmt.Errorf("%s: sha256 não confere com o manifest.json", file.Name)
	}
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", file.Name, err)
	}
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", file.Name, err)
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("%s: arquivo sem cabeçalho", file.Name)
	}
	return records[0], records[1:], nil
}

func (p *PreviousDump) manifestEntry(name string) (domain.DumpFile, error) {
	body, err := os.ReadFile(filepath.Join(p.dir, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return domain.DumpFile{}, domain.ErrNoPreviousDump
	}
	if err != nil {
		return domain.DumpFile{}, err
	}
	var manifest domain.DumpManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return domain.DumpFile{}, fmt.Errorf("manifest.json: %w", err)
	}
	for _, f := range manifest.Files {
		if f.Name == name {
			return f, nil
		}
	}
	return domain.DumpFile{}, domain.ErrNoPreviousDump
}
