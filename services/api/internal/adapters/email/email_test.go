package email

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type recSender struct{ sent []Message }

func (r *recSender) Send(_ context.Context, m Message) error {
	r.sent = append(r.sent, m)
	return nil
}

func TestSendMatchesNamesTheSource(t *testing.T) {
	sender := &recSender{}
	n := NewNotifier(sender, "https://site")
	sub := domain.Subscription{Email: "a@b.c", Query: "cota parlamentar", UnsubscribeToken: "t"}
	hits := []domain.ActHit{{Act: domain.Act{Title: "TERMO DE APROVAÇÃO"}, Snippet: "⟦cota⟧"}}
	day := time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)

	for _, g := range []domain.Gazette{
		{EditionNumber: "138", PublishedAt: day, Source: domain.SourceDiarioCamara},
		{EditionNumber: "1771", PublishedAt: day},
	} {
		if err := n.SendMatches(context.Background(), sub, g, hits); err != nil {
			t.Fatal(err)
		}
	}

	camara, prefeitura := sender.sent[0], sender.sent[1]
	if camara.Subject != "“cota parlamentar” no Diário da Câmara de 03/11" ||
		!strings.Contains(camara.HTML, "Diário Oficial Eletrônico da Câmara Municipal de São Gonçalo") {
		t.Errorf("alerta da Câmara: %q\n%s", camara.Subject, camara.HTML)
	}
	if prefeitura.Subject != "“cota parlamentar” no Diário da Prefeitura de 03/11" ||
		!strings.Contains(prefeitura.HTML, "Diário Oficial do Município de São Gonçalo") {
		t.Errorf("alerta da Prefeitura: %q\n%s", prefeitura.Subject, prefeitura.HTML)
	}
}

func TestEntityAlertEmails(t *testing.T) {
	sender := &recSender{}
	n := NewNotifier(sender, "https://site")
	sub := domain.Subscription{Email: "a@b.c", ConfirmToken: "c", UnsubscribeToken: "u",
		Entity: &domain.EntityRef{Kind: domain.EntityProcesso, Key: "81892025", Label: "8.189/2025"}}
	hits := []domain.ActHit{{Act: domain.Act{Title: "EXTRATO DO CONTRATO", Organ: "SEMED"}, Snippet: "processo ⟦8.189/2025⟧"}}
	g := domain.Gazette{EditionNumber: "1771", PublishedAt: time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC)}

	if err := n.SendConfirmation(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if err := n.SendMatches(context.Background(), sub, g, hits); err != nil {
		t.Fatal(err)
	}

	confirm, matches := sender.sent[0], sender.sent[1]
	if !strings.Contains(confirm.HTML, "<strong>processo 8.189/2025</strong>") {
		t.Errorf("confirmação: %s", confirm.HTML)
	}
	if matches.Subject != "Processo 8.189/2025 no Diário da Prefeitura de 03/11" ||
		!strings.Contains(matches.HTML, "para <strong>processo 8.189/2025</strong>") ||
		!strings.Contains(matches.HTML, "EXTRATO DO CONTRATO</strong> (SEMED)") {
		t.Errorf("alerta: %q\n%s", matches.Subject, matches.HTML)
	}
}
