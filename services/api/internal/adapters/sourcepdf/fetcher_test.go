package sourcepdf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenReturnsPDFBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("sem User-Agent")
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.WriteString(w, "%PDF-1.4 conteúdo")
	}))
	defer srv.Close()

	body, err := New().Open(context.Background(), srv.URL)

	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	b, _ := io.ReadAll(body)
	if string(b) != "%PDF-1.4 conteúdo" {
		t.Errorf("%q", b)
	}
}

func TestOpenRejectsNonOKAndNonPDF(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"404":             func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"html com 200":    func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "<html>erro</html>") },
		"corpo vazio 200": func(w http.ResponseWriter, _ *http.Request) {},
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			if body, err := New().Open(context.Background(), srv.URL); err == nil {
				body.Close()
				t.Fatal("devia falhar")
			}
		})
	}
}
