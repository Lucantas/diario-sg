package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

func TestCollectionAlertsDescribeTheFailure(t *testing.T) {
	at := time.Date(2026, 9, 23, 21, 37, 0, 0, time.UTC)
	cs := []domain.Coverage{
		{Source: domain.SourceDiarioCamara, LastRun: domain.FetchRun{ID: "a", FinishedAt: at, Error: "listar edições: context canceled"}},
		{Source: domain.SourceDiarioPrefeitura, LastRun: domain.FetchRun{ID: "b", FinishedAt: at, Failed: 2, Error: "timeout"}},
		{Source: domain.SourceDiarioPrefeitura, LastRun: domain.FetchRun{ID: "c", FinishedAt: at}},
	}

	alerts := collectionAlerts(cs)

	if len(alerts) != 2 {
		t.Fatalf("esperava dois alertas: %v", alerts)
	}
	if strings.Contains(alerts[0], "0 edição") || !strings.Contains(alerts[0], "falhou: listar edições: context canceled") {
		t.Errorf("falha sem edição contada não fala em 0 edições: %q", alerts[0])
	}
	if !strings.Contains(alerts[1], "falhou em 2 edições: timeout") {
		t.Errorf("falha em edições diz quantas: %q", alerts[1])
	}
}
