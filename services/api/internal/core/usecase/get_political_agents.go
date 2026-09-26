package usecase

import (
	"context"
	"fmt"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/ports"
)

type GetPoliticalAgents struct{ reader ports.PoliticalAgentReader }

func NewGetPoliticalAgents(r ports.PoliticalAgentReader) *GetPoliticalAgents {
	return &GetPoliticalAgents{reader: r}
}

func (uc *GetPoliticalAgents) Execute(ctx context.Context, role, query string) (domain.PoliticalAgentsReport, error) {
	if role != "" && !domain.IsAgentRole(role) {
		return domain.PoliticalAgentsReport{}, fmt.Errorf("%w: papel %q", domain.ErrInvalidInput, role)
	}
	pay, err := uc.reader.AgentPay(ctx)
	if err != nil {
		return domain.PoliticalAgentsReport{}, err
	}
	councillors, err := uc.reader.Councillors(ctx)
	if err != nil {
		return domain.PoliticalAgentsReport{}, err
	}
	report := domain.PoliticalAgentsReport{Agents: []domain.PoliticalAgent{}, Norms: domain.SubsidyNorms, Coverage: domain.PayCoverageOf(pay)}
	for _, a := range domain.BuildPoliticalAgents(pay, councillors) {
		if (role == "" || a.Role == role) && a.Matches(query) {
			report.Agents = append(report.Agents, a)
		}
	}
	return report, nil
}
