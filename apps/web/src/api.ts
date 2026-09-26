
export type ActType =
  | "nomeacao" | "exoneracao" | "contrato" | "aditivo" | "licitacao"
  | "dispensa" | "decreto" | "lei" | "portaria" | "resolucao"
  | "despacho" | "edital" | "ata" | "corrigenda" | "prestacao_contas" | "outro";

export type Source = "diario_prefeitura" | "diario_camara";

export type Phase =
  | "licitacao" | "homologacao" | "ata_registro_precos" | "dispensa" | "contrato"
  | "fiscal" | "aditivo" | "ajuste_contas" | "rescisao" | "outro";

export type EntityKind = "processo" | "contrato";

export type AlertEntityKind = EntityKind | "cnpj";

export interface Subscription {
  query: string;
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

export interface Sanction {
  register: "CEIS" | "CNEP";
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
}

export function getFederal() {
  return request<FederalReport>("/v1/federal");
}

export interface CompanyResponse {
  cnpj: string;
  total_value_cents: number;
  count_by_type: Partial<Record<ActType, number>>;
  acts: ActHit[];
  registry: Registry | null;
  registry_month: string | null;
  sanctions: Sanction[];
  sanctions_listed_on: Partial<Record<"CEIS" | "CNEP", string>>;
  payments: PaymentYear[];
  payments_coverage: { from: string; to: string } | null;
  pncp_contracts: PNCPContract[];
  stalled_works: StalledWork[];
  amendment_payments: AmendmentPayment[];
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

export function searchActs(params: URLSearchParams) {
  return request<SearchResponse>(`/v1/acts?${params}`);
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

export function listOrgans() {
  return request<{ items: Organ[] }>("/v1/organs");
}

export function getCompany(cnpj: string) {
  return request<CompanyResponse>(`/v1/entities/cnpj/${encodeURIComponent(cnpj)}`);
}

export function getEntity(kind: EntityKind, slug: string) {
  return request<EntityResponse>(`/v1/entities/${kind}/${encodeURIComponent(slug)}`);
}

export function subscribe(email: string, query: string) {
  return request<Subscription>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, query }),
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
