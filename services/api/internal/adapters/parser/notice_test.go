package parser

import (
	"strings"
	"testing"
)

const civilDefenseNotices = `SEMTRAN
EXTRATO DE RATIFICAÇÃO DE DISPENSA DE LICITAÇÃO.
Processo Administrativo n.º 18.635/2024.
VALOR TOTAL: R$ 57.999,60 (cinquenta e sete mil novecentos e
noventa e nove reais e sessenta centavos)
São Gonçalo, 23 de setembro de 2024.
FABIO RICARDO FONTES LEMOS
Secretário Municipal de Transportes
A COORDENADORIA MUNICIPAL DE DEFESA CIVIL DE SÃO
GONÇALO, de acordo com sua competência legal, vem pelo presente NOTIFICAR da
interdição o proprietário do imóvel localizado na TRAVESSA JOSÉ CARVALHO, 96.
São Gonçalo, 03 de julho de 2024.
FELIPE NASCIMENTO DE ASSUMPÇÃO
Subsecretário Municipal de Defesa Civil
A COORDENADORIA MUNICIPAL DE DEFESA CIVIL DE SÃO GONÇALO, de acordo com sua
competência legal, vem pelo presente NOTIFICAR da interdição o proprietário do
imóvel localizado na RUA MANOEL COSTA, 20.
São Gonçalo, 03 de julho de 2024.
FELIPE NASCIMENTO DE ASSUMPÇÃO`

func TestCivilDefenseNoticeEndsThePrecedingAct(t *testing.T) {
	acts := New().Parse(civilDefenseNotices)

	if len(acts) != 3 {
		t.Fatalf("esperava a dispensa e duas notificações, veio %d: %+v", len(acts), acts)
	}
	if strings.Contains(acts[0].Body, "DEFESA CIVIL") || !strings.HasSuffix(acts[0].Body, "Secretário Municipal de Transportes") {
		t.Errorf("a dispensa engoliu a notificação: %q", acts[0].Body)
	}
	for _, notice := range acts[1:] {
		if notice.Title != "NOTIFICAÇÃO DA DEFESA CIVIL" || notice.Organ != "COMDEC" {
			t.Errorf("notificação com título ou órgão errado: %+v", notice)
		}
	}
	if !strings.Contains(acts[1].Body, "TRAVESSA JOSÉ CARVALHO") || !strings.Contains(acts[2].Body, "RUA MANOEL COSTA") {
		t.Errorf("cada notificação deveria ter o seu imóvel: %q | %q", acts[1].Body, acts[2].Body)
	}
}

func TestCamaraParserLeavesCivilDefenseNoticesWithoutOrgan(t *testing.T) {
	acts := ForSource("diario_camara").Parse(civilDefenseNotices)

	if len(acts) != 3 || acts[1].Organ != "" {
		t.Fatalf("a Câmara não detecta órgão: %+v", acts)
	}
}

func TestActAfterCivilDefenseNoticesDoesNotInheritComdec(t *testing.T) {
	text := `SEMTRAN
EXTRATO DO CONTRATO Nº 5/2025
Objeto: sinalização viária.
COMDEC
A COORDENADORIA MUNICIPAL DE DEFESA CIVIL DE SÃO GONÇALO, vem pelo presente
NOTIFICAR da interdição o proprietário do imóvel na RUA MENTOR COUTO, 533.
SG-PREVI
PORTARIA - SEI Nº. 88/SG-PREVI/PRES/2025
A PRESIDENTE DO INSTITUTO DE PREVIDÊNCIA DO MUNICÍPIO DE SÃO GONÇALO resolve.`

	acts := New().Parse(text)

	last := acts[len(acts)-1]
	if !strings.HasPrefix(last.Title, "PORTARIA") || last.Organ == civilDefenseOrgan {
		t.Fatalf("a portaria do PREVI não é da Defesa Civil: %+v", last)
	}
}
