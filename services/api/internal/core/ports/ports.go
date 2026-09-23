package ports

import (
	"context"
	"io"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type GazetteRepository interface {
	FindIDByChecksum(ctx context.Context, checksum string) (id string, found bool, err error)

	SaveWithActs(ctx context.Context, g *domain.Gazette, acts []domain.Act) error
	FindByID(ctx context.Context, id string) (domain.Gazette, error)
	ListByPeriod(ctx context.Context, from, to time.Time) ([]domain.Gazette, error)
	ReplaceActs(ctx context.Context, gazetteID, editionNumber string, acts []domain.Act) error
}

type ActRepository interface {
	Search(ctx context.Context, f domain.ActFilter) (hits []domain.ActHit, total int, err error)
	SearchInGazette(ctx context.Context, gazetteID, query string) ([]domain.ActHit, error)
	ListByGazette(ctx context.Context, gazetteID string) ([]domain.Act, error)

	ReportByEntity(ctx context.Context, kind domain.EntityKind, normalized string) (domain.CompanyReport, error)
	CountByMonth(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error)
	CountByOrgan(ctx context.Context) (map[string]int, error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *domain.Subscription) error
	FindByConfirmToken(ctx context.Context, token string) (domain.Subscription, error)
	FindByUnsubscribeToken(ctx context.Context, token string) (domain.Subscription, error)
	Update(ctx context.Context, s domain.Subscription) error
	ListActive(ctx context.Context) ([]domain.Subscription, error)
}

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

type ActParser interface {
	Parse(text string) []domain.Act

	EditionNumber(text string) string
}

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
