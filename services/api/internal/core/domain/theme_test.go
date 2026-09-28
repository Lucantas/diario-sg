package domain

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestEnvironmentThemeMatchesRealLawSummaries(t *testing.T) {
	theme, err := ParseTheme("meio_ambiente")
	if err != nil {
		t.Fatal(err)
	}

	in := []string{
		"DISPÕE SOBRE AS SANÇÕES ADMINISTRATIVAS DECORRENTES DE INFRAÇÕES AMBIENTAIS, ESTABELECE OS PROCEDIMENTOS",
		"DISPÕE SOBRE A SUPRESSÃO, TRANSPLANTIO, PODA E MANEJO DE ARBORIZAÇÃO NO MUNICÍPIO DE SÃO GONÇALO",
		"DISPÕE NO ÂMBITO DO MUNICÍPIO DE SÃO GONÇALO, SOBRE A APLICAÇÃO DE MULTAS E OUTRAS SANÇÕES PARA O DESCARTE IRREGULAR DE RESÍDUOS SÓLIDOS",
		"INSTITUI O CÓDIGO MUNICIPAL DE PROTEÇÃO AOS ANIMAIS, NO ÂMBITO DO MUNICÍPIO DE SÃO GONÇALO",
		"DISPÕE SOBRE A PROIBIÇÃO DE COMERCIALIZAÇÃO, INSTALAÇÃO E USO DE EQUIPAMENTOS PARA MOTOCICLETAS QUE PRODUZAM RUÍDOS ACIMA DO LIMITE",
		"DISPÕE SOBRE A REGULAMENTAÇÃO DA UTILIZAÇÃO DE CANUDOS DE PLÁSTICO NO MUNICÍPIO DE SÃO GONÇALO.",
		"ALTERA A LEI Nº 032/2002 QUE INSTITUIU PROCEDIMENTOS DE COBRANÇA PARA LICENCIAMENTO DE ATIVIDADES POLUIDORAS - LAP",
		"AUTORIZA A PREFEITURA MUNICIPAL A INSTITUIR O IPTU VERDE.",
		"INSTITUI A POLÍTICA MUNICIPAL DE DRENAGEM E MANEJO DE ÁGUAS PLUVIAIS URBANAS",
	}
	out := []string{
		"DISPÕE SOBRE PARÂMETROS DE ATUAÇÃO PREVENTIVA NO COMBATE AOS ENTORPECENTES NO AMBIENTE ESCOLAR",
		"INSTITUI O COMBATE AO ASSÉDIO MORAL E A QUALQUER FORMA DE PERSEGUIÇÃO NO AMBIENTE LABORATIVO",
		"CRIA O SERVIÇO DE INSPEÇÃO MUNICIPAL DE PRODUTOS DE ORIGEM ANIMAL DE SÃO GONÇALO",
		"ABRE CRÉDITO SUPLEMENTAR E ALTERA O ORÇAMENTO E O QUADRO DE DETALHAMENTO DAS DESPESAS DA SECRETARIA MUNICIPAL DE CONSERVAÇÃO.",
		"DISPÕE SOBRE A DESTINÇÃO PRIORITÁRIA DOS PRIMEIROS ANDARES EM EDIFÍCIOS RESIDÊNCIAIS CONSTRUÍDOS POR PROGRAMAS HABITACIONAIS",
		"DISPÕE SOBRE A CONCESSÃO DE 01 (UM) DIA DE LICENÇA POR ANO, AOS SERVIDORES PARA EXAME GINECOLÓGICO",
		"INSTITIU A CAMPANHA DE PROMOÇÃO PARA A PREVENÇÃO AOS ACIDENTES DE TRABALHO, DENOMINADA SEMANA VERDE",
		"DISPÕE SOBRE A IMPLANTAÇÃO DO CANAL DE DENÚNCIAS OU SUSPEITAS DE MAUS TRATOS CONTRA IDOSOS",
		"DISPÕE SOBRE NORMAS ESPECÍFICAS PARA A INSTALAÇÃO DE INFRAESTRUTURA DE SUPORTE PARA EQUIPAMENTOS DE TELECOMUNICAÇÕES, LICENCIAMENTO",
		"CONCEDE REMISSÃO DE DÉBITOS TRIBUTÁRIOS RELATIVOS AO IPTU E À TAXA DE COLETA DE LIXO",
		"DISPÕE SOBRE AÇÕES PÚBLICAS DE SAÚDE, VISANDO A PREVENÇÃO DA HEPATITE \"A\" PARA QUEM TRABALHA NA COLETA DE LIXO.",
	}
	for _, s := range in {
		if !theme.MatchesText(s) {
			t.Errorf("deveria entrar: %s", s)
		}
	}
	for _, s := range out {
		if theme.MatchesText(s) {
			t.Errorf("não deveria entrar: %s", s)
		}
	}
}

func TestEnvironmentThemeCommitteeAndOrgans(t *testing.T) {
	theme, _ := ParseTheme("meio_ambiente")

	if !theme.MatchesCommittee("COMISSÃO DE DEFESA DO MEIO AMBIENTE") || theme.MatchesCommittee("COMISSÃO DE JUSTIÇA E REDAÇÃO") {
		t.Error("comissão")
	}
	if !contains(theme.Organs, "SEMMA") || !contains(theme.Organs, "COMMADS") || contains(theme.Organs, "SEMMATRAN") || theme.PartialOrgan != "SEMMATRAN" {
		t.Errorf("órgãos: %v %q", theme.Organs, theme.PartialOrgan)
	}
	if !theme.PartialOrganExcluded("RECURSOS AO CORIM – 1ª Instância - JULGAMENTO DE MULTA") ||
		!theme.PartialOrganExcluded("DISPÕE SOBRE A INTERDIÇÃO DE VIAS PARA O FLUXO DE VEÍCULOS") ||
		!theme.PartialOrganExcluded("DESPACHO DO PRESIDENTE RECURSOS A JARI – I- Sessão de 01/07/2026. JULGAMENTO DE MULTA") ||
		!theme.PartialOrganExcluded("PORTARIA - SEI Nº 117/SG-PREVI/PRES/DPV/GBP/SCB/2025 A PRESIDENTE DO INSTITUTO DE PREVIDÊNCIA") ||
		theme.PartialOrganExcluded("POSTO ABREU DOIS LTDA torna público que recebeu da Secretaria Municipal de Meio Ambiente e Transportes – SEMMATRAN, a LICENÇA MUNICIPAL DE RECUPERAÇÃO E OPERAÇÃO") {
		t.Error("exclusões da SEMMATRAN")
	}
	if theme.Rule == "" || !strings.Contains(theme.Rule, "SEMMATRAN") {
		t.Errorf("regra: %q", theme.Rule)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestThemeRegexesWorkInPostgresSyntax(t *testing.T) {
	theme, _ := ParseTheme("meio_ambiente")

	for _, re := range []string{theme.TermsSQLRegex(), theme.ExcludeSQLRegex(), theme.PartialOrganExcludeSQLRegex(), theme.PartialOrganNameSQLRegex()} {
		if strings.Contains(re, `\b`) || !strings.Contains(re, `\y`) && re == theme.TermsSQLRegex() {
			t.Errorf("regex do Postgres deveria usar \\y: %s", re)
		}
		if _, err := regexp.Compile(strings.ReplaceAll(re, `\y`, `\b`)); err != nil {
			t.Errorf("regex inválida: %v", err)
		}
	}
}

func TestParseTheme(t *testing.T) {
	if th, err := ParseTheme(""); err != nil || th.Slug != "" {
		t.Errorf("vazio: %v %v", th, err)
	}
	if th, err := ParseTheme(" Meio_Ambiente "); err != nil || th.Slug != "meio_ambiente" {
		t.Errorf("maiúsculas: %v %v", th, err)
	}
	if _, err := ParseTheme("saude"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("tema desconhecido: %v", err)
	}
}
