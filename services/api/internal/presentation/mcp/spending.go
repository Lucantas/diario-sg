package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const paymentsDescription = "Empenhos do portal da transparência da Prefeitura de São Gonçalo (2017 em diante, todas as entidades: Prefeitura, fundos, " +
	"fundações, SG-PREVI e Câmara), filtrados por cnpj do credor, entidade (parte do nome, sem acento: saude, educacao, camara), processo " +
	"(número/ano), texto (no objeto e no nome do credor) e anos (de, ate); os filtros se combinam e é preciso ao menos um. " +
	"Traz os totais (empenhos, empenhado, liquidado e pago), por_ano, por_entidade, os 10 maiores credores pelo pago e 20 empenhos, " +
	"os mais recentes primeiro (use pular para os seguintes). Com só os anos, por_ano traz pago_rreo_centavos, o pago do ano declarado " +
	"pelo Município ao Tesouro (RREO do SICONFI), como total de controle; a diferença vem de retenções e do que o portal não detalha. " +
	"O portal diz o processo de cada empenho, mas não o contrato; use contratacoes com o processo para achar a licitação e o contrato."

const contractingDescription = "Licitações, dispensas e contratos do mural da Prefeitura de São Gonçalo e contratos do PNCP, " +
	"filtrados por cnpj (as licitações e os contratos com o mesmo processo dos empenhos da empresa no portal, mais os contratos dela no PNCP), " +
	"processo (número/ano), texto (no objeto e, nos contratos, no fornecedor), orgao (a sigla no número do edital: PMSG, FMS, FMAS, FAESG…) " +
	"e anos (de, ate: da abertura nas licitações, do processo nos contratos); os filtros se combinam e é preciso ao menos um. " +
	"Traz até 20 licitações e 20 contratos (pular para os seguintes), com o total de cada e a soma dos valores dos contratos. " +
	"O mural não traz CNPJ e o número do processo não tem o órgão: a ligação por cnpj é pelo processo, confira fornecedor e objeto. " +
	"Os extratos publicados no Diário ficam em buscar_atos (tipo contrato) e entidade (processo ou contrato)."

type paymentsInput struct {
	CNPJ     string `json:"cnpj,omitempty" jsonschema:"CNPJ do credor"`
	Entidade string `json:"entidade,omitempty" jsonschema:"parte do nome da entidade que empenhou, como saude ou camara"`
	Processo string `json:"processo,omitempty" jsonschema:"número/ano do processo, como 17943/2024"`
	Texto    string `json:"texto,omitempty" jsonschema:"palavras do objeto ou do nome do credor"`
	De       int    `json:"de,omitempty" jsonschema:"primeiro ano"`
	Ate      int    `json:"ate,omitempty" jsonschema:"último ano"`
	Pular    int    `json:"pular,omitempty" jsonschema:"empenhos a pular na lista (de 20 em 20)"`
}

type spendingTotalsDTO struct {
	Empenhos          int   `json:"empenhos"`
	EmpenhadoCentavos int64 `json:"empenhado_centavos"`
	LiquidadoCentavos int64 `json:"liquidado_centavos"`
	PagoCentavos      int64 `json:"pago_centavos"`
}

type spendingYearDTO struct {
	Ano int `json:"ano"`
	spendingTotalsDTO
	PagoRREOCentavos *int64 `json:"pago_rreo_centavos,omitempty"`
}

type spendingGroupDTO struct {
	Nome string `json:"nome"`
	CNPJ string `json:"cnpj,omitempty"`
	spendingTotalsDTO
}

type paymentCommitmentDTO struct {
	Credor string `json:"credor"`
	CNPJ   string `json:"cnpj"`
	municipalCommitmentDTO
}

type paymentsOutput struct {
	Fonte       string                 `json:"fonte"`
	Totais      spendingTotalsDTO      `json:"totais"`
	PorAno      []spendingYearDTO      `json:"por_ano"`
	PorEntidade []spendingGroupDTO     `json:"por_entidade"`
	PorCredor   []spendingGroupDTO     `json:"maiores_credores"`
	Empenhos    []paymentCommitmentDTO `json:"empenhos"`
}

const paymentsSource = "Portal da transparência da Prefeitura de São Gonçalo (sistema.pmsg.rj.gov.br/portal-transparencia)"

func (s *server) payments(ctx context.Context, _ *sdk.CallToolRequest, in paymentsInput) (*sdk.CallToolResult, paymentsOutput, error) {
	q, err := domain.NewPaymentQuery(in.CNPJ, in.Entidade, in.Processo, in.Texto, in.De, in.Ate, in.Pular)
	if err != nil {
		return nil, paymentsOutput{}, err
	}
	r, err := s.queryPayments.Execute(ctx, q)
	if err != nil {
		return nil, paymentsOutput{}, err
	}
	out := paymentsOutput{Fonte: paymentsSource, Totais: totalsOf(r.Totals), PorAno: []spendingYearDTO{}, PorEntidade: []spendingGroupDTO{},
		PorCredor: []spendingGroupDTO{}, Empenhos: []paymentCommitmentDTO{}}
	for _, y := range r.ByYear {
		out.PorAno = append(out.PorAno, spendingYearDTO{Ano: y.Year, spendingTotalsDTO: totalsOf(y.SpendingTotals), PagoRREOCentavos: y.ControlPaidCents})
	}
	for _, g := range r.ByEntity {
		out.PorEntidade = append(out.PorEntidade, spendingGroupDTO{Nome: g.Name, spendingTotalsDTO: totalsOf(g.SpendingTotals)})
	}
	for _, g := range r.BySupplier {
		out.PorCredor = append(out.PorCredor, spendingGroupDTO{Nome: g.Name, CNPJ: domain.FormatCNPJ(g.Key), spendingTotalsDTO: totalsOf(g.SpendingTotals)})
	}
	for _, c := range r.Commitments {
		out.Empenhos = append(out.Empenhos, paymentCommitmentDTO{Credor: c.Name, CNPJ: domain.FormatCNPJ(c.CNPJ), municipalCommitmentDTO: commitmentOf(c)})
	}
	return nil, out, nil
}

func totalsOf(t domain.SpendingTotals) spendingTotalsDTO {
	return spendingTotalsDTO{Empenhos: t.Commitments, EmpenhadoCentavos: t.CommittedCents, LiquidadoCentavos: t.LiquidatedCents, PagoCentavos: t.PaidCents}
}

type contractingInput struct {
	CNPJ     string `json:"cnpj,omitempty" jsonschema:"CNPJ do fornecedor"`
	Processo string `json:"processo,omitempty" jsonschema:"número/ano do processo, como 17943/2024"`
	Texto    string `json:"texto,omitempty" jsonschema:"palavras do objeto ou do fornecedor"`
	Orgao    string `json:"orgao,omitempty" jsonschema:"sigla no número do edital, como PMSG, FMS ou FMAS"`
	De       int    `json:"de,omitempty" jsonschema:"primeiro ano"`
	Ate      int    `json:"ate,omitempty" jsonschema:"último ano"`
	Pular    int    `json:"pular,omitempty" jsonschema:"linhas a pular em cada lista (de 20 em 20)"`
}

type contractingOutput struct {
	Aviso                string            `json:"aviso"`
	LicitacoesTotal      int               `json:"licitacoes_total"`
	Licitacoes           []muralRowDTO     `json:"licitacoes"`
	ContratosTotal       int               `json:"contratos_total"`
	ContratosValorSomado int64             `json:"contratos_valor_somado_centavos"`
	Contratos            []muralRowDTO     `json:"contratos"`
	ContratosPNCP        []pncpContractDTO `json:"contratos_pncp,omitempty"`
}

const contractingCaveat = "Mural da Prefeitura (licitações, dispensas e contratos) e, com cnpj, o PNCP. Contratos do mural com valor 0 são " +
	"linhas sem valor na fonte (aditivos, apostilamentos); abra o documento."

func (s *server) contracting(ctx context.Context, _ *sdk.CallToolRequest, in contractingInput) (*sdk.CallToolResult, contractingOutput, error) {
	q, err := domain.NewProcurementQuery(in.CNPJ, in.Processo, in.Texto, in.Orgao, in.De, in.Ate, in.Pular)
	if err != nil {
		return nil, contractingOutput{}, err
	}
	r, err := s.queryProcurements.Execute(ctx, q)
	if err != nil {
		return nil, contractingOutput{}, err
	}
	out := contractingOutput{Aviso: contractingCaveat, LicitacoesTotal: r.ProcurementsTotal, Licitacoes: []muralRowDTO{},
		ContratosTotal: r.ContractsTotal, ContratosValorSomado: r.ContractValueCents, Contratos: []muralRowDTO{}}
	for _, p := range r.Procurements {
		out.Licitacoes = append(out.Licitacoes, muralTenderOf(p))
	}
	for _, c := range r.Contracts {
		out.Contratos = append(out.Contratos, muralContractOf(c))
	}
	if len(r.PNCP) > 0 {
		out.ContratosPNCP = pncpContractsOf(r.PNCP)
	}
	return nil, out, nil
}
