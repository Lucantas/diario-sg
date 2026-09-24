package mcp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
	"github.com/seu-usuario/diario-sg/services/api/internal/core/usecase"
	"github.com/seu-usuario/diario-sg/services/api/internal/presentation/ratelimit"
)

const (
	callsPerKey      = 60
	callsPerInstance = 600
	coverageTTL      = 5 * time.Minute
	keyPrefixField   = "key_prefix"
)

const instructions = "Dados do Diário Oficial da Prefeitura de São Gonçalo (RJ) e do Diário Oficial Eletrônico da Câmara Municipal, " +
	"extraídos automaticamente dos PDFs. O campo diario de cada ato diz de qual dos dois ele é. " +
	"Responda sempre citando a edição, a data e a página de cada ato, com o link da fonte. " +
	"O texto extraído pode ter erro; a edição original é que vale. " +
	"Consulte a ferramenta fontes para saber o período coberto antes de afirmar que algo não foi publicado. " +
	"Não monte perfis de pessoas físicas a partir dos atos."

type Deps struct {
	Search       *usecase.SearchActs
	Read         *usecase.ReadAct
	Entity       *usecase.GetEntity
	Group        *usecase.GroupActs
	Page         *usecase.ReadPage
	Coverage     *usecase.SourceCoverage
	Keys         *usecase.APIKeys
	PublicWebURL string
	Log          *slog.Logger
	Now          func() time.Time
}

type server struct {
	searchActs     *usecase.SearchActs
	readAct        *usecase.ReadAct
	getEntity      *usecase.GetEntity
	groupActs      *usecase.GroupActs
	readPage       *usecase.ReadPage
	sourceCoverage *usecase.SourceCoverage
	keys           *usecase.APIKeys
	webURL         string
	log            *slog.Logger
	now            func() time.Time

	mu         sync.Mutex
	covered    []domain.Coverage
	coveredAt  time.Time
	hasCovered bool
}

func NewHandler(d Deps) http.Handler {
	if d.Now == nil {
		d.Now = time.Now
	}
	s := &server{searchActs: d.Search, readAct: d.Read, getEntity: d.Entity, groupActs: d.Group, readPage: d.Page, sourceCoverage: d.Coverage, keys: d.Keys,
		webURL: strings.TrimRight(d.PublicWebURL, "/"), log: d.Log, now: d.Now}
	srv := sdk.NewServer(&sdk.Implementation{Name: "diario-sg", Title: "Diário SG", Version: "0.1.0", WebsiteURL: s.webURL},
		&sdk.ServerOptions{Instructions: instructions, Logger: d.Log})
	s.register(srv)
	h := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true, Logger: d.Log})
	limiter := ratelimit.New(callsPerKey, callsPerInstance, time.Minute, d.Now)
	bearer := auth.RequireBearerToken(s.verify, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})
	return requireBearerHeader(bearer(limited(limiter, rejectBatch(h))))
}

func (s *server) verify(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	key, err := s.keys.Authenticate(ctx, token)
	if errors.Is(err, domain.ErrUnauthorized) {
		return nil, errInvalidKey{}
	}
	if err != nil {
		s.log.Error("autenticação do MCP", "error", err)
		return nil, errInternal
	}
	return &auth.TokenInfo{UserID: key.ID, Extra: map[string]any{keyPrefixField: key.Prefix}}, nil
}

type errInvalidKey struct{}

func (errInvalidKey) Error() string { return domain.ErrUnauthorized.Error() }

func (errInvalidKey) Is(target error) bool { return target == auth.ErrInvalidToken }

func requireBearerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields := strings.Fields(r.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="diario-sg"`)
			http.Error(w, domain.ErrUnauthorized.Error()+"; gere a sua em /mcp no site", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func limited(l *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if info := auth.TokenInfoFromContext(r.Context()); info == nil || !l.Allow(info.UserID) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "muitas chamadas seguidas; tente de novo em um minuto", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rejectBatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "corpo da requisição grande demais", http.StatusRequestEntityTooLarge)
			return
		}
		if trimmed := bytes.TrimLeft(body, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '[' {
			http.Error(w, "lote JSON-RPC não é aceito; mande uma chamada por requisição", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

func keyFrom(req *sdk.CallToolRequest) (id, prefix string) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return "", ""
	}
	prefix, _ = req.Extra.TokenInfo.Extra[keyPrefixField].(string)
	return req.Extra.TokenInfo.UserID, prefix
}

func (s *server) cachedCoverage(ctx context.Context) ([]domain.Coverage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasCovered && s.now().Sub(s.coveredAt) < coverageTTL {
		return s.covered, nil
	}
	c, err := s.sourceCoverage.Execute(ctx)
	if err != nil {
		return c, err
	}
	s.covered, s.coveredAt, s.hasCovered = c, s.now(), true
	return c, nil
}
