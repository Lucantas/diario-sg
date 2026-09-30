package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	minDistinctiveNameWords = 3
	maxAppointmentActs      = 5
)

var agentRoleNames = map[string]string{
	RolePrefeito: "Prefeito", RoleVicePrefeito: "Vice-prefeito", RoleSecretario: "Secretário municipal",
	RoleProcuradorGeral: "Procurador-Geral do Município", RoleVereador: "Vereador",
}

func AgentRoleName(role string) string {
	if name, ok := agentRoleNames[role]; ok {
		return name
	}
	return role
}

var nameParticles = map[string]bool{"DA": true, "DE": true, "DO": true, "DAS": true, "DOS": true, "E": true}

type PublicAgentName struct {
	Name   string
	Role   string
	Office string
}

type PartnerAppointment struct {
	Name        string
	ActID       string
	PublishedAt time.Time
}

type PartnerPublicAgent struct {
	Profile           SupplierProfile
	Agents            []PublicAgentName
	AppointedPartners int
	AppointmentActIDs []string
	Appointments      int
}

func IsDistinctiveName(name string) bool {
	words := 0
	for _, w := range strings.Fields(foldText(name)) {
		if !nameParticles[w] {
			words++
		}
	}
	return words >= minDistinctiveNameWords
}

func FindPartnerPublicAgents(cnpjs []string, profiles map[string]SupplierProfile, agents []PublicAgentName,
	appointments []PartnerAppointment) []PartnerPublicAgent {
	agentByName := map[string]PublicAgentName{}
	for _, a := range agents {
		if key := foldText(a.Name); IsDistinctiveName(a.Name) {
			if _, seen := agentByName[key]; !seen {
				agentByName[key] = a
			}
		}
	}
	actsByName := map[string][]string{}
	for _, a := range appointments {
		key := foldText(a.Name)
		actsByName[key] = append(actsByName[key], a.ActID)
	}
	var out []PartnerPublicAgent
	for _, p := range companiesByBase(cnpjs, profiles) {
		found := PartnerPublicAgent{Profile: p}
		for _, partner := range p.Partners {
			if partner.Kind != PartnerPerson || !IsDistinctiveName(partner.Name) {
				continue
			}
			key := foldText(partner.Name)
			if a, ok := agentByName[key]; ok {
				found.Agents = append(found.Agents, a)
			}
			if acts := actsByName[key]; len(acts) > 0 {
				found.AppointedPartners++
				found.Appointments += len(acts)
				found.AppointmentActIDs = append(found.AppointmentActIDs, acts...)
			}
		}
		found.AppointmentActIDs = found.AppointmentActIDs[:min(len(found.AppointmentActIDs), maxAppointmentActs)]
		if len(found.Agents) > 0 || found.Appointments > 0 {
			out = append(out, found)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (len(out[i].Agents) > 0) != (len(out[j].Agents) > 0) {
			return len(out[i].Agents) > 0
		}
		return out[i].Profile.CNPJ < out[j].Profile.CNPJ
	})
	return out
}

func companiesByBase(cnpjs []string, profiles map[string]SupplierProfile) []SupplierProfile {
	byBase := map[string]SupplierProfile{}
	for _, cnpj := range cnpjs {
		p, ok := profiles[cnpj]
		if !ok {
			continue
		}
		base := CNPJBase(cnpj)
		if prev, seen := byBase[base]; !seen || cnpj < prev.CNPJ {
			byBase[base] = p
		}
	}
	out := make([]SupplierProfile, 0, len(byBase))
	for _, p := range byBase {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CNPJ < out[j].CNPJ })
	return out
}

func partnerAgentPatterns() map[PatternID]Pattern {
	return map[PatternID]Pattern{
		PatternPartnerPublicAgent: {
			ID:    PatternPartnerPublicAgent,
			Title: "Sócio com o nome de agente público",
			Rule: fmt.Sprintf("Sócio pessoa física de empresa citada no Diário em contrato, aditivo, dispensa, licitação ou licença "+
				"ambiental, com nome de pelo menos %d palavras além de "+
				"da, de, do, das, dos e e, igual (sem diferença de acento) ao de um agente político da folha da Prefeitura ou da "+
				"Câmara, ou a um nome citado em ato de nomeação ou exoneração do Diário.", minDistinctiveNameWords),
			Caveat: "Ligação possível, só pelo nome: homônimos são comuns e a Receita não publica o CPF inteiro do sócio. Quem " +
				"aparece só em nomeação ou exoneração não é nomeado aqui (ADR 0006): o nome está nos atos. Parentesco não entra, " +
				"porque nenhuma base aberta diz quem é parente de quem. " + registrySourceNote,
		},
	}
}

func PartnerPublicAgentFinding(p PartnerPublicAgent) Finding {
	detail := "Ligação possível, só pelo nome: pode ser homônimo."
	if p.Appointments > 0 {
		detail += fmt.Sprintf(" %s cita%s %s.", appointmentsCount(p.Appointments), pluralSuffix(p.Appointments), appointedNames(p.AppointedPartners))
	}
	if p.Appointments > maxAppointmentActs {
		detail += fmt.Sprintf(" Abaixo, %d deles.", maxAppointmentActs)
	}
	return Finding{Title: supplierLabel(p.Profile) + ": " + partnerAgentSubject(p), Detail: detail, ActIDs: p.AppointmentActIDs,
		Entities: cnpjMentions(p.Profile.CNPJ)}
}

func partnerAgentSubject(p PartnerPublicAgent) string {
	if len(p.Agents) > 0 {
		names := make([]string, len(p.Agents))
		for i, a := range p.Agents {
			names[i] = a.Name + ", " + AgentRoleName(a.Role)
			if a.Office != "" {
				names[i] += " (" + a.Office + ")"
			}
		}
		return "sócio com o nome de " + joinPortuguese(names)
	}
	if p.AppointedPartners > 1 {
		return fmt.Sprintf("%d sócios com nomes de pessoas nomeadas ou exoneradas no Diário", p.AppointedPartners)
	}
	return "sócio com o nome de pessoa nomeada ou exonerada no Diário"
}

func appointedNames(partners int) string {
	if partners > 1 {
		return fmt.Sprintf("os nomes de %d sócios", partners)
	}
	return "o nome"
}

func appointmentsCount(n int) string {
	if n == 1 {
		return "1 ato de nomeação ou exoneração"
	}
	return fmt.Sprintf("%d atos de nomeação ou exoneração", n)
}

func pluralSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "m"
}
