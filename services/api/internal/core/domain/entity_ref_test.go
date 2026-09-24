package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseEntityRef(t *testing.T) {
	cases := []struct {
		kind  EntityKind
		value string
		want  EntityRef
	}{
		{EntityCNPJ, "12.345.678/0001-90", EntityRef{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}},
		{EntityCNPJ, "12345678000190", EntityRef{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}},
		{EntityProcesso, "8.189/2025", EntityRef{EntityProcesso, "81892025", "8.189/2025"}},
		{EntityProcesso, "Processo nº 8.189/2025", EntityRef{EntityProcesso, "81892025", "8.189/2025"}},
		{EntityContrato, "012/2024", EntityRef{EntityContrato, "12/2024", "012/2024"}},
		{EntityContrato, "30-fms-2011", EntityRef{EntityContrato, "30/FMS/2011", "30/FMS/2011"}},
		{EntityProcesso, "8.189 / 2025", EntityRef{EntityProcesso, "81892025", "8.189/2025"}},
		{EntityContrato, "30 / FMS / 2011", EntityRef{EntityContrato, "30/FMS/2011", "30/FMS/2011"}},
	}
	for _, c := range cases {
		got, err := ParseEntityRef(c.kind, c.value)
		if err != nil || got != c.want {
			t.Errorf("ParseEntityRef(%s, %q) = %+v, %v; esperava %+v", c.kind, c.value, got, err, c.want)
		}
	}
}

func TestParseEntityRefRejectsInvalidInput(t *testing.T) {
	for _, c := range []struct {
		kind  EntityKind
		value string
	}{{EntityValor, "100"}, {"bobagem", "1/2024"}, {EntityCNPJ, "123"}, {EntityProcesso, "12"}, {EntityContrato, "abc"}} {
		if _, err := ParseEntityRef(c.kind, c.value); err == nil {
			t.Errorf("ParseEntityRef(%s, %q) deveria falhar", c.kind, c.value)
		}
	}
}

func TestParseEntityFilter(t *testing.T) {
	ref, err := ParseEntityFilter("contrato:012/2024")
	if err != nil || ref == nil || ref.Key != "12/2024" {
		t.Fatalf("veio %+v %v", ref, err)
	}
	if ref, err := ParseEntityFilter(""); ref != nil || err != nil {
		t.Errorf("filtro vazio deveria ser nil, veio %+v %v", ref, err)
	}
	for _, s := range []string{"cnpj", "cnpj:123", "valor:100"} {
		if _, err := ParseEntityFilter(s); !errors.Is(err, ErrInvalidFilter) {
			t.Errorf("%q deveria dar ErrInvalidFilter, veio %v", s, err)
		}
	}
}

func TestEntityRefDescription(t *testing.T) {
	for ref, want := range map[EntityRef]string{
		{EntityCNPJ, "12345678000190", "12.345.678/0001-90"}: "CNPJ 12.345.678/0001-90",
		{EntityProcesso, "81892025", "8.189/2025"}:           "processo 8.189/2025",
		{EntityContrato, "12/2024", "12/2024"}:               "contrato 12/2024",
	} {
		if got := ref.Description(); got != want {
			t.Errorf("veio %q, esperava %q", got, want)
		}
	}
}

func TestMentionSnippetCentersTheEvidence(t *testing.T) {
	body := strings.Repeat("antes ", 60) + "CNPJ: 12.345.678/0001-90." + strings.Repeat(" depois", 60)

	got := MentionSnippet(body, "12.345.678/0001-90")

	if !strings.Contains(got, "⟦12.345.678/0001-90⟧") || !strings.HasPrefix(got, "antes") {
		t.Fatalf("trecho inesperado: %q", got)
	}
	if n := len([]rune(got)) - 2; n > 280 {
		t.Errorf("trecho com %d caracteres, esperava até 280", n)
	}
}

func TestMentionSnippetCollapsesWhitespaceAndKeepsAccents(t *testing.T) {
	got := MentionSnippet("Contratação  da\nempresa, contrato nº\n55/2026.", "nº 55/2026")

	if got != "Contratação da empresa, contrato ⟦nº 55/2026⟧." {
		t.Errorf("veio %q", got)
	}
}

func TestMentionSnippetWithoutEvidenceIsTheStartOfTheBody(t *testing.T) {
	body := strings.Repeat("a", 400)

	if got := MentionSnippet(body, "xyz"); got != strings.Repeat("a", 280) {
		t.Errorf("veio %d caracteres", len(got))
	}
	if got := MentionSnippet("curto", ""); got != "curto" {
		t.Errorf("veio %q", got)
	}
}
