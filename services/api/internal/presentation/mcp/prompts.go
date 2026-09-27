package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

const promptRules = "Cite a edição, a data e a página de cada ato, com o link da fonte; a edição original é que vale. " +
	"Padrão é pista para conferir, não acusação: diga a regra e a ressalva de cada um. " +
	"Consulte fontes antes de afirmar que algo não foi publicado ou não foi pago. Não monte perfil de pessoa física."

func registerPrompts(srv *sdk.Server) {
	srv.AddPrompt(&sdk.Prompt{Name: "investigar_fornecedor", Title: "Investigar um fornecedor",
		Description: "Roteiro para levantar o que os Diários e as fontes cruzadas dizem de uma empresa pelo CNPJ.",
		Arguments:   []*sdk.PromptArgument{{Name: "cnpj", Description: "CNPJ da empresa, com ou sem pontuação", Required: true}}},
		supplierPrompt)
	srv.AddPrompt(&sdk.Prompt{Name: "seguir_contrato", Title: "Seguir um processo ou contrato",
		Description: "Roteiro para reconstituir a história de uma contratação: licitação ou dispensa, contrato, aditivos e pagamentos.",
		Arguments: []*sdk.PromptArgument{{Name: "numero", Description: "número do processo (06.10981/2025-7) ou do contrato (55/2026)", Required: true},
			{Name: "tipo", Description: "processo ou contrato; sem ele, o roteiro tenta as leituras que o número admite"}}},
		contractPrompt)
}

func supplierPrompt(_ context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
	cnpj, err := domain.ParseEntityInput(domain.EntityCNPJ, req.Params.Arguments["cnpj"])
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf("Investigue o fornecedor de CNPJ %s nos Diários Oficiais de São Gonçalo.\n"+
		"1. Chame entidade com tipo cnpj e numero %s: cadastro da Receita, sanções da CGU, pagamentos do TCE-RJ, empenhos do portal da Prefeitura (com o processo), contratos do PNCP, "+
		"obras paralisadas e emendas, além dos atos que citam o CNPJ.\n"+
		"2. Chame padroes com cnpj %s e explique cada achado com a regra e a ressalva.\n"+
		"3. Leia com ler_ato os atos mais importantes (primeiro contrato, aditivos, dispensas) e, se a empresa aparece só pelo nome, "+
		"use buscar_atos com nome.\n"+
		"4. Resuma: o que foi anunciado no Diário, o que foi pago, o que não bate e o que falta conferir.\n%s",
		domain.FormatCNPJ(cnpj), cnpj, cnpj, promptRules)
	return promptResult("Investigar o fornecedor "+domain.FormatCNPJ(cnpj), text), nil
}

func contractPrompt(_ context.Context, req *sdk.GetPromptRequest) (*sdk.GetPromptResult, error) {
	number := strings.TrimSpace(req.Params.Arguments["numero"])
	kinds, err := followedKinds(req.Params.Arguments["tipo"], number)
	if err != nil {
		return nil, err
	}
	calls := make([]string, len(kinds))
	for i, k := range kinds {
		calls[i] = fmt.Sprintf("tipo %s e numero %s", k, number)
	}
	which := calls[0]
	if len(calls) > 1 {
		which = "as duas leituras do número (" + strings.Join(calls, "; depois ") + ") e siga a que trouxer atos"
	}
	text := fmt.Sprintf("Siga o processo ou contrato %s nos Diários Oficiais de São Gonçalo.\n"+
		"1. Chame entidade com %s: os atos que o citam, por fase, e o que é citado junto (processos, contratos e CNPJs).\n"+
		"2. Monte a linha do tempo com ler_ato: abertura ou dispensa, homologação, extrato do contrato, aditivos e rescisão, com o valor de cada um.\n"+
		"3. Para cada CNPJ citado junto, chame entidade com tipo cnpj para ver o que foi pago; chame padroes com o mesmo tipo e número, e com cada CNPJ. "+
		"Para o número do processo, chame pagamentos e contratacoes com processo: os empenhos do portal da Prefeitura e a licitação e os contratos do mural.\n"+
		"4. Resuma quanto foi contratado, quanto cresceu com aditivos e quanto foi pago, e o que falta conferir. "+
		"Se a certeza da ligação for fraca, avise que o número pode juntar contratações diferentes.\n%s",
		number, which, promptRules)
	return promptResult("Seguir "+number, text), nil
}

func followedKinds(tipo, number string) ([]domain.EntityKind, error) {
	var candidates []domain.EntityKind
	switch strings.TrimSpace(tipo) {
	case string(domain.EntityProcesso):
		candidates = []domain.EntityKind{domain.EntityProcesso}
	case string(domain.EntityContrato):
		candidates = []domain.EntityKind{domain.EntityContrato}
	case "":
		candidates = []domain.EntityKind{domain.EntityProcesso, domain.EntityContrato}
	default:
		return nil, fmt.Errorf("%w: tipo deve ser processo ou contrato", domain.ErrInvalidInput)
	}
	var out []domain.EntityKind
	for _, k := range candidates {
		if _, err := domain.ParseEntityInput(k, number); err == nil {
			out = append(out, k)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: número de processo ou contrato %q", domain.ErrInvalidInput, number)
	}
	return out, nil
}

func promptResult(description, text string) *sdk.GetPromptResult {
	return &sdk.GetPromptResult{Description: description,
		Messages: []*sdk.PromptMessage{{Role: "user", Content: &sdk.TextContent{Text: text}}}}
}
