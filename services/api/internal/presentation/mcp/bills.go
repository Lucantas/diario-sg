package mcp

import (
	"context"
	"fmt"
	"net/url"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
)

const (
	maxBillsPerCall     = 20
	defaultBillsPerCall = 10
)

const billsDescription = "Proposições da Câmara Municipal de São Gonçalo (projetos de lei, de resolução, de emenda à Lei Orgânica, mensagens do " +
	"Executivo, emendas, indicações, moções), lidas da área pública do sistema de processo legislativo da Câmara (SICAM), com a tramitação. " +
	"Com processo (como 5564/2025) traz a proposição com a tramitação inteira e os pareceres das comissões; com tipo e numero do documento " +
	"(projeto de lei, 270/2019) traz a proposição daquele documento. Sem eles, lista com filtros: texto (ementa e autor), autor, tipo " +
	"(padrão: só as que podem virar norma; todos inclui indicações e moções), fase, situacao, parado_ha_dias (mínimo de dias sem " +
	"movimentação), tema (meio_ambiente) e de/ate (apresentação); paginada, até 20. por_fase conta as proposições dos outros filtros em " +
	"cada fase. A fase é deduzida da tramitação por regra (virou_lei, vetado, rejeitado, retirado, arquivado, enviado_ao_executivo, aprovado, " +
	"em_votacao, em_comissao, apresentado): confira na tramitação. Um projeto que virou lei também é arquivado no fim do mandato. leis traz " +
	"a lei que resultou do projeto: certeza exata vem do selo da página do SICAM, forte vem do autor da norma no SIAPEGOV, que cita o projeto. " +
	"O voto de cada vereador não está na base; o placar aparece no texto da tramitação. Cite o link da página do processo."

type billsInput struct {
	Process     string `json:"processo,omitempty" jsonschema:"número/ano do processo no SICAM, como 5564/2025"`
	Kind        string `json:"tipo,omitempty" jsonschema:"tipo do documento, como projeto de lei, mensagem, indicação legislativa; todos para qualquer tipo"`
	Number      string `json:"numero,omitempty" jsonschema:"número/ano do documento, como 270/2019 (use com tipo)"`
	Text        string `json:"texto,omitempty" jsonschema:"palavras da ementa ou do autor; aceita \"frase exata\", OR e -excluir"`
	Author      string `json:"autor,omitempty" jsonschema:"parte do nome do autor, como aparece no SICAM"`
	Phase       string `json:"fase,omitempty" jsonschema:"virou_lei, vetado, rejeitado, retirado, arquivado, enviado_ao_executivo, aprovado, em_votacao, em_comissao ou apresentado"`
	Status      string `json:"situacao,omitempty" jsonschema:"situação como o SICAM escreve: Ativo ou Arquivado"`
	MinIdleDays int    `json:"parado_ha_dias,omitempty" jsonschema:"só proposições sem movimentação há pelo menos estes dias; ordena das mais paradas para as menos"`
	Theme       string `json:"tema,omitempty" jsonschema:"meio_ambiente"`
	From        string `json:"de,omitempty" jsonschema:"apresentada a partir de, AAAA-MM-DD"`
	To          string `json:"ate,omitempty" jsonschema:"apresentada até, AAAA-MM-DD"`
	Offset      int    `json:"deslocamento,omitempty" jsonschema:"quantas pular, para paginar"`
	Limit       int    `json:"limite,omitempty" jsonschema:"por página, de 1 a 20 (padrão 10)"`
}

type billLawOut struct {
	Tipo        string `json:"tipo"`
	Numero      string `json:"numero"`
	Ementa      string `json:"ementa,omitempty"`
	Link        string `json:"link,omitempty"`
	Certeza     string `json:"certeza"`
	BuscaDiario string `json:"busca_diario"`
	BuscaNoSite string `json:"busca_no_site"`
	Promulgacao string `json:"promulgacao,omitempty"`
}

type billOut struct {
	Processo            string       `json:"processo"`
	Documento           string       `json:"documento"`
	Tipo                string       `json:"tipo"`
	Ementa              string       `json:"ementa"`
	Autores             string       `json:"autores"`
	Apresentacao        string       `json:"apresentacao,omitempty"`
	Situacao            string       `json:"situacao"`
	Fase                string       `json:"fase"`
	DiasSemMovimentacao int          `json:"dias_sem_movimentacao"`
	OrgaoAtual          string       `json:"orgao_atual"`
	UltimaMovimentacao  string       `json:"ultima_movimentacao"`
	Link                string       `json:"link"`
	LidoEm              string       `json:"lido_em"`
	Leis                []billLawOut `json:"leis,omitempty"`
}

type billEventOut struct {
	Quando string `json:"quando"`
	Rotulo string `json:"rotulo"`
	Texto  string `json:"texto"`
	Setor  string `json:"setor,omitempty"`
}

type billOpinionOut struct {
	Resultado string `json:"resultado"`
	Data      string `json:"data,omitempty"`
	Comissao  string `json:"comissao"`
	Relator   string `json:"relator"`
}

type billsOutput struct {
	Total       int              `json:"total"`
	PorFase     map[string]int   `json:"por_fase,omitempty"`
	RegraTema   string           `json:"regra_tema,omitempty"`
	Proposicoes []billOut        `json:"proposicoes"`
	Tramitacao  []billEventOut   `json:"tramitacao,omitempty"`
	Pareceres   []billOpinionOut `json:"pareceres,omitempty"`
}

func (s *server) bills(ctx context.Context, _ *sdk.CallToolRequest, in billsInput) (*sdk.CallToolResult, billsOutput, error) {
	switch {
	case in.Process != "":
		one, err := s.findBills.One(ctx, in.Process)
		if err != nil {
			return nil, billsOutput{}, err
		}
		out := billsOutput{Total: 1, Proposicoes: []billOut{s.billOut(one)}}
		for _, e := range one.Bill.Events {
			out.Tramitacao = append(out.Tramitacao, billEventOut{Quando: domain.InSaoPaulo(e.At).Format("2006-01-02 15:04"), Rotulo: e.Label, Texto: e.Text, Setor: e.Sector})
		}
		for _, o := range one.Bill.Opinions {
			op := billOpinionOut{Resultado: o.Result, Comissao: o.Committee, Relator: o.Rapporteur}
			if o.On != nil {
				op.Data = o.On.Format(time.DateOnly)
			}
			out.Pareceres = append(out.Pareceres, op)
		}
		return nil, out, nil
	case in.Number != "":
		found, err := s.findBills.ByDoc(ctx, in.Kind, in.Number)
		if err != nil {
			return nil, billsOutput{}, err
		}
		out := billsOutput{Total: len(found), Proposicoes: []billOut{}}
		for _, b := range found {
			out.Proposicoes = append(out.Proposicoes, s.billOut(b))
		}
		return nil, out, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultBillsPerCall
	}
	if limit > maxBillsPerCall {
		return nil, billsOutput{}, fmt.Errorf("%w: limite de 1 a %d", domain.ErrInvalidInput, maxBillsPerCall)
	}
	page, err := s.findBills.List(ctx, usecase.BillQuery{Text: in.Text, Author: in.Author, Kind: in.Kind, Status: in.Status, Phase: in.Phase,
		Theme: in.Theme, From: in.From, To: in.To, MinIdleDays: in.MinIdleDays, Limit: limit, Offset: in.Offset})
	if err != nil {
		return nil, billsOutput{}, err
	}
	out := billsOutput{Total: page.Total, PorFase: map[string]int{}, RegraTema: page.ThemeRule, Proposicoes: []billOut{}}
	for p, n := range page.ByPhase {
		out.PorFase[string(p)] = n
	}
	for _, b := range page.Items {
		out.Proposicoes = append(out.Proposicoes, s.billOut(b))
	}
	return nil, out, nil
}

func (s *server) billOut(sum domain.BillSummary) billOut {
	b := sum.Bill
	out := billOut{Processo: b.Key.String(), Documento: b.DocLabel, Tipo: b.Kind, Ementa: b.Summary, Autores: b.Authors, Situacao: b.Status,
		Fase: string(sum.Phase), DiasSemMovimentacao: sum.DaysIdle, OrgaoAtual: b.CurrentBody, UltimaMovimentacao: b.LastMovement,
		Link: b.URL, LidoEm: b.FetchedAt.Format(time.DateOnly)}
	if b.PresentedOn != nil {
		out.Apresentacao = b.PresentedOn.Format(time.DateOnly)
	}
	for _, l := range sum.Laws {
		search := domain.NormDiarioSearch(l.Norm)
		law := billLawOut{Tipo: string(l.Norm.Kind), Numero: l.Norm.Label(), Ementa: l.Norm.Summary, Link: l.URL, Certeza: string(l.Certainty),
			BuscaDiario: search, BuscaNoSite: s.webURL + "/?" + url.Values{"q": {search}}.Encode()}
		if l.Norm.PromulgatedOn != nil {
			law.Promulgacao = l.Norm.PromulgatedOn.Format(time.DateOnly)
		}
		out.Leis = append(out.Leis, law)
	}
	return out
}
