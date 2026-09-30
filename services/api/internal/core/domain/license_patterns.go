package domain

import (
	"fmt"
	"regexp"
	"sort"
	"time"
)

const NewCompanyLicenseDays = 730

type LicenseAct struct {
	ActID       string
	CNPJ        string
	PublishedAt time.Time
	Title       string
	Head        string
}

type NewCompanyLicense struct {
	Profile SupplierProfile
	License LicenseAct
	Days    int
	ActIDs  []string
}

var (
	licenseWithdrawalRe = regexp.MustCompile(`(?i)^(?:cancelamento|suspens[ãa]o|cassa[çc][ãa]o)`)
	licenseRequestRe    = regexp.MustCompile(`(?i)torna p[úu]blico que requereu`)
)

func (l LicenseAct) IsGrant() bool {
	return !licenseWithdrawalRe.MatchString(l.Title) && !licenseRequestRe.MatchString(l.Head)
}

func FindNewCompanyLicenses(licenses []LicenseAct, profiles map[string]SupplierProfile) []NewCompanyLicense {
	sorted := append([]LicenseAct{}, licenses...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].PublishedAt.Before(sorted[j].PublishedAt) })
	index := map[string]int{}
	var out []NewCompanyLicense
	for _, l := range sorted {
		p, ok := profiles[l.CNPJ]
		if !ok || !p.Headquarters || p.OpenedAt == nil || !l.IsGrant() {
			continue
		}
		days := int(l.PublishedAt.Sub(*p.OpenedAt).Hours() / 24)
		if days < 0 || days >= NewCompanyLicenseDays {
			continue
		}
		if i, ok := index[l.CNPJ]; ok {
			out[i].ActIDs = append(out[i].ActIDs, l.ActID)
			continue
		}
		index[l.CNPJ] = len(out)
		out = append(out, NewCompanyLicense{Profile: p, License: l, Days: days, ActIDs: []string{l.ActID}})
	}
	return out
}

func licensePatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternNewCompanyLicense: {
			ID:    PatternNewCompanyLicense,
			Title: "Empresa nova recebe licença ambiental",
			Rule: fmt.Sprintf("Concessão ou renovação de licença ambiental publicada no Diário menos de %d dias depois da abertura da "+
				"matriz da empresa no cadastro da Receita. O limite é maior que o das contratações porque um empreendimento costuma "+
				"ser licenciado meses depois de a empresa ser aberta para ele. Pedido de licença, cancelamento, suspensão e cassação "+
				"não entram.", NewCompanyLicenseDays),
			Caveat: "Empresa aberta para um empreendimento (sociedade de propósito específico) é comum e legal. A licença só aparece " +
				"aqui quando o ato cita o CNPJ. " + registrySourceNote,
		},
	}
}

func NewCompanyLicenseFinding(n NewCompanyLicense) Finding {
	detail := fmt.Sprintf("Aberta em %s; primeira licença publicada em %s.", n.Profile.OpenedAt.Format("02/01/2006"),
		n.License.PublishedAt.Format("02/01/2006"))
	if len(n.ActIDs) > 1 {
		detail += fmt.Sprintf(" %d licenças da empresa nos dois anos depois da abertura.", len(n.ActIDs))
	}
	return Finding{
		Title:    fmt.Sprintf("%s: licença ambiental %d dias depois da abertura", supplierLabel(n.Profile), n.Days),
		Detail:   detail,
		ActIDs:   n.ActIDs,
		Entities: cnpjMentions(n.Profile.CNPJ),
	}
}
