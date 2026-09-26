package agentes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const exportPage = `<form method="post"><input type="hidden" name="__VIEWSTATE" id="__VIEWSTATE" value="vs&amp;1" />
<input type="hidden" name="__VIEWSTATEGENERATOR" id="__VIEWSTATEGENERATOR" value="GEN" />
<input type="hidden" name="__EVENTVALIDATION" id="__EVENTVALIDATION" value="EV" /></form>`

func newTestSource(t *testing.T, h http.HandlerFunc) *Source {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	s := New(srv.URL+"/pref", srv.URL+"/camara", srv.URL+"/sicam/", srv.Client())
	s.pause = time.Millisecond
	return s
}

func TestPrefeituraPayAsksForTheMonth(t *testing.T) {
	var query string
	s := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	})

	body, err := s.PrefeituraPay(context.Background(), 2026, 8)

	if err != nil || string(body) != `{"success":true,"data":[]}` {
		t.Fatalf("resposta: %q %v", body, err)
	}
	if !strings.Contains(query, "flag=remuneracao") || !strings.Contains(query, "entidade=1") || !strings.Contains(query, "competencia=08%2F2026") {
		t.Errorf("consulta: %s", query)
	}
}

func TestCamaraPayPostsTheExportForm(t *testing.T) {
	s := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.SetCookie(w, &http.Cookie{Name: "ASP.NET_SessionId", Value: "abc"})
			_, _ = w.Write([]byte(exportPage))
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		cookie, _ := r.Cookie("ASP.NET_SessionId")
		if r.PostForm.Get("__VIEWSTATE") != "vs&1" || r.PostForm.Get("__EVENTVALIDATION") != "EV" || r.PostForm.Get("ctl00$containerCorpo$cbxAno") != "2025" ||
			r.PostForm.Get("ctl00$containerCorpo$cbxMes") != "00" || r.PostForm.Get("ctl00$containerCorpo$cbxFormato") != "JSON" || cookie == nil {
			http.Error(w, "formulário errado", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"ano":"2025"}]`))
	})

	body, err := s.CamaraPay(context.Background(), 2025)

	if err != nil || string(body) != `[{"ano":"2025"}]` {
		t.Fatalf("exportação: %q %v", body, err)
	}
}

func TestCamaraPayRejectsAnHTMLAnswer(t *testing.T) {
	s := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(exportPage))
	})

	if _, err := s.CamaraPay(context.Background(), 2025); err == nil {
		t.Error("página HTML aceita como exportação")
	}
}

func TestCouncillorsAsksForTheYear(t *testing.T) {
	var got string
	s := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = w.Write([]byte(`{"Parametros":[],"Total":0}`))
	})

	if _, err := s.Councillors(context.Background(), 2026); err != nil {
		t.Fatal(err)
	}
	if got != "/sicam/?Parlamentares/2026/json" {
		t.Errorf("consulta: %s", got)
	}
}

func TestErrorStatusFails(t *testing.T) {
	s := newTestSource(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fora", http.StatusForbidden)
	})

	if _, err := s.PrefeituraPay(context.Background(), 2026, 8); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("erro: %v", err)
	}
}
