package domain

import "strings"

type Organ struct {
	Acronym string
	Name    string
}

type OrganCount struct {
	Organ
	Acts int
}

func IsKnownOrgan(acronym string) bool {
	_, ok := organNames[acronym]
	return ok
}

func OrganName(acronym string) string { return organNames[acronym] }

func NormalizeOrgan(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return "", true
	}
	if !IsKnownOrgan(s) {
		return "", false
	}
	return PrincipalOrgan(s), true
}

var organNames = map[string]string{
	"CAISAN":       "Câmara Intersecretarias de Segurança Alimentar e Nutricional",
	"CEC":          "Comissão Especial de Contratação",
	"CGGMSG":       "Corregedoria-Geral da Guarda Municipal de São Gonçalo",
	"CGMSG":        "Corregedoria-Geral da Guarda Municipal de São Gonçalo",
	"CMAS":         "Conselho Municipal de Assistência Social de São Gonçalo",
	"CMC":          "Conselho Municipal de Cultura de São Gonçalo",
	"CMDCA":        "Conselho Municipal dos Direitos da Criança e do Adolescente",
	"CMDDM":        "Conselho Municipal de Defesa dos Direitos das Mulheres de São Gonçalo",
	"CMDDMSG":      "Conselho Municipal de Defesa dos Direitos da Mulher de São Gonçalo",
	"CMDM":         "Conselho Municipal dos Direitos da Mulher",
	"CME":          "Conselho Municipal de Educação",
	"CMEL":         "Conselho Municipal de Esporte e Lazer de São Gonçalo",
	"CMPA":         "Conselho Municipal de Proteção Animal",
	"CMS":          "Conselho Municipal de Saúde",
	"CODEPE":       "Conselho Municipal dos Direitos da Pessoa com Deficiência de São Gonçalo",
	"COMAD":        "Conselho Municipal Antidrogas de São Gonçalo",
	"COMDAR":       "Conselho Municipal de Desenvolvimento Agropecuário de São Gonçalo",
	"COMDEC":       "Coordenadoria Municipal de Defesa Civil de São Gonçalo",
	"COMDEPISG":    "Conselho Municipal de Defesa dos Direitos da Pessoa Idosa de São Gonçalo",
	"COMDESG":      "Conselho Municipal de Desenvolvimento Econômico de São Gonçalo",
	"COMENQ":       "Comissão de Enquadramento",
	"COMINSG":      "Conselho Municipal de Defesa dos Direitos do Negro e Promoção da Igualdade Racial e Étnica em São Gonçalo",
	"COMIRSG":      "Conselho Municipal de Defesa dos Direitos do Negro e Promoção da Igualdade Racial e Étnica em São Gonçalo",
	"COMMADS":      "Conselho Municipal de Meio Ambiente e Desenvolvimento Sustentável",
	"COMPAD":       "Conselho Municipal de Políticas Públicas sobre Álcool e Drogas",
	"COMSEA":       "Conselho Municipal de Segurança Alimentar e Nutricional de São Gonçalo",
	"COMSEP":       "Conselho Municipal de Segurança Pública",
	"CONCIDADES":   "Conselho Municipal da Cidade de São Gonçalo",
	"CONDEC":       "Coordenadoria Municipal de Defesa Civil de São Gonçalo",
	"CONGES":       "Conselho Gestor da Fundação Municipal de Assistência à Saúde dos Servidores de São Gonçalo",
	"CONSAD":       "Conselho de Administração do Instituto de Previdência do Município de São Gonçalo",
	"COPEDE":       "Conselho Municipal dos Direitos da Pessoa com Deficiência de São Gonçalo",
	"CORIM":        "Comissão de Recursos de Infrações Municipais",
	"CPAC":         "Comissão Permanente de Acúmulos de Cargos",
	"CPADAG":       "Comissão Permanente de Avaliação de Documentos do Arquivo Geral",
	"CPAPPF":       "Comissão Permanente de Análise para Progressão e Promoção Funcional",
	"CPIAD":        "Comissão Permanente de Inquérito Administrativo Disciplinar",
	"CPPF":         "Comissão Permanente de Promoção Funcional da Administração Direta",
	"CPPFGMF":      "Comissão Permanente de Promoção Funcional dos Servidores Públicos da Guarda Municipal e Assistência à Saúde dos Servidores do Município de São Gonçalo",
	"CPPFS":        "Comissão Permanente de Promoção Funcional dos Servidores Públicos da Saúde do Município de São Gonçalo",
	"EDURSAN":      "Empresa Municipal de Desenvolvimento Urbano e Saneamento Ambiental",
	"FAELSG":       "Fundação de Artes, Esporte e Lazer de São Gonçalo",
	"FAESG":        "Fundação de Artes, Esporte e Lazer de São Gonçalo",
	"FAESLG":       "Fundação de Artes, Esporte e Lazer de São Gonçalo",
	"FASG":         "Fundação de Artes de São Gonçalo",
	"FIASG":        "Fundação Municipal de Apoio à Educação e Assistência à Infância e à Adolescência de São Gonçalo",
	"FMAS":         "Fundo Municipal de Assistência Social",
	"FMS":          "Fundação Municipal de Saúde",
	"FMSSG":        "Fundação Municipal de Saúde de São Gonçalo",
	"FUMIA":        "Fundo Municipal para a Infância e Adolescência",
	"FUMPARJ":      "Fundação Municipal de Parques e Jardins",
	"FUNASG":       "Fundação Municipal de Assistência à Saúde dos Servidores de São Gonçalo",
	"FUNCULTURA":   "Fundo Municipal de Cultura",
	"FUNDESG":      "Fundo de Gestão, Desenvolvimento e Modernização da Procuradoria Geral do Município de São Gonçalo",
	"FUNPARJ":      "Fundação Municipal de Parques e Jardins",
	"GM":           "Guarda Municipal de São Gonçalo",
	"GMSG":         "Guarda Municipal de São Gonçalo",
	"IPASG":        "Instituto de Previdência dos Servidores Municipais de São Gonçalo",
	"IPIIBA":       "",
	"PGE":          "Procuradoria Geral do Município",
	"PGM":          "Procuradoria Geral do Município",
	"PROMEA":       "Programa Municipal de Educação Ambiental de São Gonçalo",
	"SAMSADC":      "Secretaria Municipal de Saúde e Defesa Civil",
	"SECGOV":       "Secretaria Municipal de Governo",
	"SECHABITA":    "Secretaria Municipal de Habitação",
	"SEMA":         "Secretaria Municipal de Meio Ambiente",
	"SEMAD":        "Secretaria Municipal de Administração",
	"SEMAIMPD":     "",
	"SEMANT":       "Secretaria Municipal Antidrogas",
	"SEMAP":        "Secretaria Municipal de Agricultura e Pesca",
	"SEMAPPAP":     "Secretaria Municipal de Agricultura e Pesca",
	"SEMAS":        "Secretaria Municipal de Assistência Social",
	"SEMCI":        "Secretaria Municipal de Controle Interno",
	"SEMCOM":       "Secretaria Municipal de Comunicação Social",
	"SEMCOMP":      "Secretaria Municipal de Compras e Suprimentos",
	"SEMCON":       "Secretaria Municipal de Conservação",
	"SEMCS":        "Secretaria Municipal de Comunicação",
	"SEMDE":        "Secretaria Municipal de Desenvolvimento Econômico",
	"SEMDUR":       "Secretaria Municipal de Desenvolvimento Urbano",
	"SEMED":        "Secretaria Municipal de Educação",
	"SEMEL":        "Secretaria Municipal de Esporte e Lazer",
	"SEMFA":        "Secretaria Municipal de Fazenda",
	"SEMGIPE":      "Secretaria Municipal de Gestão Integrada e Projetos Especiais",
	"SEMGOV":       "Secretaria Municipal de Governo",
	"SEMGOVCOM":    "Secretaria Municipal de Governo, Comunicação Social e Posturas",
	"SEMGOVCOMS":   "Secretaria Municipal de Governo, Comunicação Social e Posturas",
	"SEMGOVCON":    "Secretaria Municipal de Governo e Comunicação Social",
	"SEMHAB":       "Secretaria Municipal de Habitação",
	"SEMHABITA":    "Secretaria Municipal de Habitação",
	"SEMIDC":       "Secretaria Municipal de Integração, Defesa do Consumidor e Políticas para as Mulheres",
	"SEMIMD":       "Secretaria Municipal de Políticas Públicas para o Idoso, Mulher e Pessoa com Deficiência",
	"SEMIND":       "Secretaria Municipal de Políticas Públicas para o Idoso, Mulher e Pessoa com Deficiência",
	"SEMINDECON":   "Secretaria Municipal de Integração, Defesa do Consumidor e Políticas para as Mulheres",
	"SEMIURB":      "Secretaria Municipal de Infraestrutura e Urbanismo",
	"SEMIURBCPARJ": "Secretaria Municipal de Infraestrutura, Urbanismo e Conservação de Parques e Jardins",
	"SEMIURME":     "",
	"SEMMA":        "Secretaria Municipal de Meio Ambiente",
	"SEMMADU":      "Secretaria Municipal de Meio Ambiente e Desenvolvimento Urbano",
	"SEMMATRAN":    "Secretaria Municipal de Meio Ambiente e Transportes",
	"SEMOP":        "Secretaria Municipal de Ordem Pública",
	"SEMPAD":       "Secretaria Municipal de Políticas sobre Álcool e Drogas",
	"SEMPESCA":     "Secretaria Municipal de Pesca",
	"SEMPLAN":      "Secretaria Municipal de Planejamento",
	"SEMPPE":       "Secretaria Municipal de Planejamento e Projetos Especiais",
	"SEMSA":        "Secretaria Municipal de Saúde e Assistência",
	"SEMSAD":       "Secretaria Municipal de Saúde e Defesa Civil",
	"SEMSADC":      "Secretaria Municipal de Saúde e Defesa Civil",
	"SEMSEP":       "Secretaria Municipal de Segurança Pública",
	"SEMTCUL":      "Secretaria Municipal de Turismo e Cultura",
	"SEMTRAB":      "Secretaria Municipal do Trabalho",
	"SEMTRAN":      "Secretaria Municipal de Transportes",
	"SEOP":         "Secretaria Municipal de Ordem Pública",
	"SETURCUL":     "Secretaria Municipal de Turismo e Cultura",
	"SMAP":         "",
	"SMAS":         "Secretaria Municipal de Assistência Social",
	"SMC":          "",
	"SMDS":         "Secretaria Municipal de Desenvolvimento Social",
	"SMDSHABIA":    "Secretaria Municipal de Desenvolvimento Social, Habitação, Infância e Adolescência",
	"SMDSIA":       "Secretaria Municipal de Desenvolvimento Social, Infância e Adolescência",
	"SMPPE":        "Secretaria Municipal de Planejamento e Projetos Especiais",
	"SMSP":         "Secretaria Municipal de Segurança Pública",
	"SMSS":         "Secretaria Municipal de Seguridade Social",
	"SMTC":         "Secretaria Municipal de Turismo e Cultura",
	"SSM":          "",
	"SUBCOMP":      "Subsecretaria de Compras e Suprimentos",
}
