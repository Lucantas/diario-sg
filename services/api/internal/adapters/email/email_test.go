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
