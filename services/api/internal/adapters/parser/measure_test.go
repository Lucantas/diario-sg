package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

// Edições reais (não versionadas; ./scripts/fetch-editions.sh). Para as
// edições contadas à mão registramos o total esperado de atos.
var expectedActs = map[string]int{
	"2026_09_18": 51,
}

func TestMeasureRealEditions(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "editions", "*.txt"))
	if len(files) == 0 {
		t.Skip("sem edições reais em testdata/editions (rode ./scripts/fetch-editions.sh)")
	}
	sort.Strings(files)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".txt")
		acts := New().Parse(string(raw))
		counts := map[domain.ActType]int{}
		for _, a := range acts {
			counts[a.Type]++
		}
		t.Logf("%s: %d atos, %d outro (%.0f%%) — %s", name, len(acts), counts[domain.ActOutro],
			100*float64(counts[domain.ActOutro])/float64(max(len(acts), 1)), summary(counts))
		if os.Getenv("PARSER_DUMP") != "" {
			for _, a := range acts {
				fmt.Printf("  [%s] %-10s %s\n", name, a.Type, a.Title)
			}
		}
		if want, ok := expectedActs[name]; ok && len(acts) != want {
			t.Errorf("%s: esperava %d atos (contagem manual), veio %d", name, want, len(acts))
		}
	}
}

func summary(counts map[domain.ActType]int) string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, counts[domain.ActType(k)]))
	}
	return strings.Join(parts, " ")
}
