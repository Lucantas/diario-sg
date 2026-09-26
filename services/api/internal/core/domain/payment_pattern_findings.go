package domain

import (
	"fmt"
	"strings"
)

const tceSourceNote = "O pago vem dos empenhos do TCE-RJ (de 2021 em diante; 2020 só tem o empenhado), que não dizem a que " +
	"contrato cada pagamento se refere."

func paymentPatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternPaidUnpublished: {
			ID:    PatternPaidUnpublished,
			Title: "Empresa paga sem nenhuma publicação no Diário",
			Rule: "Credor pessoa jurídica que recebeu da Prefeitura, de seus fundos e fundações pelo menos " +
				FormatBRL(MinPaidWithoutPublicationCents) + " de 2021 em diante sem que nenhum ato do Diário da Prefeitura ou da Câmara " +
				"cite o seu CNPJ. Ficam de fora os pagamentos de encargos especiais (dívida, precatórios, PASEP) e de previdência, e os " +
				"credores de direito público.",
			Caveat: "O extrato pode ter sido publicado sem o CNPJ, só com o nome da empresa, ou com o CNPJ digitado errado: procure o " +
				"nome na busca. Concessionárias de serviço público e contratos anteriores a 2010 também podem explicar o pagamento. " +
				tceSourceNote,
		},
		PatternUnpaidContract: {
			ID:    PatternUnpaidContract,
			Title: "Contratação publicada sem nenhum pagamento à empresa",
			Rule: "Contratação de pelo menos " + FormatBRL(MinUnpaidContractCents) + " publicada no Diário de 2021 em diante sem nenhum " +
				"pagamento à empresa (qualquer unidade da Prefeitura) no ano da publicação nem no seguinte.",
			Caveat: "O contrato pode ter sido rescindido ou anulado, ou pago com outro CNPJ da mesma empresa. A contratação publicada no " +
				"fim do ano pode ter pagamento só dois anos depois. " + tceSourceNote,
		},
		PatternPaidAbove: {
			ID:    PatternPaidAbove,
			Title: "Empresa paga muito acima do que o Diário anunciou",
			Rule: "A empresa recebeu de 2021 em diante pelo menos o dobro de tudo o que o Diário da Prefeitura anunciou para ela desde " +
				"2010 (contratações, atas e aditivos, como nos painéis), com diferença de pelo menos " + FormatBRL(MinPaidAboveAnnouncedCents) +
				". Extratos sem CNPJ contam quando o nome do fornecedor lido do texto é igual à razão social da empresa.",
			Caveat: "O Diário não traz o valor de todo contrato (reajustes, apostilamentos, contratos de gestão e serviços contínuos " +
				"renovados sem valor no extrato), e o nome lido do texto pode não bater com a razão social. " + tceSourceNote,
		},
	}
}

func PaidWithoutPublicationFinding(p PaidWithoutPublication) Finding {
	return Finding{
		Title: fmt.Sprintf("%s: %s pagos de %s sem nenhuma citação no Diário", supplierLabel(p.Profile), FormatBRL(p.Paid.PaidCents),
			paymentYearsLabel(p.Paid.Years)),
		Detail: "Unidades que pagaram: " + strings.Join(p.Paid.Units, "; ") + ".",
	}
}

func UnpaidContractFinding(u UnpaidContract) Finding {
	next := u.Contract.First.Year() + unpaidGraceYears
	return Finding{
		Title: fmt.Sprintf("%s: contratação de %s publicada em %s, sem pagamento em %d nem em %d", supplierLabel(u.Profile),
			FormatBRL(u.Contract.ContractedCents), u.Contract.First.Format("02/01/2006"), u.Contract.First.Year(), next),
		Detail: "Órgão no Diário: " + orEmptyLabel(strings.Join(u.Contract.Organs, ", ")) + ".",
		ActIDs: []string{u.Contract.ActID},
	}
}

func PaidAboveAnnouncedFinding(p PaidAboveAnnounced) Finding {
	f := Finding{
		Title: fmt.Sprintf("%s: %s pagos de %s, contra %s anunciados no Diário", supplierLabel(p.Profile), FormatBRL(p.Paid.PaidCents),
			paymentYearsLabel(p.Paid.Years), FormatBRL(p.AnnouncedCents)),
		Detail: fmt.Sprintf("%s no Diário. Unidades que pagaram: %s.", contractsCount(len(p.Contracts)), strings.Join(p.Paid.Units, "; ")),
	}
	for _, c := range p.Contracts {
		f.ActIDs = append(f.ActIDs, c.ActID)
	}
	return f
}

func orEmptyLabel(s string) string {
	if s == "" {
		return "não identificado"
	}
	return s
}
