package domain

import (
	"testing"
	"time"
)

func TestStoragePathBySource(t *testing.T) {
	day := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)

	prefeitura := Edition{Number: "1771", PublishedAt: day, URL: "https://do.pmsg.rj.gov.br/diario/2025_11_03.pdf"}
	camara := Edition{Source: SourceDiarioCamara, PublishedAt: day, URL: "https://www.cmsg.rj.gov.br/diariooficialeletronico/PUBLICACOES/2025-11-03.pdf"}

	if got := prefeitura.StoragePath(); got != "gazettes/2025/11/03/edicao-1771.pdf" {
		t.Errorf("Prefeitura: %s", got)
	}
	if got := camara.StoragePath(); got != "raw/diario_camara/2025/11/03/2025-11-03.pdf" {
		t.Errorf("Câmara: %s", got)
	}
	if got := camara.MarkerPath(); got != "raw/diario_camara/2025/11/03/2025-11-03.pdf.published" {
		t.Errorf("marcador da Câmara: %s", got)
	}
}

func TestValidSource(t *testing.T) {
	for s, want := range map[string]bool{SourceDiarioPrefeitura: true, SourceDiarioCamara: true, "": false, "tce": false} {
		if ValidSource(s) != want {
			t.Errorf("ValidSource(%q) deveria ser %v", s, want)
		}
	}
}
