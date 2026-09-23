package domain

import "testing"

func TestSourceOrDefaultTreatsMissingAsPrefeitura(t *testing.T) {
	if got := SourceOrDefault(""); got != SourceDiarioPrefeitura {
		t.Errorf("fonte vazia virou %q", got)
	}
	if got := SourceOrDefault(SourceDiarioCamara); got != SourceDiarioCamara {
		t.Errorf("fonte da Câmara virou %q", got)
	}
}

func TestValidSource(t *testing.T) {
	for _, s := range Sources {
		if !ValidSource(s) {
			t.Errorf("%q deveria ser válida", s)
		}
	}
	for _, s := range []string{"", "tce", "DIARIO_CAMARA"} {
		if ValidSource(s) {
			t.Errorf("%q não deveria ser válida", s)
		}
	}
}

func TestSourceName(t *testing.T) {
	if got := SourceName(""); got != "Diário Oficial do Município de São Gonçalo" {
		t.Errorf("nome da Prefeitura: %q", got)
	}
	if got := SourceName(SourceDiarioCamara); got != "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo" {
		t.Errorf("nome da Câmara: %q", got)
	}
}
