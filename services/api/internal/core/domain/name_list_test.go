package domain

import (
	"strings"
	"testing"
)

func TestNameLinesCountsLinesThatAreOnlyACapitalizedName(t *testing.T) {
	body := "PORTARIA Nº 12/2025\nNomeia os aprovados:\nMARIA DA SILVA\nJOSÉ DE SOUZA SANTOS\nAna Paula\n" +
		"LEANDRO MACHADO MACEDO, matrícula 123\nSECRETARIA MUNICIPAL DE EDUCAÇÃO\nX\n"

	if got := NameLines(body); got != 3 {
		t.Fatalf("esperava 3 linhas de nome, veio %d", got)
	}
}

func TestLongNameListsCrossTheThreshold(t *testing.T) {
	list := strings.Repeat("FULANO DE TAL\n", NameListMinLines)

	if NameLines(list) < NameListMinLines {
		t.Fatalf("uma lista de %d nomes deveria atingir o limite", NameListMinLines)
	}
}

func TestNameFilterSearchesThePhraseAndSkipsLists(t *testing.T) {
	f := ActFilter{Query: "nomeação", Name: `  "Leandro   Machado" `}
	if err := f.Normalize(); err != nil {
		t.Fatal(err)
	}

	if got := f.TextQuery(); got != "nomeação" {
		t.Errorf("com consulta, o nome não entra nela: %q", got)
	}
	if got := f.NamePhrase(); got != `"Leandro Machado"` {
		t.Errorf("NamePhrase = %q", got)
	}
	onlyName := ActFilter{Name: "Leandro Machado"}
	if got := onlyName.TextQuery(); got != `"Leandro Machado"` {
		t.Errorf("sem consulta, o nome vira a frase buscada: %q", got)
	}
	if !f.ExcludesNameLists() {
		t.Error("busca por nome deixa de fora as listas por padrão")
	}
	f.IncludeLists = true
	if f.ExcludesNameLists() {
		t.Error("incluir listas desliga a exclusão")
	}
	if (ActFilter{Name: ` "" `}).ExcludesNameLists() {
		t.Error("nome só com aspas e espaços não é busca por nome")
	}
	plain := ActFilter{Query: "merenda"}
	if plain.TextQuery() != "merenda" || plain.ExcludesNameLists() {
		t.Errorf("sem nome a busca não muda: %+v", plain)
	}
}
