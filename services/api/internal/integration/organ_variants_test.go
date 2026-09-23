//go:build integration

package integration

import "testing"

const variantGazette = "ATOS DO PREFEITO\nDECRETO Nº 1/2026\nDispõe sobre o horário.\n" +
	"FMS\nEXTRATO DO CONTRATO Nº 3/2026\nObjeto: medicamentos.\n" +
	"FMSSG\nEXTRATO DO CONTRATO Nº 4/2026\nObjeto: seringas.\n" +
	"SEMAD\nPORTARIA Nº 10/2026\nNomeia servidor para a função."

func TestOrganVariantsCountAsThePrincipal(t *testing.T) {
	srv, _ := newServerFor(t, variantGazette)

	for _, organ := range []string{"FMS", "fmssg"} {
		var hits organHits
		getJSON(t, srv.URL+"/v1/acts?organ="+organ, &hits)
		if hits.Total != 2 {
			t.Fatalf("organ=%s deveria trazer a sigla principal e a variante: %+v", organ, hits)
		}
	}

	var organs struct {
		Items []struct {
			Acronym string `json:"acronym"`
			Acts    int    `json:"acts"`
		} `json:"items"`
	}
	getJSON(t, srv.URL+"/v1/organs", &organs)
	if len(organs.Items) != 2 || organs.Items[0].Acronym != "FMS" || organs.Items[0].Acts != 2 {
		t.Fatalf("a variante deveria somar na sigla principal: %+v", organs)
	}
}
