package receita

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "tok"

func latin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		out = append(out, byte(r))
	}
	return out
}

func zipped(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const propfindRoot = `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
<d:response><d:href>/public.php/webdav/</d:href><d:propstat><d:prop></d:prop></d:propstat></d:response>
<d:response><d:href>/public.php/webdav/2026-08/</d:href><d:propstat><d:prop></d:prop></d:propstat></d:response>
<d:response><d:href>/public.php/webdav/2026-09/</d:href><d:propstat><d:prop></d:prop></d:propstat></d:response>
<d:response><d:href>/public.php/webdav/cnpj.tar.gz</d:href><d:propstat><d:prop><d:getcontentlength>10</d:getcontentlength></d:prop></d:propstat></d:response>
</d:multistatus>`

const propfindMonth = `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
<d:response><d:href>/public.php/webdav/2026-09/</d:href><d:propstat><d:prop></d:prop></d:propstat></d:response>
<d:response><d:href>/public.php/webdav/2026-09/Empresas0.zip</d:href><d:propstat><d:prop><d:getcontentlength>100</d:getcontentlength></d:prop></d:propstat></d:response>
<d:response><d:href>/public.php/webdav/2026-09/Socios0.zip</d:href><d:propstat><d:prop><d:getcontentlength>100</d:getcontentlength></d:prop></d:propstat></d:response>
</d:multistatus>`

type fakeShare struct {
	files       map[string][]byte
	rangeCalls  atomic.Int32
	failFirstN  atomic.Int32
	sawBadCreds atomic.Bool
}

func (s *fakeShare) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != testToken || pass != "" {
		s.sawBadCreds.Store(true)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/public.php/webdav/")
	if r.Method == "PROPFIND" {
		w.WriteHeader(http.StatusMultiStatus)
		if path == "" {
			_, _ = w.Write([]byte(propfindRoot))
		} else {
			_, _ = w.Write([]byte(propfindMonth))
		}
		return
	}
	body, ok := s.files[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("Range") != "" {
		s.rangeCalls.Add(1)
		if s.failFirstN.Load() > 0 {
			s.failFirstN.Add(-1)
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
	}
	http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(body))
}

func newTestSource(t *testing.T, share *fakeShare) *Source {
	t.Helper()
	srv := httptest.NewServer(share)
	t.Cleanup(srv.Close)
	src := New(srv.URL+"/public.php/webdav/", testToken, srv.Client())
	src.retryWait = time.Millisecond
	return src
}

func TestLatestMonthIsTheNewestMonthDirectory(t *testing.T) {
	src := newTestSource(t, &fakeShare{})

	month, err := src.LatestMonth(context.Background())

	if err != nil || month != "2026-09" {
		t.Fatalf("esperava 2026-09, veio %q (%v)", month, err)
	}
}

func TestFilesListsTheZipsOfTheMonth(t *testing.T) {
	src := newTestSource(t, &fakeShare{})

	files, err := src.Files(context.Background(), "2026-09")

	if err != nil || strings.Join(files, ",") != "Empresas0.zip,Socios0.zip" {
		t.Fatalf("veio %v (%v)", files, err)
	}
}

func TestRowsReadsTheZipByRangeAndDecodesLatin1(t *testing.T) {
	csv := latin1("\"07396865\";\"CONSTRUÇÃO SÃO GONÇALO LTDA\";\"2062\";\"49\";\"1000,00\";\"05\";\"\"\n\"12345678\";\"OUTRA\";\"2062\";\"49\";\"0,00\";\"01\";\"\"\n")
	share := &fakeShare{files: map[string][]byte{"2026-09/Empresas0.zip": zipped(t, "K3241.EMPRECSV", csv)}}
	src := newTestSource(t, share)
	var rows [][]string

	sum, err := src.Rows(context.Background(), "2026-09", "Empresas0.zip", func(f []string) error {
		rows = append(rows, append([]string(nil), f...))
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0][1] != "CONSTRUÇÃO SÃO GONÇALO LTDA" || len(rows[0]) != 7 {
		t.Fatalf("linhas: %q", rows)
	}
	want := sha256.Sum256(csv)
	if sum != hex.EncodeToString(want[:]) {
		t.Errorf("sha do CSV: %s", sum)
	}
	if share.rangeCalls.Load() == 0 {
		t.Error("o zip deveria ser lido por Range")
	}
}

func TestRowsRetriesATransientFailure(t *testing.T) {
	share := &fakeShare{files: map[string][]byte{"2026-09/Socios0.zip": zipped(t, "S.SOCIOCSV", []byte("\"1\";\"2\"\n"))}}
	share.failFirstN.Store(2)
	src := newTestSource(t, share)

	_, err := src.Rows(context.Background(), "2026-09", "Socios0.zip", func([]string) error { return nil })

	if err != nil {
		t.Fatalf("duas falhas seguidas deveriam ser absorvidas: %v", err)
	}
}

func TestRowsGivesUpAfterThreeFailures(t *testing.T) {
	share := &fakeShare{files: map[string][]byte{"2026-09/Socios0.zip": zipped(t, "S.SOCIOCSV", []byte("\"1\";\"2\"\n"))}}
	share.failFirstN.Store(100)
	src := newTestSource(t, share)

	_, err := src.Rows(context.Background(), "2026-09", "Socios0.zip", func([]string) error { return nil })

	if err == nil {
		t.Fatal("falha persistente deveria virar erro")
	}
}

func TestCodesReadsTheCodeTables(t *testing.T) {
	files := map[string][]byte{}
	for name, body := range map[string]string{
		"Cnaes.zip":         "\"4120400\";\"Construção de edifícios\"\n",
		"Municipios.zip":    "\"5869\";\"NOVA IGUACU\"\n",
		"Naturezas.zip":     "\"2062\";\"Sociedade Empresária Limitada\"\n",
		"Qualificacoes.zip": "\"49\";\"Sócio-Administrador\"\n",
		"Motivos.zip":       "\"00\";\"SEM MOTIVO\"\n",
	} {
		files["2026-09/"+name] = zipped(t, "X", latin1(body))
	}
	src := newTestSource(t, &fakeShare{files: files})

	codes, err := src.Codes(context.Background(), "2026-09")

	if err != nil {
		t.Fatal(err)
	}
	if codes.Activities["4120400"] != "Construção de edifícios" || codes.Cities["5869"] != "NOVA IGUACU" ||
		codes.Natures["2062"] != "Sociedade Empresária Limitada" || codes.Roles["49"] != "Sócio-Administrador" ||
		codes.Reasons["00"] != "SEM MOTIVO" {
		t.Errorf("códigos: %+v", codes)
	}
}
