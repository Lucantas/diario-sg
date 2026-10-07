package config

import (
	"fmt"
	"net/netip"
	"os"
	"sort"
	"strings"
)

type Role string

const (
	RoleAPI       Role = "api"
	RoleWorker    Role = "worker"
	RoleMigrate   Role = "migrate"
	RoleReindex   Role = "reindex"
	RoleDump      Role = "dump"
	RoleReports   Role = "reports"
	RoleKeys      Role = "keys"
	RoleReceita   Role = "receita"
	RoleSanctions Role = "sancoes"
	RolePayments  Role = "tce"
	RolePNCP      Role = "pncp"
	RoleFederal   Role = "federal"
	RoleAgents    Role = "agentes"
	RoleSICAM     Role = "sicam"
)

type Config struct {
	Port               string
	DatabaseURL        string
	ProjectID          string
	Bucket             string
	DumpsBucket        string
	DumpDir            string
	TopicIndexed       string
	PubSubEmulatorHost string
	StorageEmulator    string
	Events             string
	PushBaseURL        string
	TrustedProxies     []netip.Prefix
	Notifier           string
	ResendAPIKey       string
	EmailFrom          string
	PublicWebURL       string
	ReceitaBaseURL     string
	ReceitaShareToken  string
	CGUBaseURL         string
	TCEBaseURL         string
	PNCPBaseURL        string
	PrefeituraPayURL   string
	CamaraPayURL       string
	SICAMURL           string
	SiconfiURL         string
	TransferegovURL    string
	PMSGPortalURL      string
	PMSGMuralURL       string
	SIAPEGOVURL        string
	SICAMSiteURL       string
}

func Load(role Role) (Config, error) {
	c := Config{
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		ProjectID:          os.Getenv("GCP_PROJECT_ID"),
		Bucket:             os.Getenv("GAZETTE_BUCKET"),
		DumpsBucket:        os.Getenv("DUMPS_BUCKET"),
		DumpDir:            os.Getenv("DUMP_DIR"),
		TopicIndexed:       os.Getenv("TOPIC_GAZETTE_INDEXED"),
		PubSubEmulatorHost: os.Getenv("PUBSUB_EMULATOR_HOST"),
		StorageEmulator:    os.Getenv("STORAGE_EMULATOR_HOST"),
		Events:             os.Getenv("EVENTS"),
		PushBaseURL:        os.Getenv("PUSH_BASE_URL"),
		Notifier:           getenv("NOTIFIER", "log"),
		ResendAPIKey:       os.Getenv("RESEND_API_KEY"),
		EmailFrom:          os.Getenv("EMAIL_FROM"),
		PublicWebURL:       getenv("PUBLIC_WEB_URL", "http://localhost:5173"),
		ReceitaBaseURL:     getenv("RECEITA_BASE_URL", "https://arquivos.receitafederal.gov.br/public.php/webdav/"),
		ReceitaShareToken:  getenv("RECEITA_SHARE_TOKEN", "YggdBLfdninEJX9"),
		CGUBaseURL:         getenv("CGU_BASE_URL", "https://portaldatransparencia.gov.br/download-de-dados/"),
		TCEBaseURL:         getenv("TCE_BASE_URL", "https://dados.tcerj.tc.br/api/v1/"),
		PNCPBaseURL:        getenv("PNCP_BASE_URL", "https://pncp.gov.br/api/consulta/v1/"),
		PrefeituraPayURL: getenv("PREFEITURA_PAY_URL",
			"https://sistema.pmsg.rj.gov.br/pmsaogoncalo/websis/portal_transparencia/financeiro/contas_publicas/lai_remuneracoes_api.php"),
		CamaraPayURL:    getenv("CAMARA_PAY_URL", "https://cmsaogoncalo-rj.portaltp.com.br/api/pessoal/api-servidores.aspx"),
		SICAMURL:        getenv("SICAM_URL", "https://sg.sicam.app/integracao/"),
		SiconfiURL:      getenv("SICONFI_URL", "https://apidatalake.tesouro.gov.br/ords/siconfi/tt/"),
		TransferegovURL: getenv("TRANSFEREGOV_URL", "https://api.transferegov.gestao.gov.br/transferenciasespeciais/"),
		PMSGPortalURL:   getenv("PMSG_PORTAL_URL", "https://sistema.pmsg.rj.gov.br/portal-transparencia/api/"),
		PMSGMuralURL:    getenv("PMSG_MURAL_URL", "https://licitacao.pmsg.rj.gov.br/"),
		SIAPEGOVURL:     getenv("SIAPEGOV_URL", "https://sistema.pmsg.rj.gov.br/pmsaogoncalo/websis/siapegov/legislativo/leis/"),
		SICAMSiteURL:    getenv("SICAM_SITE_URL", "https://sg.sicam.app/"),
	}

	proxies, err := parsePrefixes(os.Getenv("TRUSTED_PROXIES"))
	if err != nil {
		return c, err
	}
	c.TrustedProxies = proxies

	required := map[string]string{"DATABASE_URL": c.DatabaseURL}
	if role == RoleWorker {
		required["GCP_PROJECT_ID"] = c.ProjectID
		required["GAZETTE_BUCKET"] = c.Bucket
		required["TOPIC_GAZETTE_INDEXED"] = c.TopicIndexed
	}
	if role == RoleReindex || role == RoleReceita || role == RoleSanctions || role == RolePayments || role == RolePNCP || role == RoleFederal || role == RoleAgents || role == RoleSICAM {
		required["GAZETTE_BUCKET"] = c.Bucket
	}
	if role == RoleDump && c.DumpsBucket == "" && c.DumpDir == "" {
		return c, fmt.Errorf("variáveis obrigatórias ausentes: DUMPS_BUCKET ou DUMP_DIR")
	}
	if (role == RoleAPI || role == RoleWorker) && c.Notifier == "resend" {
		required["RESEND_API_KEY"] = c.ResendAPIKey
		required["EMAIL_FROM"] = c.EmailFrom
	}
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return c, fmt.Errorf("variáveis obrigatórias ausentes: %s", strings.Join(missing, ", "))
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parsePrefixes(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, err := netip.ParsePrefix(part)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXIES: faixa inválida %q", part)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}
