package domain

type Collector struct {
	Source string
	Name   string
}

var Collectors = []Collector{
	{SourceBills, "Processo legislativo da Câmara (SICAM): proposições, tramitação e pareceres"},
	{SourceNorms, "Consulta de leis da Prefeitura (SIAPEGOV): leis e decretos"},
	{SourcePoliticalAgents, "Folhas da Prefeitura e da Câmara: agentes políticos"},
	{SourceMunicipalCommitments, "Portal da transparência da Prefeitura: empenhos"},
	{SourceMural, "Mural de licitações e contratos da Prefeitura"},
	{SourceTCE, "TCE-RJ: empenhos do município"},
	{SourceStaff, "TCE-RJ: pessoal"},
	{SourceOversight, "TCE-RJ: contas, débitos e obras paralisadas"},
	{SourceSiconfi, "SICONFI (Tesouro): RREO"},
	{SourcePNCP, "PNCP: contratos"},
	{SourceReceita, "Receita Federal: cadastro de CNPJ"},
	{SourceSanctions, "CGU: sanções (CEIS, CNEP, CEPIM)"},
	{SourceAmendments, "CGU: emendas parlamentares"},
	{SourceTransfers, "CGU: transferências da União"},
	{SourceSpecialTransfers, "Transferegov: transferências especiais"},
}

func CollectorSources() []string {
	out := make([]string, len(Collectors))
	for i, c := range Collectors {
		out[i] = c.Source
	}
	return out
}
