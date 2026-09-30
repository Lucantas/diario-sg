package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type PatternID string

const (
	PatternSplitDispensa     PatternID = "fracionamento_dispensa"
	PatternElectionHiring    PatternID = "pico_pessoal_eleicao"
	PatternExcessiveAddenda  PatternID = "aditivo_acima_do_limite"
	PatternRenewedEmergency  PatternID = "emergencial_renovada"
	PatternNewCompany        PatternID = "empresa_nova_contratada"
	PatternUndercapitalized  PatternID = "capital_menor_que_contrato"
	PatternSharedPartner     PatternID = "socio_em_comum"
	PatternSharedAddress     PatternID = "endereco_em_comum"
	PatternSanctioned        PatternID = "sancionado_contratado"
	PatternPaidUnpublished   PatternID = "pago_sem_publicacao"
	PatternUnpaidContract    PatternID = "contrato_sem_pagamento"
	PatternPaidAbove         PatternID = "pago_acima_do_anunciado"
	PatternNewCompanyLicense PatternID = "empresa_nova_licenciada"
)

type Pattern struct {
	ID                  PatternID
	Title, Rule, Caveat string
}

type Finding struct {
	Title    string
	Detail   string
	ActIDs   []string
	Entities []EntityMention
	Search   *ActFilter
	Link     *FindingLink
}

func (f Finding) Mentions(kind EntityKind, key string) bool {
	for _, e := range f.Entities {
		if e.Kind == kind && e.Key == key {
			return true
		}
	}
	return false
}

func cnpjMentions(cnpjs ...string) []EntityMention {
	var out []EntityMention
	for _, c := range cnpjs {
		if n, ok := NormalizeCNPJ(c); ok {
			out = append(out, EntityMention{Kind: EntityCNPJ, Key: n, Label: FormatCNPJ(n)})
		}
	}
	return out
}

func refMentions(refs []string) []EntityMention {
	var out []EntityMention
	seen := map[string]bool{}
	for _, ref := range refs {
		kind, key, ok := strings.Cut(ref, ":")
		if !ok || key == "" || seen[ref] || (EntityKind(kind) != EntityProcesso && EntityKind(kind) != EntityContrato) {
			continue
		}
		seen[ref] = true
		out = append(out, EntityMention{Kind: EntityKind(kind), Key: key, Label: EntityLabel(EntityKind(kind), key)})
	}
	return out
}

type FindingLink struct {
	Label, URL string
}

type PatternReport struct {
	Pattern  Pattern
	Findings []Finding
}

var monthNames = [...]string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

var dispensaCategoryLabel = map[DispensaCategory]string{DispensaGoods: "compras e serviços", DispensaWorks: "obras e serviços de engenharia"}

var peakTypeLabel = map[ActType]string{ActNomeacao: "nomeação", ActExoneracao: "exoneração"}

func PatternCatalog() map[PatternID]Pattern {
	catalog := diarioPatterns()
	for id, p := range supplierPatterns() {
		catalog[id] = p
	}
	for id, p := range paymentPatterns() {
		catalog[id] = p
	}
	for id, p := range pncpPatterns() {
		catalog[id] = p
	}
	for id, p := range licensePatterns() {
		catalog[id] = p
	}
	return catalog
}

func diarioPatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternSplitDispensa: {
			ID:    PatternSplitDispensa,
			Title: "Dispensas do mesmo fornecedor que, somadas no ano, passam do limite",
			Rule: "Duas ou mais contratações do mesmo CNPJ no mesmo ano e da mesma categoria, cada uma por dispensa de licitação em " +
				"razão do valor, cada uma abaixo do limite da dispensa e com soma acima dele. Compras e serviços: art. 24, II, da Lei " +
				"8.666 ou art. 75, II, da Lei 14.133, com limite de R$ 8.000,00 até 18/07/2018, R$ 17.600,00 depois, e pela Lei 14.133 " +
				"R$ 50.000,00, atualizados todo ano (R$ 65.492,11 em 2026). Obras e serviços de engenharia (e, pela Lei 14.133, " +
				"manutenção de veículos): art. 24, I, ou art. 75, I, " +
				"com limite de R$ 15.000,00 até 18/07/2018, R$ 33.000,00 depois, e pela Lei 14.133 R$ 100.000,00, atualizados todo ano " +
				"(R$ 130.984,20 em 2026). Uma contratação reúne os atos que citam o mesmo processo; republicações e atos sem número de " +
				"processo com o mesmo valor no mesmo ano contam como a mesma contratação.",
			Caveat: "A lei soma o que cada unidade gestora gasta no ano com objetos de mesma natureza. O Diário não diz a natureza do " +
				"objeto de forma padronizada, e o mesmo fornecedor pode vender coisas diferentes para órgãos diferentes. Só entram atos " +
				"em que o CNPJ e o valor foram lidos do texto, então parte das dispensas fica de fora.",
		},
		PatternExcessiveAddenda: {
			ID:    PatternExcessiveAddenda,
			Title: "Aditivos que somam mais acréscimo do que a lei permite",
			Rule: "Aditivos do mesmo contrato (mesmo número de contrato e mesmo órgão) cujos acréscimos declarados no texto somam " +
				"mais de 25% do valor inicial, ou mais de 50% quando o contrato é de reforma (art. 65, § 1º, da Lei 8.666; art. 125 da " +
				"Lei 14.133). Só entram os percentuais que o próprio aditivo declara como acréscimo; reajuste e retificação não contam, " +
				"supressões não abatem, e cada aditivo (primeiro, segundo…) conta uma vez.",
			Caveat: "O limite vale para acréscimos de quantidade ou de objeto, não para reajuste de preço, e o texto nem sempre separa " +
				"os dois. Aditivos que só dizem o valor em reais ficam de fora, porque o valor declarado costuma ser o novo total do " +
				"contrato. O percentual pode ter sido calculado sobre bases diferentes em cada aditivo.",
		},
		PatternRenewedEmergency: {
			ID:    PatternRenewedEmergency,
			Title: "Contratação emergencial da mesma empresa, uma depois da outra",
			Rule: "Duas ou mais contratações por dispensa emergencial (art. 24, IV, da Lei 8.666 ou art. 75, VIII, da Lei 14.133) da " +
				"mesma empresa no mesmo órgão, cada uma começando entre 30 dias e 24 meses depois da anterior. A contratação reúne os " +
				"atos que citam o mesmo processo ou contrato; a empresa é o CNPJ ou, sem CNPJ no texto, o nome lido do ato. A Lei " +
				"8.666 veda prorrogar a contratação emergencial, e a Lei 14.133 veda também recontratar a mesma empresa.",
			Caveat: "Uma nova emergência pode justificar outra contratação. O nome da empresa é lido do texto e pode vir incompleto; " +
				"contratações no mesmo mês, para objetos diferentes, não contam como renovação. A prorrogação por aditivo do mesmo " +
				"contrato conta como a mesma contratação e não aparece aqui.",
		},
		PatternElectionHiring: {
			ID:    PatternElectionHiring,
			Title: "Picos de nomeação e de exoneração antes da eleição municipal",
			Rule: "Nos seis meses anteriores ao mês de cada eleição municipal (2012, 2016, 2020 e 2024), os meses em que o Diário da " +
				"Prefeitura publicou pelo menos 1,5 vez a mediana de atos de nomeação (ou de exoneração) do mesmo mês nos anos sem " +
				"eleição municipal.",
			Caveat: "Conta atos, não pessoas: uma portaria pode nomear ou exonerar várias. Um pico pode ter explicação comum, como troca " +
				"de secretário ou posse de concursados; o padrão só aponta o mês para verificar.",
		},
	}
}

func SplitDispensaFinding(s SplitDispensa) Finding {
	f := Finding{Title: fmt.Sprintf("CNPJ %s em %d: %d dispensas de %s somam %s, acima do limite de %s",
		FormatCNPJ(s.CNPJ), s.Year, len(s.Contracts), dispensaCategoryLabel[s.Category], FormatBRL(s.TotalCents), FormatBRL(s.LimitCents)),
		Entities: cnpjMentions(s.CNPJ)}
	parts := make([]string, 0, len(s.Contracts))
	for _, c := range s.Contracts {
		parts = append(parts, contractSummary(c))
		f.ActIDs = append(f.ActIDs, c.ActIDs...)
		for _, p := range c.Processes {
			if !f.Mentions(EntityProcesso, p.Key) {
				f.Entities = append(f.Entities, EntityMention{Kind: EntityProcesso, Key: p.Key, Label: p.Label})
			}
		}
	}
	f.Detail = strings.Join(parts, " ")
	return f
}

func contractSummary(c DispensaContract) string {
	labels := make([]string, 0, len(c.Processes))
	for _, p := range c.Processes {
		labels = append(labels, p.Label)
	}
	var who string
	switch len(labels) {
	case 0:
		who = "Sem número de processo"
	case 1:
		who = "Processo " + labels[0]
	default:
		who = "Processos " + strings.Join(labels[:len(labels)-1], ", ") + " e " + labels[len(labels)-1]
	}
	if len(c.Organs) > 0 {
		who += " (" + strings.Join(c.Organs, ", ") + ")"
	}
	return fmt.Sprintf("%s, %s: %s.", who, c.FirstPublished.Format("02/01/2006"), FormatBRL(c.ValueCents))
}

func ElectionPeakFinding(p HiringPeak) Finding {
	first := civilDate(p.Year, p.Month, 1)
	month := monthNames[p.Month]
	return Finding{
		Title: fmt.Sprintf("%s de %d: %d atos de %s", strings.ToUpper(month[:1])+month[1:], p.Year, p.Count, peakTypeLabel[p.Type]),
		Detail: fmt.Sprintf("Mediana de %s nos %d anos sem eleição municipal: %s. Eleição em %s.",
			month, p.BaselineYears, strings.Replace(strconv.FormatFloat(p.BaselineMedian, 'f', -1, 64), ".", ",", 1), p.Election.Format("02/01/2006")),
		Search: &ActFilter{Type: p.Type, Source: SourceDiarioPrefeitura, From: first, To: first.AddDate(0, 1, -1)},
	}
}

var ordinalLabels = [...]string{"", "Primeiro", "Segundo", "Terceiro", "Quarto", "Quinto", "Sexto", "Sétimo", "Oitavo", "Nono", "Décimo"}

func ExcessiveAddendumFinding(e ExcessiveAddendum) Finding {
	f := Finding{Title: fmt.Sprintf("Contrato %s%s: aditivos somam %s de acréscimo, acima do limite de %s",
		e.ContractKey, organSuffix(e.Organ), FormatPercentBP(e.TotalBP), FormatPercentBP(e.LimitBP)),
		Entities: []EntityMention{{Kind: EntityContrato, Key: e.ContractKey, Label: e.ContractKey}}}
	parts := make([]string, 0, len(e.Acts))
	for _, a := range e.Acts {
		label := "Aditivo"
		if ord := AddendumOrdinal(a.Title, a.Body); ord > 0 && ord < len(ordinalLabels) {
			label = ordinalLabels[ord] + " termo aditivo"
		}
		parts = append(parts, fmt.Sprintf("%s, %s: %s.", label, a.PublishedAt.Format("02/01/2006"), FormatPercentBP(a.IncreaseBP)))
		f.ActIDs = append(f.ActIDs, a.ActID)
	}
	f.Detail = strings.Join(parts, " ")
	return f
}

func RenewedEmergencyFinding(r RenewedEmergency) Finding {
	first, last := r.Contracts[0].First, r.Contracts[len(r.Contracts)-1].First
	f := Finding{Title: fmt.Sprintf("%s%s: %d contratações emergenciais seguidas, de %s a %s",
		r.SupplierLabel, organInSuffix(r.Organ), len(r.Contracts), first.Format("02/01/2006"), last.Format("02/01/2006"))}
	if cnpj, ok := strings.CutPrefix(r.Supplier, cnpjSupplierPrefix); ok {
		f.Entities = cnpjMentions(cnpj)
	}
	var refs []string
	for _, c := range r.Contracts {
		refs = append(refs, c.Refs...)
	}
	f.Entities = append(f.Entities, refMentions(refs)...)
	dates := make([]string, 0, len(r.Contracts))
	for _, c := range r.Contracts {
		dates = append(dates, c.First.Format("02/01/2006"))
		f.ActIDs = append(f.ActIDs, c.ActIDs...)
	}
	f.Detail = "Contratações emergenciais em " + joinPortuguese(dates) + "."
	return f
}

func FormatPercentBP(bp int) string {
	if bp%basisPointsPerPoint == 0 {
		return fmt.Sprintf("%d%%", bp/basisPointsPerPoint)
	}
	return fmt.Sprintf("%d,%02d%%", bp/basisPointsPerPoint, bp%basisPointsPerPoint)
}

func organSuffix(organ string) string {
	if organ == "" {
		return ""
	}
	return " (" + organ + ")"
}

func organInSuffix(organ string) string {
	if organ == "" {
		return ""
	}
	return " em " + organ
}

func joinPortuguese(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " e " + items[len(items)-1]
}
