import { Bill, CompanyResponse, EntityResponse, Organ, Phase, Suggestion } from "./api";
import { billPath } from "./bills";
import { entityPath } from "./entity";
import { formatCompactCents } from "./panels";
import { EMPTY_STATE, TYPE_OPTIONS, queryFromState } from "./searchState";
import { PHASES, PHASE_LABEL } from "./types";

export interface Ficha {
  kind: string;
  title: string;
  facts: string[];
  cta: string;
  href: string;
}

const MONTHS = ["jan.", "fev.", "mar.", "abr.", "maio", "jun.", "jul.", "ago.", "set.", "out.", "nov.", "dez."];
const MIN_ORGAN_NAME_MATCH = 8;
const MIN_LOCAL_MATCH = 2;
const PER_GROUP = 3;
const MAX_ORGANS_IN_FICHA = 3;

export function fold(s: string): string {
  return s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase().trim();
}

export function cnpjDigits(q: string): string | null {
  const t = q.trim();
  if (!/^[\d.\/\-\s]+$/.test(t)) return null;
  const d = t.replace(/\D/g, "");
  return d.length === 14 ? d : null;
}

export function looksLikeNumber(q: string): boolean {
  return /^(processo|contrato)?\s*(n\.?\s*[º°o]\.?\s*)?\d/i.test(q.trim());
}

export function exactMatch(q: string, items: Suggestion[]): Suggestion | null {
  const cnpj = cnpjDigits(q);
  if (cnpj) return items.find((s) => s.kind === "cnpj" && s.key === cnpj) ?? null;
  return items.find((s) => s.kind !== "cnpj") ?? null;
}

export function matchOrgan(q: string, organs: Organ[]): Organ | null {
  const f = fold(q);
  if (f.length < MIN_LOCAL_MATCH) return null;
  const exact = organs.find((o) => f === fold(o.acronym) || f === fold(o.name));
  if (exact) return exact;
  if (f.length < MIN_ORGAN_NAME_MATCH) return null;
  const partial = organs.filter((o) => fold(o.name).includes(f));
  return partial.length === 1 ? partial[0] : null;
}

function acts(n: number): string {
  return `${n.toLocaleString("pt-BR")} ${n === 1 ? "ato" : "atos"}`;
}

function year(iso: string): string {
  return iso.slice(0, 4);
}

function monthYear(iso: string): string {
  return `${MONTHS[Number(iso.slice(5, 7)) - 1]} ${year(iso)}`;
}

function span(dates: string[], format: (iso: string) => string): string {
  if (dates.length === 0) return "";
  const sorted = [...dates].sort();
  const first = format(sorted[0]);
  const last = format(sorted[sorted.length - 1]);
  return first === last ? `em ${first}` : `de ${first} a ${last}`;
}

export function companyFicha(s: Suggestion, c: CompanyResponse | null): Ficha {
  const facts: string[] = [];
  if (c?.registry?.status) facts.push(`Situação cadastral: ${c.registry.status.toLowerCase()}`);
  const when = c ? span(c.acts.map((a) => a.published_at), year) : "";
  facts.push(`${acts(s.acts)} nos Diários${when ? `, ${when}` : ""}`);
  if (c && c.total_value_cents > 0) facts.push(`${formatCompactCents(c.total_value_cents)} citados em atos`);
  const paid = c?.payments.reduce((sum, p) => sum + p.paid_cents, 0) ?? 0;
  if (paid > 0) facts.push(`${formatCompactCents(paid)} pagos (TCE-RJ)`);
  return {
    kind: `Empresa · CNPJ ${s.label}`,
    title: c?.registry?.name || s.name || s.label,
    facts,
    cta: "Ver a ficha da empresa →",
    href: `/empresa/${s.key}`,
  };
}

function phaseFlow(counts: Partial<Record<Phase, number>>): string {
  const present = PHASES.filter((p) => p !== "outro" && (counts[p] ?? 0) > 0);
  return present.map((p, i) => (i === 0 ? PHASE_LABEL[p] : PHASE_LABEL[p].toLowerCase())).join(" → ");
}

export function entityFicha(s: Suggestion, e: EntityResponse | null): Ficha {
  const facts: string[] = [];
  const when = e ? span(e.acts.map((a) => a.published_at), monthYear) : "";
  facts.push(`${acts(e?.total_acts ?? s.acts)}${when ? `, ${when}` : ""}`);
  const organs = (e?.organs ?? []).map((o) => o.organ).filter(Boolean).slice(0, MAX_ORGANS_IN_FICHA);
  if (organs.length > 0) facts.push(organs.join(" e "));
  const flow = e ? phaseFlow(e.count_by_phase) : "";
  if (flow) facts.push(flow);
  const isProcess = s.kind === "processo";
  return {
    kind: isProcess ? "Processo administrativo" : "Contrato",
    title: s.label,
    facts,
    cta: "Ver a linha do tempo →",
    href: entityPath(isProcess ? "processo" : "contrato", s.label.replace(/\//g, "-")),
  };
}

export function organFicha(o: Organ): Ficha {
  return {
    kind: "Órgão da Prefeitura",
    title: o.name ? `${o.acronym} · ${o.name}` : o.acronym,
    facts: [`${acts(o.acts)} nos Diários`],
    cta: "Filtrar a busca por este órgão →",
    href: "/" + queryFromState({ ...EMPTY_STATE, organ: o.acronym }),
  };
}

export interface SuggestItem {
  label: string;
  meta: string;
  href: string;
}

export interface SuggestGroup {
  title: string;
  items: SuggestItem[];
}

function suggestionItem(s: Suggestion): SuggestItem {
  if (s.kind === "cnpj") return { label: s.name || s.label, meta: `${s.label} · ${acts(s.acts)}`, href: `/empresa/${s.key}` };
  const kind = s.kind === "processo" ? "Processo" : "Contrato";
  return { label: `${kind} ${s.label}`, meta: acts(s.acts), href: entityPath(s.kind, s.label.replace(/\//g, "-")) };
}

export function suggestGroups(q: string, remote: Suggestion[], organs: Organ[], bills: Bill[]): SuggestGroup[] {
  const f = fold(q);
  if (f.length < MIN_LOCAL_MATCH) return [];
  const local = (text: string) => fold(text).includes(f);
  const groups: SuggestGroup[] = [
    { title: "Empresas", items: remote.filter((s) => s.kind === "cnpj").map(suggestionItem) },
    { title: "Processos e contratos", items: remote.filter((s) => s.kind !== "cnpj").map(suggestionItem) },
    {
      title: "Órgãos",
      items: organs.filter((o) => local(`${o.acronym} ${o.name}`)).map((o) => ({
        label: o.name ? `${o.acronym} · ${o.name}` : o.acronym,
        meta: acts(o.acts),
        href: "/" + queryFromState({ ...EMPTY_STATE, organ: o.acronym }),
      })),
    },
    {
      title: "Tipos de ato",
      items: TYPE_OPTIONS.filter((t) => t.value && local(t.label)).map((t) => ({
        label: t.label,
        meta: "",
        href: "/" + queryFromState({ ...EMPTY_STATE, type: t.value }),
      })),
    },
    { title: "Proposições", items: bills.map((b) => ({ label: b.document, meta: b.summary, href: billPath(b.process) })) },
  ];
  return groups.map((g) => ({ ...g, items: g.items.slice(0, PER_GROUP) })).filter((g) => g.items.length > 0);
}

export function moveFocus(current: number, delta: number, count: number): number {
  if (count === 0) return -1;
  return Math.max(-1, Math.min(count - 1, current + delta));
}

export function suggestionsStatus(groups: SuggestGroup[]): string {
  const n = groups.reduce((sum, g) => sum + g.items.length, 0);
  if (n === 0) return "";
  return `${n} ${n === 1 ? "sugestão" : "sugestões"}. Use a seta para baixo para chegar a elas.`;
}
