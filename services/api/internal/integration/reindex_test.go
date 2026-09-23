//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/entities"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/parser"
	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/migrations"
)

func TestReindexReplacesActsWithoutPublishing(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	ctx := context.Background()
	db := openTestDB(t, ctx, url)
	defer db.Close()
	if _, err := postgres.Migrate(ctx, db, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, resetTables); err != nil {
		t.Fatal(err)
	}
	gaz := postgres.NewGazetteRepo(db)
	pub := &recPub{}
	published := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	in := usecase.IndexGazetteInput{EditionNumber: "1", PublishedAt: published,
		SourceURL: "https://exemplo/1.pdf", StoragePath: "1.pdf", Checksum: strings.Repeat("d", 64)}
	if err := usecase.NewIndexGazette(gaz, textStore{}, passthroughExtractor{}, parser.New(), entities.New(), pub).Execute(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE acts SET page_start = NULL, page_end = NULL, organ = 'VELHO'`); err != nil {
		t.Fatal(err)
	}
	var entitiesBefore int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM act_entities`).Scan(&entitiesBefore); err != nil {
		t.Fatal(err)
	}

	res, err := usecase.NewReindexGazettes(gaz, textStore{}, passthroughExtractor{}, parser.New(), entities.New()).
		Execute(ctx, published, published)

	if err != nil || res != (usecase.ReindexResult{Found: 1, Reindexed: 1}) {
		t.Fatalf("reindexação: %+v %v", res, err)
	}
	var acts, stale, entitiesAfter int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*), count(*) FILTER (WHERE page_start IS NULL OR organ = 'VELHO'),
		       (SELECT count(*) FROM act_entities)
		FROM acts`).Scan(&acts, &stale, &entitiesAfter); err != nil {
		t.Fatal(err)
	}
	if acts != 5 || stale != 0 || entitiesAfter != entitiesBefore || entitiesAfter == 0 {
		t.Fatalf("esperava 5 atos refeitos com entidades (%d), veio atos=%d velhos=%d entidades=%d",
			entitiesBefore, acts, stale, entitiesAfter)
	}
	if len(pub.ids) != 1 {
		t.Fatalf("reindexação não pode publicar gazette.indexed; eventos: %v", pub.ids)
	}
}
