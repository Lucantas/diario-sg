package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type want struct {
	title string
	typ   domain.ActType
}

// Trechos reais do Diário (nomes de pessoas físicas trocados por fictícios),
// no formato que o pdftotext produz em modo de leitura.
func TestParseRealFixtures(t *testing.T) {
	cases := map[string][]want{
		"atos_prefeito.txt": {
			{"DECRETO Nº 441/2026", domain.ActDecreto},
			{"DECRETO Nº 444/2026", domain.ActDecreto},
			{"PORTARIA - SEI N.º 1004/SEMAD/SUBRH/CIF/2026", domain.ActPortaria},
			{"DESPACHO DO SECRETÁRIO", domain.ActDespacho},
			{"DESPACHO DO SECRETÁRIO", domain.ActDespacho},
			{"TERMO DE APROVAÇÃO DE PRESTAÇÃO DE CONTAS COM RESSALVA", domain.ActOutro},
		},
		"pessoal.txt": {
			{"PORTARIA Nº 1490/2026", domain.ActExoneracao},
			{"PORTARIA Nº 1492/2026", domain.ActPortaria},
			{"CORRIGENDA DA PORTARIA Nº 1409/2026", domain.ActOutro},
			{"Continuação do D.O.E. em 18/09/2026", domain.ActOutro},
			{"Port. nº 1497/2026", domain.ActPortaria},
			{"Port. nº 1498/2026", domain.ActNomeacao},
			{"Port. nº 1499/2026", domain.ActExoneracao},
			{"Port. nº 1500/2026", domain.ActExoneracao},
			{"Port. nº 1502/2026", domain.ActPortaria},
		},
		"licitacoes.txt": {
			{"DECRETO Nº 446/2026", domain.ActDecreto},
			{"EXTRATO DO CONTRATO DE COMODATO DE IMÓVEL", domain.ActContrato},
			{"EXTRATO DE INEXIGIBILIDADE DE LICITAÇÃO", domain.ActDispensa},
			{"EXTRATO DO QUINTO TERMO ADITIVO DE PRORROGAÇÃO AO CONTRATO DE LOCAÇÃO 006/2020.", domain.ActAditivo},
			{"AVISO DE DISPENSA ELETRÔNICA N° 19/2026", domain.ActDispensa},
			{"AVISO DE LICITAÇÃO SRP", domain.ActLicitacao},
			{"TERMO DE APREENSÃO ADMINISTRATIVA Nº 202/SEMMATRAN/MA/GERENCIARAGP/2026", domain.ActOutro},
			{"NOTIFICAÇÃO Nº 191/SEMMATRAN/MA/GAB/2026", domain.ActOutro},
			{"CONCESSÃO DE LICENÇA", domain.ActOutro},
		},
	}
	for file, wants := range cases {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", file))
			if err != nil {
				t.Fatal(err)
			}
			acts := New().Parse(string(raw))
			if len(acts) != len(wants) {
				t.Fatalf("esperava %d atos, veio %d:\n%s", len(wants), len(acts), titles(acts))
			}
			for i, w := range wants {
				if acts[i].Title != w.title || acts[i].Type != w.typ || acts[i].Position != i {
					t.Errorf("ato %d: esperava %q/%s, veio %q/%s (pos %d)", i, w.title, w.typ, acts[i].Title, acts[i].Type, acts[i].Position)
				}
			}
		})
	}
}

func TestParseDropsCoverRosterAndPageFurniture(t *testing.T) {
	raw, _ := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", "atos_prefeito.txt"))
	acts := New().Parse(string(raw))
	all := titles(acts)
	for _, a := range acts {
		all += "\n" + a.Body
	}
	for _, forbidden := range []string{"CAPITÃO FULANO EXEMPLO", "https://do.pmsg.rj.gov.br/", "DIÁRIO OFICIAL ELETRÔNICO", "ATOS DO PREFEITO", "\nSEMAD\n", "\nSEMED\n"} {
		if strings.Contains(all, forbidden) {
			t.Errorf("%q não deveria aparecer em nenhum ato", forbidden)
		}
	}
	if !strings.Contains(acts[0].Body, "ANEXO DECRETO Nº 441/2026") {
		t.Error("o anexo do decreto deve ficar no corpo do próprio decreto")
	}
	if !strings.Contains(acts[1].Body, "Pedro Exemplo Moreira") {
		t.Error("o corpo do decreto 444 deve conter os nomes incluídos")
	}
}

func TestSplitWordHeaderIsJoinedButBodyWordsAreNot(t *testing.T) {
	raw, _ := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", "licitacoes.txt"))
	acts := New().Parse(string(raw))
	var inexig domain.Act
	for _, a := range acts {
		if strings.HasPrefix(a.Title, "EXTRATO DE INEXIGIBILIDADE") {
			inexig = a
		}
	}
	if !strings.Contains(inexig.Body, "ASSOCIAÇÃO\nDO\nFUNDO\nDE") {
		t.Errorf("palavras soltas do corpo não devem ser reunidas: %q", inexig.Body)
	}
}

func titles(acts []domain.Act) string {
	var b strings.Builder
	for _, a := range acts {
		b.WriteString("  " + string(a.Type) + "\t" + a.Title + "\n")
	}
	return b.String()
}
