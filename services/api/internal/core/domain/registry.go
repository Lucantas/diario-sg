package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	SourceReceita       = "receita_cnpj"
	RecordEstablishment = "estabelecimento"

	cnpjBaseDigits      = 8
	companyFields       = 7
	establishmentFields = 30
	partnerFields       = 11
	registryDateLayout  = "20060102"
	centsDigits         = 2
)

type PartnerKind int

const (
	PartnerCompany PartnerKind = 1
	PartnerPerson  PartnerKind = 2
	PartnerForeign PartnerKind = 3
)

type Activity struct {
	Code        string
	Description string
}

type RegistryCompany struct {
	Base         string
	Name         string
	LegalNature  string
	CapitalCents int64
	Size         string
}

type RegistryEstablishment struct {
	CNPJ            string
	Headquarters    bool
	TradeName       string
	Status          string
	StatusSince     *time.Time
	StatusReason    string
	OpenedAt        *time.Time
	MainActivity    Activity
	OtherActivities []Activity
	Street          string
	Number          string
	Complement      string
	District        string
	ZIP             string
	City            string
	UF              string
}

type RegistryPartner struct {
	Base     string
	Kind     PartnerKind
	Name     string
	Document string
	Role     string
	Since    *time.Time
}

type CompanyRegistry struct {
	Month         time.Time
	Company       RegistryCompany
	Establishment RegistryEstablishment
	Partners      []RegistryPartner
}

type RegistryLoad struct {
	Companies      []RegistryCompany
	Establishments []RegistryEstablishment
	Partners       []RegistryPartner
}

type RegistryCodes struct {
	Activities map[string]string
	Cities     map[string]string
	Natures    map[string]string
	Roles      map[string]string
	Reasons    map[string]string
}

var companySizes = map[string]string{
	"00": "Não informado", "01": "Microempresa", "03": "Empresa de pequeno porte", "05": "Demais",
}

var establishmentStatuses = map[string]string{
	"01": "Nula", "02": "Ativa", "03": "Suspensa", "04": "Inapta", "08": "Baixada",
}

func CNPJBase(cnpj string) string {
	if len(cnpj) < cnpjBaseDigits {
		return cnpj
	}
	return cnpj[:cnpjBaseDigits]
}

func ParseCompanyRow(f []string, c RegistryCodes) (RegistryCompany, error) {
	if len(f) != companyFields {
		return RegistryCompany{}, fieldCountError("empresa", companyFields, len(f))
	}
	capital, err := parseRegistryCents(f[4])
	if err != nil {
		return RegistryCompany{}, fmt.Errorf("capital social de %s: %w", f[0], err)
	}
	return RegistryCompany{
		Base: f[0], Name: squeezed(f[1]), LegalNature: described(c.Natures, f[2]),
		CapitalCents: capital, Size: described(companySizes, f[5]),
	}, nil
}

func ParseEstablishmentRow(f []string, c RegistryCodes) (RegistryEstablishment, error) {
	if len(f) != establishmentFields {
		return RegistryEstablishment{}, fieldCountError("estabelecimento", establishmentFields, len(f))
	}
	return RegistryEstablishment{
		CNPJ:            f[0] + f[1] + f[2],
		Headquarters:    f[3] == "1",
		TradeName:       squeezed(f[4]),
		Status:          described(establishmentStatuses, f[5]),
		StatusSince:     parseRegistryDate(f[6]),
		StatusReason:    described(c.Reasons, f[7]),
		OpenedAt:        parseRegistryDate(f[10]),
		MainActivity:    activity(c, f[11]),
		OtherActivities: activities(c, f[12]),
		Street:          squeezed(f[13] + " " + f[14]),
		Number:          squeezed(f[15]),
		Complement:      squeezed(f[16]),
		District:        squeezed(f[17]),
		ZIP:             f[18],
		UF:              f[19],
		City:            described(c.Cities, f[20]),
	}, nil
}

func ParsePartnerRow(f []string, c RegistryCodes) (RegistryPartner, error) {
	if len(f) != partnerFields {
		return RegistryPartner{}, fieldCountError("sócio", partnerFields, len(f))
	}
	kind, err := strconv.Atoi(f[1])
	if err != nil || kind < int(PartnerCompany) || kind > int(PartnerForeign) {
		return RegistryPartner{}, fmt.Errorf("tipo de sócio %q de %s", f[1], f[0])
	}
	return RegistryPartner{
		Base: f[0], Kind: PartnerKind(kind), Name: squeezed(f[2]), Document: f[3],
		Role: described(c.Roles, f[4]), Since: parseRegistryDate(f[5]),
	}, nil
}

func fieldCountError(what string, want, got int) error {
	return fmt.Errorf("linha de %s com %d campos, esperava %d", what, got, want)
}

func described(codes map[string]string, code string) string {
	if d, ok := codes[code]; ok {
		return d
	}
	return code
}

func activity(c RegistryCodes, code string) Activity {
	return Activity{Code: code, Description: c.Activities[code]}
}

func activities(c RegistryCodes, codes string) []Activity {
	out := []Activity{}
	for _, code := range strings.Split(codes, ",") {
		if code = strings.TrimSpace(code); code != "" {
			out = append(out, activity(c, code))
		}
	}
	return out
}

func parseRegistryDate(s string) *time.Time {
	t, err := time.Parse(registryDateLayout, s)
	if err != nil {
		return nil
	}
	return &t
}

func parseRegistryCents(s string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ",")
	frac = (frac + strings.Repeat("0", centsDigits))[:centsDigits]
	return strconv.ParseInt(whole+frac, 10, 64)
}

func squeezed(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
