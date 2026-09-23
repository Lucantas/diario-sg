package domain

const (
	SourceDiarioPrefeitura = "diario_prefeitura"
	SourceDiarioCamara     = "diario_camara"
)

var Sources = []string{SourceDiarioPrefeitura, SourceDiarioCamara}

var sourceNames = map[string]string{
	SourceDiarioPrefeitura: "Diário Oficial do Município de São Gonçalo",
	SourceDiarioCamara:     "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo",
}

func ValidSource(s string) bool {
	_, ok := sourceNames[s]
	return ok
}

func SourceOrDefault(s string) string {
	if s == "" {
		return SourceDiarioPrefeitura
	}
	return s
}

func SourceName(s string) string { return sourceNames[SourceOrDefault(s)] }
