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
	Agent             *PublicAgentName
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
		for _, partner := range p.Partners {
			key := foldText(partner.Name)
			if partner.Kind != PartnerPerson || !IsDistinctiveName(partner.Name) {
				continue
			}
			found := PartnerPublicAgent{Profile: p, Appointments: len(actsByName[key])}
			if a, ok := agentByName[key]; ok {
				found.Agent = &a
			}
			found.AppointmentActIDs = actsByName[key][:min(len(actsByName[key]), maxAppointmentActs)]
			if found.Agent != nil || found.Appointments > 0 {
				out = append(out, found)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Agent != nil) != (out[j].Agent != nil) {
			return out[i].Agent != nil
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
			Rule: fmt.Sprintf("Sócio pessoa física de empresa contratada ou licenciada, com nome de pelo menos %d palavras além de "+
				"da, de, do, das, dos e e, igual (sem diferença de acento) ao de um agente político da folha da Prefeitura ou da "+
				"Câmara, ou a um nome citado em ato de nomeação ou exoneração do Diário.", minDistinctiveNameWords),
			Caveat: "Ligação possível, só pelo nome: homônimos são comuns e a Receita não publica o CPF inteiro do sócio. Quem " +
				"aparece só em nomeação ou exoneração não é nomeado aqui (ADR 0006): o nome está nos atos. Parentesco não entra, " +
				"porque nenhuma base aberta diz quem é parente de quem. " + registrySourceNote,
		},
	}
}

func PartnerPublicAgentFinding(p PartnerPublicAgent) Finding {
	title := fmt.Sprintf("%s: sócio com o nome de pessoa nomeada ou exonerada no Diário", supplierLabel(p.Profile))
	if p.Agent != nil {
		title = fmt.Sprintf("%s: sócio com o nome de %s, %s", supplierLabel(p.Profile), p.Agent.Name, AgentRoleName(p.Agent.Role))
		if p.Agent.Office != "" {
			title += " (" + p.Agent.Office + ")"
		}
	}
	detail := "Ligação possível, só pelo nome: pode ser homônimo."
	if p.Appointments > 0 {
		detail += fmt.Sprintf(" %s cita%s o nome.", appointmentsCount(p.Appointments), pluralSuffix(p.Appointments))
	}
	if p.Appointments > maxAppointmentActs {
		detail += fmt.Sprintf(" Abaixo, os %d mais recentes.", maxAppointmentActs)
	}
	return Finding{Title: title, Detail: detail, ActIDs: p.AppointmentActIDs, Entities: cnpjMentions(p.Profile.CNPJ)}
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
