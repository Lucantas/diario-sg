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
	EntityHitsInGazette(ctx context.Context, gazetteID string, ref domain.EntityRef) ([]domain.ActHit, error)
	ListByGazette(ctx context.Context, gazetteID string) ([]domain.Act, error)

	ReportByEntity(ctx context.Context, kind domain.EntityKind, normalized string) (domain.CompanyReport, error)
	CountByMonth(ctx context.Context, f domain.ActFilter) ([]domain.MonthCount, error)
	CountByOrgan(ctx context.Context) (map[string]int, error)
	Export(ctx context.Context, f domain.ActFilter, yield func(h domain.ActHit, total int) error) error
}

type PatternSource interface {
	DispensaActs(ctx context.Context) ([]domain.DispensaAct, error)
	AddendumActs(ctx context.Context) ([]domain.AddendumAct, error)
	EmergencyActs(ctx context.Context) ([]domain.EmergencyAct, error)
	MonthlyActCounts(ctx context.Context, types []domain.ActType, source string) ([]domain.MonthlyActCount, error)
	HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error)
}

type SupplierPatternSource interface {
	PanelActs(ctx context.Context, source string) ([]domain.PanelAct, error)
	SupplierProfiles(ctx context.Context) (map[string]domain.SupplierProfile, error)
	AllSanctions(ctx context.Context) ([]domain.Sanction, error)
	PaidCreditors(ctx context.Context) ([]domain.CreditorPaid, error)
	CitedInDiario(ctx context.Context) (map[string]bool, error)
	PaymentsCoverage(ctx context.Context) (*domain.PaymentCoverage, error)
	PanelActsWithoutCNPJ(ctx context.Context, source string) ([]domain.PanelAct, error)
	AllPNCPContracts(ctx context.Context) ([]domain.PNCPContract, error)
	CitedProcesses(ctx context.Context) (map[string]bool, error)
	LatestGazetteDay(ctx context.Context, source string) (time.Time, error)
}

type PanelSource interface {
	PanelActs(ctx context.Context, source string) ([]domain.PanelAct, error)
	HitsByIDs(ctx context.Context, ids []string) ([]domain.ActHit, error)
}

type ActGrouper interface {
	Group(ctx context.Context, q domain.GroupQuery) (domain.ActGroups, error)
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
	Extract(ctx context.Context, r io.Reader, source string) (string, error)
}

type PageExtractor interface {
	ExtractPage(ctx context.Context, r io.Reader, source string, page int) (text string, pages int, err error)
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

type DumpSource interface {
	Snapshot(ctx context.Context) (DumpSnapshot, error)
}

type DumpSnapshot interface {
	Tables() []string
	WriteTable(ctx context.Context, table string, w io.Writer) (rows int, err error)
	Close() error
}

type ObjectWriter interface {
	Put(ctx context.Context, name, contentType string, body io.Reader) error
}

type ErrorReportRepository interface {
	Create(ctx context.Context, r *domain.ErrorReport) error
	List(ctx context.Context, status domain.ReportStatus) ([]domain.ErrorReport, error)
	Close(ctx context.Context, id string, status domain.ReportStatus) error
}

type APIKeyRepository interface {
	Create(ctx context.Context, hash, prefix string) (domain.APIKey, error)
	FindActive(ctx context.Context, hash string) (domain.APIKey, error)
	Revoke(ctx context.Context, hash string) error
	RecordUse(ctx context.Context, keyID, tool string) error
	List(ctx context.Context) ([]domain.APIKey, error)
	RevokeByPrefix(ctx context.Context, prefix string) error
}

type CoverageReader interface {
	Coverage(ctx context.Context) ([]domain.Coverage, error)
}

type EntityReader interface {
	ReportByKey(ctx context.Context, kind domain.EntityKind, key, source string) (domain.EntityReport, error)
}

type FetchRunRepository interface {
	Save(ctx context.Context, r domain.FetchRun) error
}

type ActParsers interface {
	For(source string) ActParser
}

type RegistrySource interface {
	LatestMonth(ctx context.Context) (string, error)
	Files(ctx context.Context, month string) ([]string, error)
	Rows(ctx context.Context, month, file string, each func(fields []string) error) (string, error)
	Codes(ctx context.Context, month string) (domain.RegistryCodes, error)
}

type RegistryRepository interface {
	Ready(ctx context.Context) error
	CitedCNPJs(ctx context.Context) ([]string, error)
	Replace(ctx context.Context, month time.Time, load domain.RegistryLoad) error
}

type PNCPSource interface {
	Contracts(ctx context.Context, org string, year int, each func(c domain.PNCPContract, company bool) error) (string, error)
}

type PNCPRepository interface {
	Ready(ctx context.Context) error
	ReplaceYears(ctx context.Context, from, to int, contracts []domain.PNCPContract) error
}

type PNCPReader interface {
	PNCPContractsBySupplier(ctx context.Context, cnpj string) ([]domain.PNCPContract, error)
}

type StaffSource interface {
	Staff(ctx context.Context, year int) ([]byte, error)
}

type StaffRepository interface {
	Ready(ctx context.Context) error
	ReplaceStaffYear(ctx context.Context, year int, rows []domain.StaffRow) error
}

type StaffReader interface {
	StaffRows(ctx context.Context, unit string) ([]domain.StaffRow, error)
	StaffUnits(ctx context.Context) ([]string, error)
}

type FederalSource interface {
	LatestMonth(ctx context.Context, dataset string) (time.Time, error)
	ZipCSVs(ctx context.Context, file string, each func(name string, header, row []string) error) (string, error)
}

type FederalRepository interface {
	Ready(ctx context.Context) error
	ReplaceAmendments(ctx context.Context, amendments []domain.Amendment, payments []domain.AmendmentPayment) error
	ReplaceTransferMonth(ctx context.Context, month time.Time, transfers []domain.FederalTransfer) error
}

type FederalReader interface {
	Amendments(ctx context.Context) ([]domain.Amendment, error)
	AmendmentPayments(ctx context.Context) ([]domain.AmendmentPayment, error)
	TransferTotals(ctx context.Context) ([]domain.TransferTotal, error)
	TransferMonths(ctx context.Context) (from, to *time.Time, err error)
}

type AmendmentPaymentReader interface {
	AmendmentPaymentsByCNPJ(ctx context.Context, cnpj string) ([]domain.AmendmentPayment, error)
}

type OversightSource interface {
	Dataset(ctx context.Context, name string) ([]byte, error)
}

type OversightRepository interface {
	Ready(ctx context.Context) error
	ReplaceOversight(ctx context.Context, o domain.TCEOversight) error
}

type OversightReader interface {
	Oversight(ctx context.Context) (domain.TCEOversight, error)
}

type StalledWorkReader interface {
	StalledWorksByCNPJ(ctx context.Context, cnpj string) ([]domain.StalledWork, error)
}

type ActMonthCounter interface {
	MonthlyActCounts(ctx context.Context, types []domain.ActType, source string) ([]domain.MonthlyActCount, error)
}

type PaymentSource interface {
	Commitments(ctx context.Context, year int, each func(header, row []string) error) (string, error)
}

type PaymentRepository interface {
	Ready(ctx context.Context) error
	ReplaceYear(ctx context.Context, source string, year int, payments []domain.Payment) error
}

type PaymentReader interface {
	PaymentsByCNPJ(ctx context.Context, cnpj string) ([]domain.PaymentYear, error)
	PaymentsCoverage(ctx context.Context) (*domain.PaymentCoverage, error)
	PaidByCNPJYear(ctx context.Context) ([]domain.PaidTotal, error)
}

type SanctionSource interface {
	LatestDay(ctx context.Context, register string) (time.Time, error)
	Rows(ctx context.Context, register string, day time.Time, each func(header, row []string) error) (string, error)
}

type SanctionRepository interface {
	Ready(ctx context.Context) error
	CitedCNPJs(ctx context.Context) ([]string, error)
	Save(ctx context.Context, load domain.SanctionLoad) error
}

type SanctionReader interface {
	SanctionsByCNPJ(ctx context.Context, cnpj string) ([]domain.Sanction, error)
	SanctionsListedOn(ctx context.Context) (map[string]time.Time, error)
}

type RegistryReader interface {
	RegistryByCNPJ(ctx context.Context, cnpj string) (*domain.CompanyRegistry, error)
	RegistryMonth(ctx context.Context) (*time.Time, error)
	NamesByCNPJ(ctx context.Context, cnpjs []string) (map[string]string, error)
}
