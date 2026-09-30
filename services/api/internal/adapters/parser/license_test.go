package parser

import (
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestClassifyEnvironmentalLicenses(t *testing.T) {
	cases := []struct {
		name, title, body string
		want              domain.ActType
	}{
		{"concessão de licença prévia", "CONCESSÃO DE LICENÇA",
			"CONCESSÃO DE LICENÇA\nMICAL INVEST E PARTICIPAÇÕES LTDA torna público que recebeu da Secretaria Municipal de Meio Ambiente e Transportes – SEMMATRAN, a LICENÇA MUNICIPAL PRÉVIA (LMP) nº 008/2026",
			domain.ActLicencaAmbiental},
		{"cassação de licença de operação", "CASSAÇÃO DE LICENÇA DE OPERAÇÃO",
			"CASSAÇÃO DE LICENÇA DE OPERAÇÃO\nA Secretaria Municipal de Meio Ambiente torna pública a cassação", domain.ActLicencaAmbiental},
		{"licença de servidor", "CONCESSÃO DE LICENÇA",
			"CONCESSÃO DE LICENÇA\nConcede licença para tratar de interesses particulares ao servidor FULANO, matrícula 123.", domain.ActOutro},
		{"título com o nome da empresa", "SOBERANO INDÚSTRIA E COMÉRCIO DE ALIMENTOS LTDA",
			"SOBERANO INDÚSTRIA E COMÉRCIO DE ALIMENTOS LTDA\nCNPJ: 00.000.000/0001-00\ntorna público que recebeu da Secretaria Municipal de Meio Ambiente a Licença Municipal de Operação",
			domain.ActLicencaAmbiental},
		{"edital que cita licença continua edital", "EDITAL DE CONVOCAÇÃO",
			"EDITAL DE CONVOCAÇÃO\nA empresa torna público que recebeu a licença ambiental", domain.ActEdital},
	}
	for _, c := range cases {
		if got := classify(c.title, c.body); got != c.want {
			t.Errorf("%s: esperava %s, veio %s", c.name, c.want, got)
		}
	}
}
