package pmsgmural

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTrustedRootsIncludeISRGRootYR(t *testing.T) {
	roots, err := trustedRoots()

	if err != nil || roots == nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(isrgRootYR), "BEGIN CERTIFICATE") {
		t.Error("raiz embutida não é PEM")
	}
}

func TestListFetchesThePageWithTheProjectAgent(t *testing.T) {
	var path, agent string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, agent = r.URL.Path, r.UserAgent()
		_, _ = w.Write([]byte("<table></table>"))
	}))
	defer srv.Close()
	s, err := New(srv.URL+"/", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	s.client = srv.Client()
	s.pause = 0

	body, err := s.List(context.Background(), "contratos")

	if err != nil || string(body) != "<table></table>" || path != "/contratos.php" || agent != userAgent {
		t.Errorf("veio %q %v %s %q", body, err, path, agent)
	}
}
