package mcp

import (
	"fmt"
	"time"

	"github.com/seu-usuario/diario-sg/services/api/internal/core/domain"
)

type registryPartnerDTO struct {
	Tipo         string `json:"tipo"`
	Nome         string `json:"nome"`
	Documento    string `json:"documento"`
	Qualificacao string `json:"qualificacao"`
	Desde        string `json:"desde,omitempty"`
}

type registryDTO struct {
	MesReferencia      string               `json:"mes_referencia"`
	RazaoSocial        string               `json:"razao_social"`
	NomeFantasia       string               `json:"nome_fantasia,omitempty"`
	NaturezaJuridica   string               `json:"natureza_juridica"`
	CapitalSocial      int64                `json:"capital_social_centavos"`
	Porte              string               `json:"porte"`
	Situacao           string               `json:"situacao"`
	SituacaoDesde      string               `json:"situacao_desde,omitempty"`
	MotivoSituacao     string               `json:"motivo_situacao,omitempty"`
	Abertura           string               `json:"abertura,omitempty"`
	AtividadePrincipal string               `json:"atividade_principal"`
	Endereco           string               `json:"endereco"`
	Socios             []registryPartnerDTO `json:"socios"`
}

var partnerKindNames = map[domain.PartnerKind]string{
	domain.PartnerCompany: "pessoa jurídica", domain.PartnerPerson: "pessoa física", domain.PartnerForeign: "estrangeiro",
}

func optionalDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateOnly)
}

func registryOf(r *domain.CompanyRegistry) *registryDTO {
	if r == nil {
		return nil
	}
	e := r.Establishment
	out := &registryDTO{MesReferencia: r.Month.Format("2006-01"), RazaoSocial: r.Company.Name, NomeFantasia: e.TradeName,
		NaturezaJuridica: r.Company.LegalNature, CapitalSocial: r.Company.CapitalCents, Porte: r.Company.Size,
		Situacao: e.Status, SituacaoDesde: optionalDate(e.StatusSince), Abertura: optionalDate(e.OpenedAt),
		AtividadePrincipal: fmt.Sprintf("%s %s", e.MainActivity.Code, e.MainActivity.Description),
		Endereco:           fmt.Sprintf("%s, %s %s, %s, %s/%s, CEP %s", e.Street, e.Number, e.Complement, e.District, e.City, e.UF, e.ZIP),
		Socios:             make([]registryPartnerDTO, len(r.Partners))}
	if e.Status != "Ativa" {
		out.MotivoSituacao = e.StatusReason
	}
	for i, p := range r.Partners {
		out.Socios[i] = registryPartnerDTO{Tipo: partnerKindNames[p.Kind], Nome: p.Name, Documento: p.Document, Qualificacao: p.Role, Desde: optionalDate(p.Since)}
	}
	return out
}

func registryAbsence(kind domain.EntityKind, r *domain.CompanyRegistry, month *time.Time) string {
	if kind != domain.EntityCNPJ || r != nil || month == nil {
		return ""
	}
	return "CNPJ não encontrado no cadastro da Receita de " + month.Format("01/2006") + " (pode ser erro de digitação no Diário)."
}
