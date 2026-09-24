package domain

import (
	"regexp"
	"strconv"
)

type Modality string

const (
	ModalityDispensa        Modality = "dispensa"
	ModalityInexigibilidade Modality = "inexigibilidade"
	ModalityPregao          Modality = "pregao"
	ModalityConcorrencia    Modality = "concorrencia"
	ModalityTomadaDePrecos  Modality = "tomada_de_precos"
	ModalityConvite         Modality = "convite"
	ModalityChamamento      Modality = "chamamento_publico"
	ModalityCredenciamento  Modality = "credenciamento"
	ModalityAdesaoAta       Modality = "adesao_ata"
	ModalityLeilao          Modality = "leilao"
)

var modalityPatterns = []struct {
	modality Modality
	re       *regexp.Regexp
}{
	{ModalityInexigibilidade, regexp.MustCompile(`(?i)\binexigibilidade|\binexig[íi]vel`)},
	{ModalityDispensa, regexp.MustCompile(`(?i)\bdispensa\s+(?:de\s+licita|eletr[ôo]nica)|\blicita[çc][ãa]o\s+dispensada|\bdispens[áa]vel`)},
	{ModalityPregao, regexp.MustCompile(`(?i)\bpreg[ãa]o`)},
	{ModalityConcorrencia, regexp.MustCompile(`(?i)\bconcorr[êe]ncia`)},
	{ModalityTomadaDePrecos, regexp.MustCompile(`(?i)\btomada\s+de\s+pre[çc]os`)},
	{ModalityConvite, regexp.MustCompile(`(?i)\bcarta[- ]convite|\bmodalidade\s+convite`)},
	{ModalityChamamento, regexp.MustCompile(`(?i)\bchamamento\s+p[úu]blico`)},
	{ModalityCredenciamento, regexp.MustCompile(`(?i)\bcredenciamento`)},
	{ModalityAdesaoAta, regexp.MustCompile(`(?i)\bades[ãa]o\s+[àa]\s+ata|\bcarona\b`)},
	{ModalityLeilao, regexp.MustCompile(`(?i)\bleil[ãa]o`)},
}

var contractingTypes = map[ActType]bool{
	ActContrato: true, ActAditivo: true, ActDispensa: true, ActLicitacao: true,
	ActAta: true, ActEdital: true, ActOutro: true, ActDespacho: true,
}

func (m Modality) Valid() bool {
	for _, p := range modalityPatterns {
		if p.modality == m {
			return true
		}
	}
	return false
}

func ModalityOf(t ActType, title, body string) Modality {
	if !contractingTypes[t] {
		return ""
	}
	text := title + "\n" + body
	best, bestAt := Modality(""), len(text)+1
	for _, p := range modalityPatterns {
		if loc := p.re.FindStringIndex(text); loc != nil && loc[0] < bestAt {
			best, bestAt = p.modality, loc[0]
		}
	}
	return best
}

const moneyAfterLabel = `[^$]{0,40}R\$\s?(\d{1,3}(?:\.\d{3})+|\d+),(\d{2})\b`

var mainValueRes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:valor\s+(?:global|total|do\s+contrato|do\s+aditivo|do\s+termo|contratado|registrado|homologado|adjudicado)|no\s+valor\s+(?:global|total)\s+de|pelo\s+valor\s+(?:global\s+|total\s+)?de|import[âa]ncia\s+de|acr[ée]scimo\s+de|montante\s+de)` + moneyAfterLabel),
	regexp.MustCompile(`(?i)(?:valor\s+(?:estimado|anual|mensal)|no\s+valor\s+de)` + moneyAfterLabel),
}

func MainValueCents(t ActType, body string) int64 {
	if !contractingTypes[t] && t != ActPrestacaoContas {
		return 0
	}
	for _, re := range mainValueRes {
		if m := re.FindStringSubmatch(body); m != nil {
			cents, err := strconv.ParseInt(nonDigitRe.ReplaceAllString(m[1], "")+m[2], 10, 64)
			if err != nil {
				return 0
			}
			return cents
		}
	}
	return 0
}
