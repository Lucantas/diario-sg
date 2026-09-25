package domain

import (
	"fmt"
	"strconv"
	"strings"
)

type PatternID string

const (
	PatternSplitDispensa  PatternID = "fracionamento_dispensa"
	PatternElectionHiring PatternID = "pico_pessoal_eleicao"
)

type Pattern struct {
	ID                  PatternID
	Title, Rule, Caveat string
}

type Finding struct {
	Title  string
	Detail string
	ActIDs []string
	Search *ActFilter
}

type PatternReport struct {
	Pattern  Pattern
	Findings []Finding
}

var monthNames = [...]string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}

var dispensaCategoryLabel = map[DispensaCategory]string{DispensaGoods: "compras e serviços", DispensaWorks: "obras e serviços de engenharia"}

var peakTypeLabel = map[ActType]string{ActNomeacao: "nomeação", ActExoneracao: "exoneração"}

func PatternCatalog() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternSplitDispensa: {
			ID:    PatternSplitDispensa,
			Title: "Dispensas do mesmo fornecedor que, somadas no ano, passam do limite",
			Rule: "Duas ou mais contratações do mesmo CNPJ no mesmo ano e da mesma categoria, cada uma por dispensa de licitação em " +
				"razão do valor, cada uma abaixo do limite da dispensa e com soma acima dele. Compras e serviços: art. 24, II, da Lei " +
				"8.666 ou art. 75, II, da Lei 14.133, com limite de R$ 8.000,00 até 18/07/2018, R$ 17.600,00 depois, e pela Lei 14.133 " +
				"R$ 50.000,00, atualizados todo ano (R$ 65.492,11 em 2026). Obras e serviços de engenharia: art. 24, I, ou art. 75, I, " +
				"com limite de R$ 15.000,00 até 18/07/2018, R$ 33.000,00 depois, e pela Lei 14.133 R$ 100.000,00, atualizados todo ano " +
				"(R$ 130.984,20 em 2026). Uma contratação reúne os atos que citam o mesmo processo; republicações e atos sem número de " +
				"processo com o mesmo valor no mesmo ano contam como a mesma contratação.",
			Caveat: "A lei soma o que cada unidade gestora gasta no ano com objetos de mesma natureza. O Diário não diz a natureza do " +
				"objeto de forma padronizada, e o mesmo fornecedor pode vender coisas diferentes para órgãos diferentes. Só entram atos " +
				"em que o CNPJ e o valor foram lidos do texto, então parte das dispensas fica de fora.",
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
		FormatCNPJ(s.CNPJ), s.Year, len(s.Contracts), dispensaCategoryLabel[s.Category], FormatBRL(s.TotalCents), FormatBRL(s.LimitCents))}
	parts := make([]string, 0, len(s.Contracts))
	for _, c := range s.Contracts {
		parts = append(parts, contractSummary(c))
		f.ActIDs = append(f.ActIDs, c.ActIDs...)
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
