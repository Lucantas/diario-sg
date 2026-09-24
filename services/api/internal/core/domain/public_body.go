package domain

const municipality = "Município de São Gonçalo"

var publicBodyRoots = map[string]string{
	"28636579": municipality,
}

var publicBodies = map[string]string{
	"28579636000100": municipality,
	"39260120000163": "Fundação Municipal de Saúde de São Gonçalo",
	"14472412000139": "FUNASG",
	"32538167000105": "SG-PREVI",
	"04541202000100": "Fundação de Artes, Esporte e Lazer de São Gonçalo",
	"11109114000190": "Fundo Municipal de Assistência Social",
	"29846003000122": "Câmara Municipal de São Gonçalo",
}

func PublicBody(cnpj string) (string, bool) {
	digits, ok := NormalizeCNPJ(cnpj)
	if !ok {
		return "", false
	}
	if name, ok := publicBodies[digits]; ok {
		return name, true
	}
	name, ok := publicBodyRoots[digits[:8]]
	return name, ok
}

func PublicBodyCNPJs() []string {
	out := make([]string, 0, len(publicBodies))
	for cnpj := range publicBodies {
		out = append(out, cnpj)
	}
	return out
}

func PublicBodyRoots() []string {
	out := make([]string, 0, len(publicBodyRoots))
	for root := range publicBodyRoots {
		out = append(out, root)
	}
	return out
}
