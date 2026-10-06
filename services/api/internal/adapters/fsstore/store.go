package fsstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct{ dir string }

func New(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Put(_ context.Context, name, _ string, body io.Reader) error {
	path := filepath.Join(s.dir, filepath.FromSlash(name))
	rel, err := filepath.Rel(s.dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("caminho fora do diretório do dump: %q", name)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
