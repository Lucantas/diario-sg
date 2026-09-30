//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/seu-usuario/diario-sg/services/api/internal/adapters/postgres"
)

const licenseToNewCompany = "CONCESSÃO DE LICENÇA\nA Secretaria de Meio Ambiente torna público que concedeu a Licença Municipal Prévia LMP nº 008/2026 " +
	"a F.P. VIEIRA ENGENHARIA LTDA, CNPJ 14.180.324/0001-63, para loteamento residencial no Engenho Pequeno.\n" +
	"CANCELAMENTO DE LICENÇA\nFica cancelada a Licença Municipal de Operação LMO nº 3/2020 de F.P. VIEIRA ENGENHARIA LTDA, CNPJ 14.180.324/0001-63, por pedido da empresa."

func TestNewCompanyLicenseComesFromTheLicenseActAndTheRegistry(t *testing.T) {
	srv, db := newServerFor(t, licenseToNewCompany)
	loadRegistry(t, postgres.NewRegistryRepo(db), postgres.NewFetchRunRepo(db),
		registryShare{month: "2026-09", name: "F.P. VIEIRA ENGENHARIA LTDA", opened: "20250601"})

	var res patternsResponse
	getJSON(t, srv.URL+"/v1/patterns", &res)

	var findings []patternFinding
	for _, item := range res.Items {
		if item.ID == "empresa_nova_licenciada" {
			findings = item.Findings
		}
	}
	if len(findings) != 1 || findings[0].Title != "F.P. VIEIRA ENGENHARIA LTDA (14.180.324/0001-63): licença ambiental 474 dias depois da abertura" ||
		len(findings[0].Acts) != 1 || !strings.Contains(findings[0].Acts[0].Title, "CONCESSÃO") {
		t.Fatalf("empresa nova licenciada: %+v", findings)
	}
}
