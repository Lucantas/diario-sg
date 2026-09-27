package siapegov

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormsPostsTheWholeCategoryAndDecodesLatin1(t *testing.T) {
	var form, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		form, path = string(b), r.URL.Path
		_, _ = w.Write([]byte{'S', 0xe3, 'o'})
	}))
	defer srv.Close()
	s := New(srv.URL+"/leis/", srv.Client())
	s.pause = 0

	body, err := s.Norms(context.Background(), "05")

	if err != nil || string(body) != "São" {
		t.Fatalf("veio %q %v", body, err)
	}
	if path != "/leis/resulta_leis.php" || !strings.Contains(form, "Vcategoria=05") || !strings.Contains(form, "paginacao=0") {
		t.Errorf("pedido: %s %s", path, form)
	}
}
