
export type ActType =
  | "nomeacao" | "exoneracao" | "contrato" | "aditivo" | "licitacao"
  | "dispensa" | "decreto" | "lei" | "portaria" | "resolucao"
  | "despacho" | "edital" | "ata" | "corrigenda" | "prestacao_contas" | "licenca_ambiental" | "outro";

export type Source = "diario_prefeitura" | "diario_camara";

export type Phase =
  | "licitacao" | "homologacao" | "ata_registro_precos" | "dispensa" | "contrato"
  | "fiscal" | "aditivo" | "ajuste_contas" | "rescisao" | "outro";

export type EntityKind = "processo" | "contrato";

export type AlertEntityKind = EntityKind | "cnpj";

export interface AlertFilters {
  source: Source | "";
  type: ActType | "";
  organ: string;
  theme: string;
}

export interface Subscription {
  query: string;
  filters: AlertFilters;
  subject: string;
  status: string;
}

export interface Mention {
  kind: EntityKind;
  key: string;
  label: string;
  slug: string;
}

export interface ActHit {
  id: string;
  gazette_id: string;
  source: Source;
  source_name: string;
  position: number;
  type: ActType;
  title: string;
  organ: string;
  organ_name: string;
  snippet: string;
  edition_number: string;
  published_at: string;
  is_extra: boolean;
  source_url: string;
  page_start: number | null;
  page_end: number | null;
  pdf_sha256: string;
  cnpjs: string[];
  values_cents: number[];
  warnings: string[];
  mentions: Mention[];
  phase?: Phase;
}

export interface SearchResponse {
  items: ActHit[];
  total: number;
  limit: number;
  offset: number;
}

export interface Organ {
  acronym: string;
  name: string;
  acts: number;
}

export interface RegistryActivity {
  code: string;
  description: string;
}

export interface RegistryPartner {
  kind: "pessoa_juridica" | "pessoa_fisica" | "estrangeiro";
  name: string;
  document: string;
  role: string;
  since: string | null;
}

export interface Registry {
  month: string;
  name: string;
  trade_name: string;
  legal_nature: string;
  capital_cents: number;
  size: string;
  headquarters: boolean;
  status: string;
  status_since: string | null;
  status_reason: string;
  opened_at: string | null;
  main_activity: RegistryActivity;
  other_activities: RegistryActivity[];
  street: string;
  number: string;
  complement: string;
  district: string;
  zip: string;
  city: string;
  uf: string;
  partners: RegistryPartner[];
}

export type SanctionState = "no_cadastro" | "prazo_encerrado" | "fora_do_cadastro";

export type SanctionRegister = "CEIS" | "CNEP" | "CEPIM";

export interface Sanction {
  register: SanctionRegister;
  code: string;
  cnpj: string;
  name: string;
  category: string;
  starts_at: string | null;
  ends_at: string | null;
  published_at: string | null;
  process: string;
  organ: string;
  organ_uf: string;
  sphere: string;
  scope: string;
  legal_basis: string;
  fine_cents: number | null;
  first_seen: string;
  last_seen: string;
  state: SanctionState;
}

export interface PaymentYear {
  year: number;
  units: string[];
  committed_cents: number;
  liquidated_cents: number;
  paid_cents: number;
}

export interface PNCPContract {
  control_number: string;
  url: string;
  org_cnpj: string;
  unit: string;
  kind: string;
  number: string;
  process: string;
  object: string;
  value_cents: number;
  signed_at: string | null;
  starts_at: string | null;
  ends_at: string | null;
}

export interface StalledWork {
  contract: string;
  cnpj: string;
  contractor: string;
  organ: string;
  function: string;
  total_cents: number;
  paid_cents: number;
  stalled_at: string | null;
  started_at: string | null;
  stalled_for: string;
  reason: string;
  contract_status: string;
  funding: string;
}

export interface TCEOversight {
  accounts: { year: number; opinion: string; process: string; responsible: string }[];
  penalties: {
    process: string;
    search: string;
    organs: string[];
    natures: string[];
    total_cents: number;
    last_session: string | null;
    condemnations: { condemnation: string; year: number; value_cents: number; organ: string; session_date: string | null }[];
  }[];
  works: StalledWork[];
  fiscal_control: FiscalControl[];
}

export interface FiscalControl {
  year: number;
  period: number;
  committed_cents: number;
  liquidated_cents: number;
  paid_cents: number;
  source_url: string;
  tce_committed_cents: number;
  tce_paid_cents: number;
  tce_loaded: boolean;
  paid_coverage_bp: number;
  low_coverage: boolean;
  portal_paid_cents: number | null;
}

export function getOversight() {
  return request<TCEOversight>("/v1/tce");
}

export interface AmendmentPayment {
  code: string;
  author: string;
  kind: string;
  month: string;
  cnpj: string;
  name: string;
  value_cents: number;
}

export interface FederalReport {
  transfers_coverage: { from: string; to: string } | null;
  transfers: { year: number; kind: string; function: string; value_cents: number }[];
  amendments: {
    code: string;
    year: number;
    kind: string;
    author: string;
    function: string;
    action: string;
    committed_cents: number;
    liquidated_cents: number;
    paid_cents: number;
  }[];
  favored: { cnpj: string; name: string; payments: number; value_cents: number; authors: string[]; first: string; last: string }[];
  special_transfers: SpecialTransfer[];
}

export interface SpecialTransfer {
  plan_id: number;
  code: string;
  year: number;
  status: string;
  author: string;
  amendment: string;
  area: string;
  value_cents: number;
  executors: { cnpj: string; name: string; object: string; value_cents: number }[];
  committed_cents: number;
  paid_cents: number;
  last_paid_at: string | null;
  work_plan_status: string;
  execution_end: string | null;
  report_kind: string;
  report_at: string | null;
  executed_cents: number;
  pending_cents: number;
}

export function getFederal() {
  return request<FederalReport>("/v1/federal");
}

export type AgentRole = "prefeito" | "vice_prefeito" | "secretario" | "procurador_geral" | "vereador";

export interface PoliticalAgent {
  body: "prefeitura" | "camara";
  role: AgentRole;
  name: string;
  offices: string[];
  party?: string;
  parliamentary_name?: string;
  first: string;
  last: string;
  months: { month: string; office: string; gross_cents: number; discount_cents: number | null; net_cents: number | null }[];
}

export interface PoliticalAgentsReport {
  agents: PoliticalAgent[];
  norms: { role: AgentRole; from_year: number; to_year: number; value_cents: number; norm: string; diario: string; search: string }[];
  coverage: { body: "prefeitura" | "camara"; from: string; to: string }[];
}

export function getPoliticalAgents() {
  return request<PoliticalAgentsReport>("/v1/agentes");
}

export interface CompanyResponse {
  cnpj: string;
  total_value_cents: number;
  count_by_type: Partial<Record<ActType, number>>;
  acts: ActHit[];
  registry: Registry | null;
  registry_month: string | null;
  sanctions: Sanction[];
  sanctions_listed_on: Partial<Record<SanctionRegister, string>>;
  payments: PaymentYear[];
  payments_coverage: { from: string; to: string } | null;
  pncp_contracts: PNCPContract[];
  stalled_works: StalledWork[];
  amendment_payments: AmendmentPayment[];
  municipal_commitments: MunicipalSupplier;
  procurements: MuralMatches;
  diario_sanctions: { kind: DiarioSanctionKind; act: ActHit }[];
}

export type DiarioSanctionKind = "advertencia" | "multa" | "suspensao" | "impedimento" | "inidoneidade";

export interface MuralMatches {
  procurements: {
    list: string; id: number; notice: string; process: string; modality: string; criterion: string; opens_at: string | null;
    object: string; status: string; url: string;
  }[];
  contracts: {
    procurement_id: number; notice: string; process: string; modality: string; object: string; value_cents: number; supplier: string;
    instrument: string; document_url: string;
  }[];
}

export interface MunicipalCommitment {
  entity: string;
  year: number;
  number: string;
  date: string;
  object: string;
  process_kind: string;
  process: string;
  modality: string;
  committed_cents: number;
  liquidated_cents: number;
  paid_cents: number;
}

export interface MunicipalSupplier {
  years: { year: number; commitments: number; committed_cents: number; paid_cents: number }[];
  recent: MunicipalCommitment[];
}

export interface OrganCount {
  organ: string;
  organ_name: string;
  acts: number;
}

export interface Related {
  kind: EntityKind | "cnpj";
  key: string;
  label: string;
  slug: string;
  acts: number;
}

export interface EntityResponse {
  kind: EntityKind;
  key: string;
  label: string;
  certainty: string;
  diarios: number;
  total_acts: number;
  count_by_phase: Partial<Record<Phase, number>>;
  organs: OrganCount[];
  related: Related[];
  warnings: string[];
  acts: ActHit[];
}

export interface PatternSearch {
  type: ActType;
  from: string;
  to: string;
  source: Source;
}

export interface Finding {
  title: string;
  detail: string;
  acts: ActHit[];
  search: PatternSearch | null;
  link: { label: string; url: string } | null;
}

export interface Pattern {
  id: string;
  title: string;
  rule: string;
  caveat: string;
  findings: Finding[];
}

export interface PanelAmounts {
  contracts: number;
  contracted_cents: number;
  registered_cents: number;
  amended_cents: number;
  paid_cents: number;
}

export interface SupplierRow extends PanelAmounts {
  cnpj: string;
  name: string;
  first: string;
  last: string;
  organs: string[];
  largest: ActHit | null;
}

export interface SupplierPanel extends PanelAmounts {
  source: Source;
  year: number | null;
  organ: string;
  organ_name: string;
  suppliers: number;
  items: SupplierRow[];
  years: (PanelAmounts & { year: number })[];
  organs: (PanelAmounts & { organ: string; organ_name: string })[];
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error ?? `Erro ${res.status}`);
  return body as T;
}

export function searchActs(params: URLSearchParams, signal?: AbortSignal) {
  return request<SearchResponse>(`/v1/acts?${params}`, { signal });
}

export interface PatternHighlightResponse {
  pattern_id: string;
  pattern_title: string;
  title: string;
  detail: string;
  findings: number;
}

export function listPatternHighlights(limit: number) {
  return request<{ items: PatternHighlightResponse[] }>(`/v1/patterns/highlights?limit=${limit}`);
}

export function listPatterns() {
  return request<{ items: Pattern[] }>("/v1/patterns");
}

export function getSupplierPanel(path: string) {
  return request<SupplierPanel>(path);
}

export interface StaffTotal {
  headcount: number;
  remuneration_cents: number;
}

export interface StaffMonth extends StaffTotal {
  month: string;
  groups: StaffTotal[];
  appointments: number;
  dismissals: number;
}

export interface StaffPanel {
  units: string[];
  unit: string;
  diario_source: Source;
  groups: { group: string; label: string }[];
  months: StaffMonth[];
}

export function getStaffPanel(path: string) {
  return request<StaffPanel>(path);
}

export interface TypeTotal {
  type: ActType;
  acts: number;
  value_cents: number;
}

export interface LatestEdition {
  gazette_id: string;
  source: Source;
  source_name: string;
  published_at: string;
  edition_number: string;
  is_extra: boolean;
  total_acts: number;
  types: TypeTotal[];
}

export function listLatestEditions() {
  return request<{ items: LatestEdition[] }>("/v1/gazettes/latest");
}

export type SuggestionKind = "cnpj" | "processo" | "contrato";

export interface Suggestion {
  kind: SuggestionKind;
  key: string;
  label: string;
  name: string;
  acts: number;
}

export function suggest(q: string, signal?: AbortSignal) {
  return request<{ items: Suggestion[] }>(`/v1/suggest?${new URLSearchParams({ q })}`, { signal });
}

export function listOrgans() {
  return request<{ items: Organ[] }>("/v1/organs");
}

export function getCompany(cnpj: string, signal?: AbortSignal) {
  return request<CompanyResponse>(`/v1/entities/cnpj/${encodeURIComponent(cnpj)}`, { signal });
}

export function getEntity(kind: EntityKind, slug: string, signal?: AbortSignal) {
  return request<EntityResponse>(`/v1/entities/${kind}/${encodeURIComponent(slug)}`, { signal });
}

export function subscribe(email: string, query: string, filters: AlertFilters) {
  return request<Subscription>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, query, filters }),
  });
}

export function subscribeEntity(email: string, kind: AlertEntityKind, value: string) {
  return request<Subscription>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, entity: { kind, value } }),
  });
}

export function confirmSubscription(token: string) {
  return request<Subscription>("/v1/subscriptions/confirm", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export function unsubscribe(token: string) {
  return request<unknown>("/v1/subscriptions/unsubscribe", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export type ReportKind = "texto_errado" | "tipo_errado" | "orgao_errado" | "pagina_errada" | "outro";

export interface ErrorReport {
  gazette_id: string;
  position: number;
  act_title: string;
  kind: ReportKind;
  message: string;
  website: string;
}

export function reportError(report: ErrorReport) {
  return request<{ status: string }>("/v1/reports", {
    method: "POST",
    body: JSON.stringify(report),
  });
}

export interface IssuedKey {
  key: string;
  prefix: string;
  mcp_url: string;
}

export function issueMcpKey(website: string) {
  return request<IssuedKey>("/v1/mcp/keys", {
    method: "POST",
    body: JSON.stringify({ website }),
  });
}

export async function revokeMcpKey(key: string) {
  const res = await fetch("/api/v1/mcp/keys", {
    method: "DELETE",
    headers: { Authorization: `Bearer ${key.trim()}` },
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Erro ${res.status}`);
  }
}

export type BillPhase =
  | "virou_norma" | "vetado" | "rejeitado" | "retirado" | "arquivado"
  | "enviado_ao_executivo" | "aprovado" | "em_votacao" | "em_comissao" | "apresentado";

export interface BillLaw {
  kind: string;
  kind_name: string;
  number: string;
  summary: string;
  url: string;
  certainty: "exata" | "forte" | "fraca";
  diario_search: string;
}

export interface Bill {
  process: string;
  kind: string;
  document: string;
  summary: string;
  authors: string;
  presented_on: string | null;
  status: string;
  phase: BillPhase;
  days_idle: number;
  current_body: string;
  last_movement: string;
  url: string;
  fetched_at: string;
  laws: BillLaw[];
}

export interface BillDetail extends Bill {
  events: { at: string; label: string; text: string; sector?: string }[];
  opinions: { result: string; on: string | null; committee: string; rapporteur: string }[];
}

export interface BillsResponse {
  total: number;
  by_phase: Partial<Record<BillPhase, number>>;
  theme_rule: string;
  items: Bill[];
}

export function listBills(params: URLSearchParams, signal?: AbortSignal) {
  return request<BillsResponse>(`/v1/bills?${params}`, { signal });
}

export function getBill(process: string) {
  return request<BillDetail>(`/v1/bills/${process}`);
}
