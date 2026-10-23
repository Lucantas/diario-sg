package usecase

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

var (
	actsHeader     = []string{"id", "gazette_id", "body"}
	entitiesHeader = []string{"act_id", "kind"}
)

type fakePrevious struct {
	tables map[string][][]string
	err    error
}

func (f fakePrevious) Table(_ context.Context, name string) ([]string, [][]string, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	t := f.tables[name]
	return t[0], t[1:], nil
}

func previousWith(acts, entities [][]string) fakePrevious {
	return fakePrevious{tables: map[string][][]string{
		"acts":         append([][]string{actsHeader}, acts...),
		"act_entities": append([][]string{entitiesHeader}, entities...),
	}}
}

func incrementalSnapshot(fingerprints []domain.GazetteFingerprint, records map[string][]domain.DumpRecord) *fakeSnapshot {
	return &fakeSnapshot{
		order:        []string{"gazettes", "acts", "act_entities"},
		tables:       map[string]string{"gazettes": "id\ng1\n"},
		headers:      map[string][]string{"acts": actsHeader, "act_entities": entitiesHeader},
		fingerprints: fingerprints,
		records:      records,
	}
}

func fingerprint(id string, acts, entities [][]string) domain.GazetteFingerprint {
	return domain.GazetteFingerprint{GazetteID: id, Hash: domain.FingerprintOf(acts, entities)}
}

func TestIncrementalDumpReusesUnchangedGazettesAndFetchesOnlyStaleOnes(t *testing.T) {
	previous := previousWith(
		[][]string{{"a1", "g1", "um"}, {"a2", "g2", "dois"}},
		[][]string{{"a1", "cnpj"}},
	)
	src := incrementalSnapshot(
		[]domain.GazetteFingerprint{
			fingerprint("g1", [][]string{{"a1", "g1", "um"}}, [][]string{{"a1", "cnpj"}}),
			{GazetteID: "g2", Hash: "mudou"},
			{GazetteID: "g3", Hash: "nova"},
		},
		map[string][]domain.DumpRecord{
			"acts": {
				{GazetteID: "g2", Fields: []string{"a2", "g2", "dois, revisto"}},
				{GazetteID: "g3", Fields: []string{"a3", "g3", "três"}},
			},
			"act_entities": {{GazetteID: "g3", Fields: []string{"a3", "cpf"}}},
		},
	)
	objects := &memObjects{}

	report, err := NewPublishDump(src, objects, previous).Execute(context.Background(), time.Now())

	if err != nil {
		t.Fatal(err)
	}
	if got := gunzip(t, objects.data["latest/acts.csv.gz"]); got != "id,gazette_id,body\na1,g1,um\na2,g2,\"dois, revisto\"\na3,g3,três\n" {
		t.Fatalf("atos inesperados: %q", got)
	}
	if got := gunzip(t, objects.data["latest/act_entities.csv.gz"]); got != "act_id,kind\na1,cnpj\na3,cpf\n" {
		t.Fatalf("entidades inesperadas: %q", got)
	}
	want := []string{"g2", "g3"}
	if !reflect.DeepEqual(src.asked["acts"], want) || !reflect.DeepEqual(src.asked["act_entities"], want) {
		t.Fatalf("deveria buscar só g2 e g3: %v", src.asked)
	}
	if !reflect.DeepEqual(src.wrote, []string{"gazettes"}) {
		t.Fatalf("só a tabela de edições deve vir inteira do banco: %v", src.wrote)
	}
	if !report.Incremental || report.Gazettes != 3 || report.FetchedAgain != 2 || report.Manifest.Files[1].Rows != 3 {
		t.Fatalf("relatório inesperado: %+v", report)
	}
}

func TestIncrementalDumpDropsGazettesNoLongerInTheDatabase(t *testing.T) {
	previous := previousWith([][]string{{"a1", "g1", "um"}, {"a2", "g2", "dois"}}, nil)
	src := incrementalSnapshot([]domain.GazetteFingerprint{fingerprint("g1", [][]string{{"a1", "g1", "um"}}, nil)}, nil)
	objects := &memObjects{}

	_, err := NewPublishDump(src, objects, previous).Execute(context.Background(), time.Now())

	if err != nil {
		t.Fatal(err)
	}
	if got := gunzip(t, objects.data["latest/acts.csv.gz"]); got != "id,gazette_id,body\na1,g1,um\n" {
		t.Fatalf("atos inesperados: %q", got)
	}
	if src.asked != nil {
		t.Fatalf("nada mudou, não deveria buscar atos: %v", src.asked)
	}
}

func TestDumpFallsBackToFullWhenPreviousCannotBeReused(t *testing.T) {
	cases := map[string]fakePrevious{
		"sem dump anterior": {err: domain.ErrNoPreviousDump},
		"cabeçalho mudou": {tables: map[string][][]string{
			"acts":         {{"id", "gazette_id"}},
			"act_entities": {entitiesHeader},
		}},
	}
	for name, previous := range cases {
		t.Run(name, func(t *testing.T) {
			src := incrementalSnapshot(nil, nil)
			src.tables["acts"] = "id\n"
			src.tables["act_entities"] = "act_id\n"

			report, err := NewPublishDump(src, &memObjects{}, previous).Execute(context.Background(), time.Now())

			if err != nil {
				t.Fatal(err)
			}
			if report.Incremental || report.FallbackCause == "" {
				t.Fatalf("esperava dump completo com motivo: %+v", report)
			}
			if !reflect.DeepEqual(src.wrote, []string{"gazettes", "acts", "act_entities"}) {
				t.Fatalf("todas as tabelas deveriam vir do banco: %v", src.wrote)
			}
		})
	}
}

func TestDumpFailsWhenPreviousDumpIsUnreadable(t *testing.T) {
	src := incrementalSnapshot(nil, nil)
	objects := &memObjects{}

	_, err := NewPublishDump(src, objects, fakePrevious{err: errors.New("sha256 não confere")}).Execute(context.Background(), time.Now())

	if err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("esperava erro do dump anterior, veio %v", err)
	}
	if len(objects.names) != 0 {
		t.Fatalf("nada deveria ser publicado: %v", objects.names)
	}
}
