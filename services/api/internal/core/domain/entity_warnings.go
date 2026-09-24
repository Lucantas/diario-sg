package domain

import "fmt"

func EntityWarnings(r EntityReport) []string {
	var out []string
	switch {
	case r.Kind != EntityCNPJ && r.Sources > 1:
		out = append(out, "Este número aparece no Diário da Prefeitura e no da Câmara, que numeram processos e contratos cada um à sua maneira: "+
			"podem ser registros diferentes. Use diario para ver só um dos dois e confira o campo diario de cada ato.")
	case r.Certainty == CertaintyWeak:
		out = append(out, "Número de contrato sem a sigla do órgão: estes atos podem ser de contratos diferentes, de órgãos diferentes, com o mesmo número e ano. Confira o órgão em cada ato.")
	}
	if n := namedOrgans(r.Organs); r.Kind != EntityCNPJ && n > 1 && r.Certainty != CertaintyWeak {
		out = append(out, fmt.Sprintf("Este número aparece em %d órgãos. Podem ser processos diferentes com o mesmo número, "+
			"ou uma ata de registro de preços usada por vários órgãos. Confira o órgão em cada ato.", n))
	}
	return out
}

func namedOrgans(os []OrganCount) int {
	n := 0
	for _, o := range os {
		if o.Organ != "" {
			n++
		}
	}
	return n
}
