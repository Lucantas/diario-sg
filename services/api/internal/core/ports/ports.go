// Package ports define as interfaces que o core exige do mundo externo.
// Adapters (Postgres, GCS, Pub/Sub, e-mail...) implementam estas interfaces.
package ports

import (
	"context"
	"io"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type GazetteRepository interface {
	// FindIDByChecksum permite idempotência: a mesma edição não é indexada 2x.
	FindIDByChecksum(ctx context.Context, checksum string) (id string, found bool, err error)
	// SaveWithActs grava a edição e seus atos numa transação e preenche g.ID.
	SaveWithActs(ctx context.Context, g *domain.Gazette, acts []domain.Act) error
	FindByID(ctx context.Context, id string) (domain.Gazette, error)
}

type ActRepository interface {
	Search(ctx context.Context, f domain.ActFilter) (hits []domain.ActHit, total int, err error)
	SearchInGazette(ctx context.Context, gazetteID, query string) ([]domain.ActHit, error)
	ListByGazette(ctx context.Context, gazetteID string) ([]domain.Act, error)
	// ReportByEntity lista os atos em que uma entidade normalizada aparece,
	// do mais recente ao mais antigo, com soma dos valores e contagem por tipo.
	ReportByEntity(ctx context.Context, kind domain.EntityKind, normalized string) (domain.CompanyReport, error)
	CountByMonth(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *domain.Subscription) error
	FindByConfirmToken(ctx context.Context, token string) (domain.Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (domain.Subscription, error)
	Update(ctx context.Context, s domain.Subscription) error
	ListActive(ctx context.Context) ([]domain.Subscription, error)
}

// NotificationLog evita alertas duplicados quando mensagens são reentregues.
type NotificationLog interface {
	WasSent(ctx context.Context, subscriptionID, gazetteID string) (bool, error)
	MarkSent(ctx context.Context, subscriptionID, gazetteID string) error
}

type FileStorage interface {
	Get(ctx context.Context, path string) (io.ReadCloser, error)
}

type TextExtractor interface {
	Extract(ctx context.Context, r io.Reader) (string, error)
}

// ActParser separa o texto de uma edição em atos. Hoje é por regex; no
// futuro pode ser um modelo de ML/LLM sem mudar o caso de uso.
type ActParser interface {
	Parse(text string) []domain.Act
	// EditionNumber lê o número da edição impresso no texto ("" se não achar).
	// O site da prefeitura não informa o número; só o PDF traz.
	EditionNumber(text string) string
}

// EntityExtractor encontra campos (CNPJ, valores, contratos, processos) no
// corpo de um ato. Separado do ActParser porque segmentar e extrair evoluem
// em ritmos diferentes: na fase 2 este adapter pode virar um modelo sem
// tocar na segmentação.
type EntityExtractor interface {
	Extract(body string) []domain.Entity
}

type EventPublisher interface {
	GazetteIndexed(ctx context.Context, gazetteID string, actsCount int) error
}

type Notifier interface {
	SendConfirmation(ctx context.Context, s domain.Subscription) error
	SendMatches(ctx context.Context, s domain.Subscription, g domain.Gazette, hits []domain.ActHit) error
}
