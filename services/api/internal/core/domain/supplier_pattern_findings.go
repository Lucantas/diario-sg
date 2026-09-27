package domain

import (
	"fmt"
	"strings"
)

const registrySourceNote = "Capital, endereço, abertura e sócios são do cadastro mais recente da Receita, não da data do contrato."

func supplierPatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternNewCompany: {
			ID:    PatternNewCompany,
			Title: "Empresa aberta pouco antes da primeira contratação",
			Rule: fmt.Sprintf("A primeira contratação da empresa publicada no Diário da Prefeitura saiu menos de %d dias depois da "+
				"abertura da matriz no cadastro da Receita. A contratação reúne os atos da mesma empresa ligados pelo processo ou pelo "+
				"contrato, como nos painéis.", NewCompanyDays),
			Caveat: "Empresa nova pode ser legítima. O Diário pode não ter publicado contratação anterior, e a data de abertura de " +
				"filial não entra. Contratação publicada antes da abertura fica de fora, porque costuma ser CNPJ digitado errado.",
		},
		PatternUndercapitalized: {
			ID:    PatternUndercapitalized,
			Title: "Capital social menor que 10% do valor contratado",
			Rule: "A maior contratação da empresa, de pelo menos " + FormatBRL(MinUndercapitalizedCents) + ", passa de dez vezes o " +
				"capital social declarado à Receita. Dez por cento do valor da contratação é o máximo de capital mínimo que a lei deixa " +
				"a Administração exigir (art. 31, § 3º, da Lei 8.666; art. 69, § 4º, da Lei 14.133). Só entram sociedades, EIRELI e " +
				"empresários individuais; cooperativas, associações e fundações têm capital de outra natureza.",
			Caveat: "A lei permite, não obriga, exigir capital mínimo, e aceita patrimônio líquido no lugar. " + registrySourceNote +
				" O valor é o declarado no extrato, não o pago.",
		},
		PatternSharedPartner: {
			ID:    PatternSharedPartner,
			Title: "Fornecedores com sócio em comum",
			Rule: "Duas ou mais empresas contratadas (CNPJ básico diferente) com o mesmo sócio no cadastro da Receita, pelo nome e " +
				"pelo documento como a Receita publica. O sócio pessoa física não é nomeado aqui: ele aparece na página de cada empresa.",
			Caveat: "O CPF vem mascarado pela Receita (seis dígitos visíveis), então nome e dígitos iguais indicam, mas não provam, a " +
				"mesma pessoa. Empresas do mesmo grupo podem disputar objetos diferentes. " + registrySourceNote,
		},
		PatternSharedAddress: {
			ID:    PatternSharedAddress,
			Title: "Fornecedores no mesmo endereço",
			Rule: "Duas ou mais empresas contratadas (CNPJ básico diferente) com o mesmo endereço completo no cadastro da Receita: " +
				"logradouro, número, complemento, bairro, cidade e CEP, sem diferença de acento ou pontuação. Endereço sem número não entra.",
			Caveat: "Escritório de contabilidade, sala comercial compartilhada e centro empresarial explicam muitos casos. " + registrySourceNote,
		},
		PatternSanctioned: {
			ID:    PatternSanctioned,
			Title: "Contratação durante sanção que alcança São Gonçalo",
			Rule: "Contratação publicada no Diário entre o início e o fim de uma sanção do CEIS ou do CNEP aplicada à empresa (qualquer " +
				"estabelecimento) que alcança o Município: declaração de inidoneidade, abrangência em todas as esferas e poderes, ou " +
				"órgão sancionador de São Gonçalo. Impedimento e suspensão aplicados por outro ente não entram (art. 156, § 4º, da Lei 14.133).",
			Caveat: "A base só tem as sanções que estavam no CEIS e no CNEP desde a primeira carga diária; sanção antiga que já saiu do " +
				"cadastro não aparece. A data da contratação é a da publicação do primeiro ato, não a da assinatura.",
		},
	}
}

func supplierLabel(p SupplierProfile) string {
	if p.Name == "" {
		return "CNPJ " + FormatCNPJ(p.CNPJ)
	}
	return fmt.Sprintf("%s (%s)", p.Name, FormatCNPJ(p.CNPJ))
}

func contractValueCents(c SupplierContract) int64 {
	if c.ContractedCents > 0 {
		return c.ContractedCents
	}
	return c.RegisteredCents
}

func NewCompanyFinding(n NewCompanyContract) Finding {
	return Finding{
		Title: fmt.Sprintf("%s: primeira contratação %d dias depois da abertura", supplierLabel(n.Profile), n.Days),
		Detail: fmt.Sprintf("Aberta em %s; primeira contratação publicada em %s, de %s%s.", n.Profile.OpenedAt.Format("02/01/2006"),
			n.Contract.First.Format("02/01/2006"), FormatBRL(contractValueCents(n.Contract)), organsSuffix(n.Contract.Organs)),
		ActIDs:   []string{n.Contract.ActID},
		Entities: cnpjMentions(n.Profile.CNPJ),
	}
}

func UndercapitalizedFinding(u UndercapitalizedContract) Finding {
	times := float64(u.Contract.ContractedCents) / float64(u.Profile.CapitalCents)
	return Finding{
		Title: fmt.Sprintf("%s: contratação de %s com capital social de %s", supplierLabel(u.Profile),
			FormatBRL(u.Contract.ContractedCents), FormatBRL(u.Profile.CapitalCents)),
		Detail: fmt.Sprintf("O valor é %s vezes o capital (%s). Contratação publicada em %s%s.",
			strings.Replace(fmt.Sprintf("%.1f", times), ".", ",", 1), u.Profile.LegalNature,
			u.Contract.First.Format("02/01/2006"), organsSuffix(u.Contract.Organs)),
		ActIDs:   []string{u.Contract.ActID},
		Entities: cnpjMentions(u.Profile.CNPJ),
	}
}

func SharedPartnerFinding(g SharedSupplierGroup) Finding {
	var named []string
	people := 0
	for _, p := range g.Partners {
		if p.Kind == PartnerCompany {
			named = append(named, p.Name)
		} else {
			people++
		}
	}
	var who []string
	if people == 1 {
		who = append(who, "1 sócio pessoa física")
	} else if people > 1 {
		who = append(who, fmt.Sprintf("%d sócios pessoa física", people))
	}
	if len(named) > 0 {
		who = append(who, "sócio pessoa jurídica "+joinPortuguese(named))
	}
	f := Finding{Title: fmt.Sprintf("%s: %s em comum", groupNames(g), joinPortuguese(who)), Entities: groupMentions(g)}
	f.Detail, f.ActIDs = groupDetail(g)
	return f
}

func SharedAddressFinding(g SharedSupplierGroup) Finding {
	f := Finding{Title: fmt.Sprintf("%s: mesmo endereço, %s", groupNames(g), g.Address), Entities: groupMentions(g)}
	f.Detail, f.ActIDs = groupDetail(g)
	return f
}

func groupMentions(g SharedSupplierGroup) []EntityMention {
	var out []EntityMention
	for _, s := range g.Suppliers {
		out = append(out, cnpjMentions(s.CNPJ)...)
	}
	return out
}

func groupNames(g SharedSupplierGroup) string {
	names := make([]string, len(g.Suppliers))
	for i, s := range g.Suppliers {
		names[i] = s.Name
		if names[i] == "" {
			names[i] = FormatCNPJ(s.CNPJ)
		}
	}
	return joinPortuguese(names)
}

func groupDetail(g SharedSupplierGroup) (string, []string) {
	parts := make([]string, 0, len(g.Suppliers))
	var ids []string
	for _, s := range g.Suppliers {
		var total int64
		n := 0
		for _, c := range g.Contracts {
			if c.CNPJ == s.CNPJ {
				total += contractValueCents(c)
				n++
				ids = append(ids, c.ActID)
			}
		}
		parts = append(parts, fmt.Sprintf("%s: %s, %s.", supplierLabel(s), contractsCount(n), FormatBRL(total)))
	}
	return strings.Join(parts, " "), ids
}

func contractsCount(n int) string {
	if n == 1 {
		return "1 contratação"
	}
	return fmt.Sprintf("%d contratações", n)
}

func SanctionedFinding(s SanctionedContract) Finding {
	parts := make([]string, 0, len(s.Sanctions))
	for _, x := range s.Sanctions {
		period := "sem data final"
		if x.EndsAt != nil {
			period = "até " + x.EndsAt.Format("02/01/2006")
		}
		start := ""
		if x.StartsAt != nil {
			start = " de " + x.StartsAt.Format("02/01/2006")
		}
		parts = append(parts, fmt.Sprintf("%s (%s %s), %s%s %s.", x.Category, x.Register, x.Code, x.Organ, start, period))
	}
	return Finding{
		Title: fmt.Sprintf("%s: contratação de %s publicada em %s durante sanção", supplierLabel(s.Profile),
			FormatBRL(contractValueCents(s.Contract)), s.Contract.First.Format("02/01/2006")),
		Detail:   strings.Join(parts, " "),
		ActIDs:   []string{s.Contract.ActID},
		Entities: cnpjMentions(s.Profile.CNPJ),
	}
}

func organsSuffix(organs []string) string {
	if len(organs) == 0 {
		return ""
	}
	return " (" + strings.Join(organs, ", ") + ")"
}
