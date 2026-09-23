package config

import "testing"

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("GCP_PROJECT_ID", "p")
	t.Setenv("GAZETTE_BUCKET", "b")
	t.Setenv("TOPIC_GAZETTE_FETCHED", "t")
	t.Setenv("TOPIC_FETCH_COMPLETED", "r")
}

func TestLoadDefaultsToThePrefeitura(t *testing.T) {
	setRequired(t)
	t.Setenv("SOURCE", "")
	t.Setenv("SOURCE_URL", "")

	c, err := Load()

	if err != nil || c.Source != "diario_prefeitura" || c.SourceURL != "https://do.pmsg.rj.gov.br/" {
		t.Fatalf("padrão: %+v %v", c, err)
	}
}

func TestLoadCamaraUsesItsOwnURL(t *testing.T) {
	setRequired(t)
	t.Setenv("SOURCE", "diario_camara")
	t.Setenv("SOURCE_URL", "")

	c, err := Load()

	if err != nil || c.SourceURL != "https://www.cmsg.rj.gov.br/diariooficialeletronico/" {
		t.Fatalf("Câmara: %+v %v", c, err)
	}
}

func TestLoadRejectsUnknownSource(t *testing.T) {
	setRequired(t)
	t.Setenv("SOURCE", "tce")

	if _, err := Load(); err == nil {
		t.Fatal("fonte desconhecida deveria dar erro")
	}
}

func TestLoadRejectsTheURLOfTheOtherSource(t *testing.T) {
	setRequired(t)
	t.Setenv("SOURCE", "diario_camara")
	t.Setenv("SOURCE_URL", "https://do.pmsg.rj.gov.br/")

	if _, err := Load(); err == nil {
		t.Fatal("a Câmara com a URL da Prefeitura deveria dar erro")
	}
}
