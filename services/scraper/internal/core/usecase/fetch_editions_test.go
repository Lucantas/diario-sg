package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/scraper/internal/core/domain"
)

type fakeSource struct {
	editions []domain.Edition
	from, to *time.Time
}

func (f fakeSource) ListEditions(_ context.Context, from, to time.Time) ([]domain.Edition, error) {
	if f.from != nil {
		*f.from, *f.to = from, to
	}
	return f.editions, nil
}
func (f fakeSource) Download(context.Context, domain.Edition) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("%PDF-1.7 fake"))), nil
}

type fakeStorage struct{ objects map[string][]byte }

func (f *fakeStorage) Exists(_ context.Context, p string) (bool, error) {
	_, ok := f.objects[p]
	return ok, nil
}
func (f *fakeStorage) Put(_ context.Context, p, _ string, r io.Reader) error {
	b, _ := io.ReadAll(r)
	f.objects[p] = b
	return nil
}

type fakePublisher struct {
	events []domain.FetchedEdition
	fail   bool
}

func (f *fakePublisher) EditionFetched(_ context.Context, e domain.FetchedEdition) error {
	if f.fail {
		return errors.New("pubsub fora do ar")
	}
	f.events = append(f.events, e)
	return nil
}

func TestFetchEditions_IsIdempotent(t *testing.T) {
	day := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	src := fakeSource{editions: []domain.Edition{{Number: "1234", PublishedAt: day, URL: "https://exemplo/1234.pdf"}}}
	st := &fakeStorage{objects: map[string][]byte{}}
	pub := &fakePublisher{}
	uc := NewFetchEditions(src, st, pub)

	res, err := uc.Execute(context.Background(), 72*time.Hour)
	if err != nil || res.Stored != 1 || len(pub.events) != 1 {
		t.Fatalf("primeira execução: res=%+v err=%v eventos=%d", res, err, len(pub.events))
	}
	if pub.events[0].StoragePath != "gazettes/2026/09/18/edicao-1234.pdf" {
		t.Errorf("caminho inesperado: %s", pub.events[0].StoragePath)
	}

	res, err = uc.Execute(context.Background(), 72*time.Hour)
	if err != nil || res.Skipped != 1 || len(pub.events) != 1 {
		t.Fatalf("segunda execução deveria pular: res=%+v err=%v", res, err)
	}
}

func TestFetchEditions_RetriesWhenPublishFails(t *testing.T) {
	src := fakeSource{editions: []domain.Edition{{Number: "9", PublishedAt: time.Now(), URL: "u"}}}
	st := &fakeStorage{objects: map[string][]byte{}}
	pub := &fakePublisher{fail: true}
	uc := NewFetchEditions(src, st, pub)

	if res, err := uc.Execute(context.Background(), time.Hour); err == nil || res.Failed != 1 {
		t.Fatalf("esperava falha: res=%+v err=%v", res, err)
	}
	pub.fail = false
	if res, err := uc.Execute(context.Background(), time.Hour); err != nil || res.Stored != 1 {
		t.Fatalf("esperava nova tentativa com sucesso: res=%+v err=%v", res, err)
	}
}

func TestExecuteRangePassesPeriodToSourceAndRejectsInvertedPeriod(t *testing.T) {
	var from, to time.Time
	src := fakeSource{from: &from, to: &to}
	uc := NewFetchEditions(src, &fakeStorage{objects: map[string][]byte{}}, &fakePublisher{})
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	if _, err := uc.ExecuteRange(context.Background(), start, end); err != nil {
		t.Fatal(err)
	}
	if !from.Equal(start) || !to.Equal(end) {
		t.Errorf("período repassado errado: %s..%s", from, to)
	}
	if _, err := uc.ExecuteRange(context.Background(), end, start); err == nil {
		t.Error("período invertido deve dar erro")
	}
}
