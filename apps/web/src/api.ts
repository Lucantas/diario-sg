
export type ActType =
  | "nomeacao" | "exoneracao" | "contrato" | "aditivo" | "licitacao"
  | "dispensa" | "decreto" | "lei" | "portaria" | "resolucao"
  | "despacho" | "edital" | "ata" | "corrigenda" | "prestacao_contas" | "outro";

export type Source = "diario_prefeitura" | "diario_camara";

export type Phase =
  | "licitacao" | "homologacao" | "ata_registro_precos" | "dispensa" | "contrato"
  | "fiscal" | "aditivo" | "ajuste_contas" | "rescisao" | "outro";

export type EntityKind = "processo" | "contrato";

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

export interface CompanyResponse {
  cnpj: string;
  total_value_cents: number;
  count_by_type: Partial<Record<ActType, number>>;
  acts: ActHit[];
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
  return request<{ query: string; status: string }>("/v1/subscriptions", {
    method: "POST",
    body: JSON.stringify({ email, query }),
  });
}

export function confirmSubscription(token: string) {
  return request<{ query: string }>("/v1/subscriptions/confirm", {
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
