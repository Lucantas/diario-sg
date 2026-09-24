//go:build integration

package integration

import (
	"strings"
	"testing"
)

func nameSearchGazette() string {
	return "PORTARIA Nº 10/2026\nProcesso nº06.10981/2025-7. Nomeia LEANDRO MACHADO\nMACEDO para o cargo de assessor.\n" +
		"PORTARIA Nº 11/2026\nHomologa o resultado do concurso:\n" +
		strings.Repeat("FULANO DE TAL\n", 60) + "LEANDRO MACHADO MACEDO\n" +
		"PORTARIA Nº 12/2026\nExonera MACHADO, LEANDRO e MACEDO de suas funções."
}

type nameSearch struct {
	Total        int `json:"total"`
	OmittedLists int `json:"atos_em_listas_omitidos"`
	Acts         []struct {
		Title string `json:"titulo"`
	} `json:"atos"`
}

func TestMCPNameSearchMatchesTheAdjacentNameAndSkipsLongLists(t *testing.T) {
	srv, _ := newServerFor(t, nameSearchGazette())
	_, key := issueKey(t, srv.URL, "")
	session, err := connect(t, srv.URL, key)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	byName, _ := call[nameSearch](t, session, "buscar_atos", map[string]any{"nome": "Leandro Machado Macedo"})
	withLists, _ := call[nameSearch](t, session, "buscar_atos", map[string]any{"nome": "Leandro Machado Macedo", "incluir_listas": true})

	if byName.Total != 1 || byName.Acts[0].Title != "PORTARIA Nº 10/2026" {
		t.Fatalf("só a nomeação tem o nome com as palavras juntas fora de uma lista: %+v", byName)
	}
	if byName.OmittedLists != 1 {
		t.Errorf("a resposta diz quantas listas ficaram de fora: %+v", byName)
	}
	if withLists.Total != 2 || withLists.OmittedLists != 0 {
		t.Errorf("incluir_listas traz a homologação: %+v", withLists)
	}

	withNumber, _ := call[nameSearch](t, session, "buscar_atos", map[string]any{"consulta": "06.10981/2025-7", "nome": "Leandro Machado Macedo"})
	if withNumber.Total != 1 || withNumber.Acts[0].Title != "PORTARIA Nº 10/2026" {
		t.Errorf("consulta por número colado ao nº continua casando junto com nome: %+v", withNumber)
	}

	grouped, _ := call[groupsOut](t, session, "agrupar", map[string]any{"por": "tipo", "nome": "Leandro Machado Macedo"})
	if grouped.MatchedActs != 1 {
		t.Errorf("agrupar aplica o mesmo filtro de nome: %+v", grouped)
	}
}
